package function

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"oblak/internal/verifier"
)

const DepsDirName = "deps"

// InstallDependencies installs pinned requirements into work/deps on deploy.
// Skips when requirements.txt is missing or has no package lines.
func InstallDependencies(ctx context.Context, workDir, pythonBin string) error {
	reqPath := filepath.Join(workDir, "requirements.txt")
	if !verifier.RequirementsHasPackages(reqPath) {
		return nil
	}

	depsDir := filepath.Join(workDir, DepsDirName)
	if err := os.RemoveAll(depsDir); err != nil {
		return fmt.Errorf("clean deps dir: %w", err)
	}
	if err := os.MkdirAll(depsDir, 0o755); err != nil {
		return fmt.Errorf("create deps dir: %w", err)
	}

	if pythonBin == "" {
		pythonBin = resolvePythonBin()
	}

	if ctx == nil {
		ctx = context.Background()
	}
	installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(installCtx, pythonBin,
		"-m", "pip", "install",
		"-r", reqPath,
		"-t", depsDir,
		"--no-cache-dir",
	)
	cmd.Dir = workDir

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("pip install failed: %w: %s", err, msg)
		}
		return fmt.Errorf("pip install failed: %w", err)
	}
	return nil
}

// InvalidateExt4Cache removes a cached Firecracker function image after deploy.
func InvalidateExt4Cache(runsDir, functionID, versionID string) error {
	if runsDir == "" || functionID == "" || versionID == "" {
		return nil
	}
	imagePath := filepath.Join(runsDir, "cache", "functions", functionID, versionID+".ext4")
	if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("invalidate function image cache: %w", err)
	}
	return nil
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
