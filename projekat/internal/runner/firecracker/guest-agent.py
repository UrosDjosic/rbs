#!/usr/bin/env python3
"""
Firecracker Guest Agent

Runs inside the Firecracker VM to execute functions and communicate with the host.
Listens on vsock and receives invocation requests from the host runner.

To use:
1. Place this script inside the VM rootfs
2. Add to systemd/init to run on boot
3. Ensure Python 3 and socket module are available

The host will send JSON on the vsock socket with:
{
    "function_id": "fn-xxx",
    "version_id": "v1-xxx", 
    "payload": "..."
}

The agent will respond with:
{
    "exit_code": 0,
    "stdout": "...",
    "stderr": "..."
}
"""

import socket
import json
import subprocess
import sys
import os
import logging
from pathlib import Path

# Setup logging (to syslog or file in guest)
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[
        logging.FileHandler('/var/log/function-agent.log'),
        logging.StreamHandler(sys.stderr),
    ]
)
logger = logging.getLogger(__name__)

# Constants
VSOCK_PORT = 8
VSOCK_CID = socket.VMADDR_CID_ANY
FUNCTION_MOUNT_PATH = "/function"
FUNCTION_SCRIPT = "main.py"
EXECUTION_TIMEOUT = 30
FUNCTION_BLOCK_DEVICE = "/dev/vdb"


def create_vsock_server():
    """Create a listening socket on AF_VSOCK."""
    sock = socket.socket(socket.AF_VSOCK, socket.SOCK_STREAM)
    try:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    except (AttributeError, OSError):
        pass
    sock.bind((VSOCK_CID, VSOCK_PORT))
    sock.listen(128)
    logger.info(f"Listening on vsock://{VSOCK_CID}:{VSOCK_PORT}")
    return sock


def ensure_function_mount():
    """Mount the function block device if /function/main.py is not visible yet."""
    mount_path = Path(FUNCTION_MOUNT_PATH)
    script_path = mount_path / FUNCTION_SCRIPT
    mount_path.mkdir(parents=True, exist_ok=True)

    if script_path.exists():
        return

    if not Path(FUNCTION_BLOCK_DEVICE).exists():
        logger.warning(f"Function block device not found: {FUNCTION_BLOCK_DEVICE}")
        return

    try:
        subprocess.run(
            ["mount", "-o", "ro", FUNCTION_BLOCK_DEVICE, FUNCTION_MOUNT_PATH],
            check=True,
            capture_output=True,
            timeout=10,
        )
        logger.info(f"Mounted {FUNCTION_BLOCK_DEVICE} at {FUNCTION_MOUNT_PATH}")
    except subprocess.CalledProcessError as e:
        stderr = e.stderr.decode("utf-8", errors="replace")
        logger.error(f"Function mount failed: {stderr}")
    except Exception as e:
        logger.error(f"Function mount error: {e}")


def execute_function(fn_id: str, version_id: str, payload: str) -> dict:
    """
    Execute a function and return the result.
    
    Args:
        fn_id: Function ID
        version_id: Version ID
        payload: Input payload (usually JSON)
    
    Returns:
        Dict with exit_code, stdout, stderr
    """
    logger.info(f"Executing {fn_id}@{version_id}")
    
    # Build script path
    script_path = Path(FUNCTION_MOUNT_PATH) / FUNCTION_SCRIPT
    
    if not script_path.exists():
        logger.error(f"Script not found: {script_path}")
        return {
            "exit_code": 127,
            "stdout": "",
            "stderr": f"Function script not found: {script_path}",
        }
    
    try:
        # Execute the function script
        result = subprocess.run(
            [sys.executable, str(script_path)],
            input=payload.encode('utf-8'),
            capture_output=True,
            timeout=EXECUTION_TIMEOUT,
            cwd=str(Path(FUNCTION_MOUNT_PATH)),
        )
        
        logger.info(
            f"Function completed: exit_code={result.returncode}, "
            f"stdout_len={len(result.stdout)}, stderr_len={len(result.stderr)}"
        )
        
        return {
            "exit_code": result.returncode,
            "stdout": result.stdout.decode('utf-8', errors='replace'),
            "stderr": result.stderr.decode('utf-8', errors='replace'),
        }
    
    except subprocess.TimeoutExpired:
        logger.error(f"Function timeout after {EXECUTION_TIMEOUT}s")
        return {
            "exit_code": 124,
            "stdout": "",
            "stderr": f"Function execution timeout after {EXECUTION_TIMEOUT} seconds",
        }
    
    except Exception as e:
        logger.error(f"Function execution error: {e}")
        return {
            "exit_code": 1,
            "stdout": "",
            "stderr": f"Execution error: {str(e)}",
        }


def handle_client(conn: socket.socket, addr):
    """Handle a single client connection."""
    logger.info(f"Client connected: {addr}")
    
    try:
        # Read request (one line of JSON)
        data = b""
        while not data.endswith(b'\n'):
            chunk = conn.recv(4096)
            if not chunk:
                logger.warning("Client disconnected without sending request")
                return
            data += chunk
        
        request_json = data.decode('utf-8').strip()
        request = json.loads(request_json)
        
        # Validate request
        fn_id = request.get("function_id")
        version_id = request.get("version_id")
        payload = request.get("payload", "")
        
        if not fn_id or not version_id:
            logger.error("Invalid request: missing function_id or version_id")
            response = {
                "exit_code": 1,
                "stdout": "",
                "stderr": "Invalid request: missing function_id or version_id",
            }
        else:
            # Execute the function
            response = execute_function(fn_id, version_id, payload)
        
        # Send response
        response_json = json.dumps(response) + '\n'
        conn.sendall(response_json.encode('utf-8'))
        logger.info("Response sent")
    
    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON: {e}")
        error_response = {
            "exit_code": 1,
            "stdout": "",
            "stderr": f"Invalid JSON in request: {str(e)}",
        }
        try:
            conn.sendall((json.dumps(error_response) + '\n').encode('utf-8'))
        except:
            pass
    
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
    
    finally:
        try:
            conn.close()
        except:
            pass
        logger.info("Client disconnected")


def main():
    """Main agent loop."""
    logger.info("Starting function agent")

    ensure_function_mount()
    
    # Create and start server
    server = create_vsock_server()
    
    try:
        while True:
            try:
                conn, addr = server.accept()
                handle_client(conn, addr)
            except KeyboardInterrupt:
                logger.info("Interrupt received, shutting down")
                break
            except Exception as e:
                logger.error(f"Unexpected error in main loop: {e}")
    
    finally:
        server.close()
        logger.info("Agent shutdown complete")


if __name__ == "__main__":
    main()
