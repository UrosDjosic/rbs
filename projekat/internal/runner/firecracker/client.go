package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// Client communicates with Firecracker via its Unix socket API
type Client struct {
	socketPath string
	client     *http.Client
}

// NewClient creates a new Firecracker API client
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
			Timeout: 30 * time.Second,
		},
	}
}

// MachineConfig represents the VM machine configuration
type MachineConfig struct {
	VCPUCount   int64  `json:"vcpu_count"`
	MemSizeMib  int32  `json:"mem_size_mib"`
	Tracks      string `json:"track_dirty_pages,omitempty"`
}

// BootSource represents the kernel boot source
type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args,omitempty"`
}

// Drive represents a virtual block device
type Drive struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

// Vsock represents a vsock device
type Vsock struct {
	GuestCID int32  `json:"guest_cid"`
	UdsPath  string `json:"uds_path"`
}

// InstanceActionRequest represents an instance action
type InstanceActionRequest struct {
	ActionType string `json:"action_type"`
}

// PutMachineConfig sends machine configuration to Firecracker
func (c *Client) PutMachineConfig(ctx context.Context, config MachineConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return c.request(ctx, "PUT", "/machine-config", bytes.NewReader(data), nil)
}

// PutBootSource sends boot source configuration to Firecracker
func (c *Client) PutBootSource(ctx context.Context, bootSrc BootSource) error {
	data, err := json.Marshal(bootSrc)
	if err != nil {
		return err
	}
	return c.request(ctx, "PUT", "/boot-source", bytes.NewReader(data), nil)
}

// PutDrive sends a drive configuration to Firecracker
func (c *Client) PutDrive(ctx context.Context, drive Drive) error {
	data, err := json.Marshal(drive)
	if err != nil {
		return err
	}
	return c.request(ctx, "PUT", "/drives/"+drive.DriveID, bytes.NewReader(data), nil)
}

// PutVsock sends a vsock configuration to Firecracker
func (c *Client) PutVsock(ctx context.Context, vsock Vsock) error {
	data, err := json.Marshal(vsock)
	if err != nil {
		return err
	}
	return c.request(ctx, "PUT", "/vsock", bytes.NewReader(data), nil)
}

// CreateInstanceAction sends an instance action (e.g., start/stop)
func (c *Client) CreateInstanceAction(ctx context.Context, action InstanceActionRequest) error {
	data, err := json.Marshal(action)
	if err != nil {
		return err
	}
	return c.request(ctx, "PUT", "/actions", bytes.NewReader(data), nil)
}

// request makes a generic request to the Firecracker API
func (c *Client) request(ctx context.Context, method, path string, body io.Reader, respBody interface{}) error {
	url := "http://localhost" + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("firecracker request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("firecracker api error (%d): %s", resp.StatusCode, string(respData))
	}

	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("failed to parse firecracker response: %w", err)
		}
	}

	return nil
}

// GetSocketPath returns the socket path for this client
func (c *Client) GetSocketPath() string {
	return c.socketPath
}

// Helper to construct socket path for a run
func SocketPath(runDir string) string {
	return filepath.Join(runDir, "firecracker.sock")
}
