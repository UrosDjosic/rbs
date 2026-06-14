package firecracker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"oblak/internal/common/ids"
	"oblak/internal/runner"
)

// FirecrackerRunner executes functions using Firecracker microVMs
// Each invocation spawns a lightweight VM that runs the function
type FirecrackerRunner struct {
	kernelPath     string // Path to kernel image
	rootfsPath     string // Path to rootfs image
	runsDir        string // Base directory for run data
	cacheDir       string // Persistent cache for immutable function images
	firecrackerBin string // Path to firecracker binary
}

// NewFirecrackerRunner creates a new Firecracker runner
func NewFirecrackerRunner(kernelPath, rootfsPath, runsDir string) (*FirecrackerRunner, error) {
	// Verify files exist
	for _, path := range []string{kernelPath, rootfsPath} {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("firecracker resource not found: %s: %w", path, err)
		}
	}

	// Create runs directory
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create runs directory: %w", err)
	}
	cacheDir := filepath.Join(runsDir, "cache", "functions")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Find firecracker binary
	fcBin, err := exec.LookPath("firecracker")
	if err != nil {
		return nil, fmt.Errorf("firecracker binary not found in PATH: %w", err)
	}

	return &FirecrackerRunner{
		kernelPath:     kernelPath,
		rootfsPath:     rootfsPath,
		runsDir:        runsDir,
		cacheDir:       cacheDir,
		firecrackerBin: fcBin,
	}, nil
}

// Invoke executes a function in a Firecracker VM
func (fr *FirecrackerRunner) Invoke(ctx context.Context, req runner.InvokeRequest) (*runner.InvokeResult, error) {
	// Generate unique run ID
	runID, err := ids.NewToken(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate run ID: %w", err)
	}

	runDir := filepath.Join(fr.runsDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}
	defer os.RemoveAll(runDir) // Clean up run directory after invocation

	// Copy rootfs for this VM (since each VM needs its own writable copy)
	vmRootfs := filepath.Join(runDir, "rootfs.ext4")
	if err := copyFile(fr.rootfsPath, vmRootfs); err != nil {
		return nil, fmt.Errorf("failed to copy rootfs: %w", err)
	}

	functionImage, err := fr.cachedFunctionImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create function image: %w", err)
	}

	// Create vsock socket path
	vsockPath := filepath.Join(runDir, "vsock.sock")
	socketPath := filepath.Join(runDir, "firecracker.sock")

	// Start firecracker process
	fcProc, err := fr.startFirecracker(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to start firecracker: %w", err)
	}
	defer fcProc.Process.Kill() //nolint

	// Wait for API socket to be ready
	if err := waitForSocket(socketPath, 5*time.Second); err != nil {
		return nil, fmt.Errorf("firecracker api socket not ready: %w", err)
	}

	// Configure and start VM
	client := NewClient(socketPath)
	if err := fr.configureVM(ctx, client, vmRootfs, functionImage, vsockPath); err != nil {
		return nil, fmt.Errorf("failed to configure vm: %w", err)
	}

	// Connect to guest via vsock. The VM can be started while userspace is
	// still booting, so wait until the guest agent is actually listening.
	invokeResult, err := fr.invokeGuestWithRetry(ctx, vsockPath, 8, req.FunctionID, req.VersionID, req.Payload, 45*time.Second)
	if err != nil {
		return nil, fmt.Errorf("guest invocation failed: %w", err)
	}

	// Stop VM
	if err := client.CreateInstanceAction(ctx, InstanceActionRequest{ActionType: "SendCtrlAltDel"}); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to stop vm: %v\n", err)
	}

	return invokeResult, nil
}

// startFirecracker starts the firecracker process with the given socket path
func (fr *FirecrackerRunner) startFirecracker(socketPath string) (*exec.Cmd, error) {
	cmd := exec.Command(fr.firecrackerBin,
		"--api-sock", socketPath,
	)

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// configureVM configures the VM with kernel, rootfs, function drive, and vsock.
func (fr *FirecrackerRunner) configureVM(ctx context.Context, client *Client, vmRootfs, functionImage, vsockPath string) error {
	// Set machine config
	if err := client.PutMachineConfig(ctx, MachineConfig{
		VCPUCount:  2,
		MemSizeMib: 512,
	}); err != nil {
		return fmt.Errorf("failed to set machine config: %w", err)
	}

	// Set boot source
	if err := client.PutBootSource(ctx, BootSource{
		KernelImagePath: fr.kernelPath,
		BootArgs:        "console=ttyS0 reboot=k panic=1 pci=off nomodules root=/dev/vda rw",
	}); err != nil {
		return fmt.Errorf("failed to set boot source: %w", err)
	}

	// Add rootfs drive
	if err := client.PutDrive(ctx, Drive{
		DriveID:      "rootfs",
		PathOnHost:   vmRootfs,
		IsRootDevice: true,
		IsReadOnly:   false,
	}); err != nil {
		return fmt.Errorf("failed to add rootfs drive: %w", err)
	}

	// Add function volume (work directory as read-only)
	if err := client.PutDrive(ctx, Drive{
		DriveID:      "function",
		PathOnHost:   functionImage,
		IsRootDevice: false,
		IsReadOnly:   true,
	}); err != nil {
		return fmt.Errorf("failed to add function drive: %w", err)
	}

	// Configure vsock
	if err := client.PutVsock(ctx, Vsock{
		GuestCID: 3, // Standard guest CID
		UdsPath:  vsockPath,
	}); err != nil {
		return fmt.Errorf("failed to configure vsock: %w", err)
	}

	// Start the VM
	if err := client.CreateInstanceAction(ctx, InstanceActionRequest{
		ActionType: "InstanceStart",
	}); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	// Give the kernel a short head start. Guest-agent readiness is checked by
	// invokeGuestWithRetry, because first boot can take longer than VM start.
	time.Sleep(500 * time.Millisecond)

	return nil
}

func (fr *FirecrackerRunner) invokeGuestWithRetry(ctx context.Context, vsockPath string, port uint32, fnID, verID string, payload []byte, timeout time.Duration) (*runner.InvokeResult, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, err := fr.invokeGuest(ctx, vsockPath, port, fnID, verID, payload)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryableVsockError(err) {
			return nil, err
		}

		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("guest agent did not become ready")
	}
	return nil, fmt.Errorf("guest agent not ready after %v: %w", timeout, lastErr)
}

