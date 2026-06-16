# Enable Firecracker - Quick Start

Your Firecracker setup is ready! Here are two ways to enable it:

## Option 1: Use the Setup Script (Recommended)

```bash
cd ~/firecracker-workspace  # or wherever you are
bash /mnt/c/Users/Aleksa/Desktop/rbs/projekat/run-firecracker.sh
```

This script will:
1. Verify Firecracker is installed
2. Check kernel and rootfs exist
3. Verify /dev/kvm is available
4. Copy resources to project storage
5. Set environment variables
6. Start the API server

## Option 2: Manual Setup

```bash
# In Ubuntu terminal
cd /mnt/c/Users/Aleksa/Desktop/rbs/projekat

# Set environment variables
export FIRECRACKER_KERNEL=$HOME/firecracker-workspace/vmlinux
export FIRECRACKER_ROOTFS=$HOME/firecracker-workspace/rootfs.ext4

# Start server
go run ./cmd/api/main.go
```

## Expected Output

When starting with Firecracker enabled, you should see:

```
Attempting to initialize Firecracker runner...
  Kernel: /home/aleksa/firecracker-workspace/vmlinux
  Rootfs: /home/aleksa/firecracker-workspace/rootfs.ext4
Firecracker runner initialized successfully
  Kernel: /home/aleksa/firecracker-workspace/vmlinux
  Rootfs: /home/aleksa/firecracker-workspace/rootfs.ext4
  Runs:   /mnt/c/Users/Aleksa/Desktop/rbs/projekat/storage/runs
api listening on http://127.0.0.1:8080
ui at http://127.0.0.1:8080/ui
```

## Test Firecracker

Once running, test a function invocation:

```bash
# Create test function
mkdir -p storage/functions/test/v1/work
echo 'print("Hello from Firecracker VM!")' > storage/functions/test/v1/work/main.py

# Invoke (in another terminal)
curl -X POST http://127.0.0.1:8080/invoke/test
```

Response should include:
```json
{
  "exit_code": 0,
  "stdout": "Hello from Firecracker VM!\n",
  "stderr": ""
}
```

Note: First invocation will take 2-3 seconds (VM startup).

## Troubleshooting

**Error: "Kernel not found"**
```bash
cd ~/firecracker-workspace
ls -lh vmlinux rootfs.ext4
```

**Error: "Rootfs not found"**
```bash
cd ~/firecracker-workspace  
ls -lh rootfs.ext4
# Should be ~1GB file
```

**Error: "/dev/kvm not found"**
```bash
ls -l /dev/kvm
# If not found: nested virt not enabled in WSL
# Edit ~/.wslconfig and add: nestedVirtualization=true
# Then: wsl --shutdown && wsl -d Ubuntu-24.04
```

**Error: "/dev/kvm permission denied"**
```bash
sudo usermod -a -G kvm $USER
newgrp kvm
```

## Next Steps

1. ✅ Firecracker runner implemented in Go
2. ✅ Guest agent installed in rootfs
3. ✅ Environment variables configured
4. ⏳ Test VM startup and function execution
5. ⏳ Add security policies (seccomp, AppArmor)
6. ⏳ Optimize VM boot time

See [FIRECRACKER_ARCHITECTURE.md](FIRECRACKER_ARCHITECTURE.md) for architecture details.
