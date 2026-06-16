package runner

import "context"

// InvokeRequest represents a function invocation request
type InvokeRequest struct {
	FunctionID string
	VersionID  string
	WorkDir    string // Path to unpacked function directory
	Payload    []byte // Request payload/stdin
}

// InvokeResult represents the result of a function invocation
type InvokeResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

// Runner interface represents the execution backend
type Runner interface {
	// Invoke executes a function and returns the result
	Invoke(ctx context.Context, req InvokeRequest) (*InvokeResult, error)
	// Close performs cleanup (stop VMs, close sockets, etc.)
	Close() error
}