func isRetryableVsockError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "failed to connect to guest") ||
		strings.Contains(msg, "failed to read vsock connect ack") ||
		strings.Contains(msg, "rejected connect request")
}

// invokeGuest communicates with guest agent over vsock to invoke the function
func (fr *FirecrackerRunner) invokeGuest(ctx context.Context, vsockPath string, port uint32, fnID, verID string, payload []byte) (*runner.InvokeResult, error) {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{
		Name: vsockPath,
		Net:  "unix",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guest: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set vsock deadline: %w", err)
	}

	// Firecracker's host-side vsock UDS expects a text connect command before
	// application data. It replies with "OK <host-port>\n" after the guest ACKs.
	if _, err := fmt.Fprintf(conn, "CONNECT %d\n", port); err != nil {
		return nil, fmt.Errorf("failed to select guest vsock port %d: %w", port, err)
	}
	reader := bufio.NewReader(conn)
	ack, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read vsock connect ack: %w", err)
	}
	if !strings.HasPrefix(strings.ToUpper(ack), "OK ") {
		return nil, fmt.Errorf("guest vsock port %d rejected connect request: %q", port, strings.TrimSpace(ack))
	}
	if err := conn.SetDeadline(time.Now().Add(35 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set invocation deadline: %w", err)
	}

	// Send invocation request to guest agent
	invokeReq := map[string]interface{}{
		"function_id": fnID,
		"version_id":  verID,
		"payload":     string(payload),
	}

	reqData, err := json.Marshal(invokeReq)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Write(append(reqData, '\n')); err != nil {
		return nil, fmt.Errorf("failed to send request to guest: %w", err)
	}

	// Read response from guest agent
	var result runner.InvokeResult
	dec := json.NewDecoder(reader)
	if err := dec.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to read response from guest: %w", err)
	}

	return &result, nil
}

func (fr *FirecrackerRunner) cachedFunctionImage(ctx context.Context, req runner.InvokeRequest) (string, error) {
	if err := validateCacheComponent(req.FunctionID, "function id"); err != nil {
		return "", err
	}
	if err := validateCacheComponent(req.VersionID, "version id"); err != nil {
		return "", err
	}

	functionCacheDir := filepath.Join(fr.cacheDir, req.FunctionID)
	if err := os.MkdirAll(functionCacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create function cache directory: %w", err)
	}

	imagePath := filepath.Join(functionCacheDir, req.VersionID+".ext4")
	if _, err := os.Stat(imagePath); err == nil {
		return imagePath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to stat cached function image: %w", err)
	}

	tmpImage := filepath.Join(functionCacheDir, req.VersionID+"."+fmt.Sprint(time.Now().UnixNano())+".tmp")
	if err := createExt4ImageFromDir(ctx, req.WorkDir, tmpImage); err != nil {
		_ = os.Remove(tmpImage)
		return "", err
	}

	if _, err := os.Stat(imagePath); err == nil {
		_ = os.Remove(tmpImage)
		return imagePath, nil
	} else if !os.IsNotExist(err) {
		_ = os.Remove(tmpImage)
		return "", fmt.Errorf("failed to stat cached function image after build: %w", err)
	}

	if err := os.Rename(tmpImage, imagePath); err != nil {
		_ = os.Remove(tmpImage)
		return "", fmt.Errorf("failed to store cached function image: %w", err)
	}

	return imagePath, nil
}

func validateCacheComponent(value, name string) error {
	if value == "" {
		return fmt.Errorf("%s is empty", name)
	}
	if value == "." || value == ".." || strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("%s contains unsafe path characters", name)
	}
	return nil
}

// Close performs cleanup
func (fr *FirecrackerRunner) Close() error {
	// Clean up any remaining runs
	return os.RemoveAll(fr.runsDir)
}

// Helper functions

func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("socket not available after %v", timeout)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

func createExt4ImageFromDir(ctx context.Context, srcDir, imagePath string) error {
	const minImageBytes int64 = 16 << 20

	var total int64
	if err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	}); err != nil {
		return err
	}

	size := total + (8 << 20)
	if size < minImageBytes {
		size = minImageBytes
	}

	img, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	if err := img.Truncate(size); err != nil {
		_ = img.Close()
		return err
	}
	if err := img.Close(); err != nil {
		return err
	}

	mkfs, err := findExecutable("mkfs.ext4", "/usr/sbin/mkfs.ext4", "/sbin/mkfs.ext4")
	if err != nil {
		return fmt.Errorf("mkfs.ext4 not found in PATH: %w", err)
	}

	cmd := exec.CommandContext(ctx, mkfs, "-q", "-F", "-d", srcDir, imagePath)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func findExecutable(name string, fallbacks ...string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	for _, path := range fallbacks {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}
