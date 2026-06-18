package local

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"oblak/internal/runner"
)

// LocalRunner executes functions as subprocess on the host machine
// Suitable for development and testing on macOS, Windows, and Linux
type LocalRunner struct {
	pythonBin string
}

// NewLocalRunner creates a new local runner using the system Python
func NewLocalRunner(pythonBin string) *LocalRunner {
	if pythonBin == "" {
		pythonBin = resolvePythonBin()
	}
	return &LocalRunner{pythonBin: pythonBin}
}

func resolvePythonBin() string {
	candidates := []string{"python3", "python"}
	if runtime.GOOS == "windows" {
		candidates = []string{"python", "python3"}
	}
	for _, bin := range candidates {
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}
	return "python3"
}

// Invoke executes a function by running main.py with the provided payload
func (lr *LocalRunner) Invoke(ctx context.Context, req runner.InvokeRequest) (*runner.InvokeResult, error) {
	// Locate main.py in the work directory
	mainPath := filepath.Join(req.WorkDir, "main.py")
	if _, err := os.Stat(mainPath); err != nil {
		return nil, fmt.Errorf("main.py not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, lr.pythonBin, "main.py")
	cmd.Dir = req.WorkDir
	if depsDir := filepath.Join(req.WorkDir, "deps"); dirExists(depsDir) {
		cmd.Env = append(os.Environ(), "PYTHONPATH="+depsDir)
	}

	// Set up I/O
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(req.Payload)

	// Run the command
	err := cmd.Run()

	result := &runner.InvokeResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Error = err.Error()
		}
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// Close performs cleanup (no-op for local runner)
func (lr *LocalRunner) Close() error {
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
