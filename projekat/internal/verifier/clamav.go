package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func scanClamAV(workDir string, opts Options) LayerResult {
	layer := LayerResult{Name: LayerClamAV, OK: true}
	if opts.SkipClamAV {
		layer.Skipped = true
		return layer
	}

	clamscan, err := resolveClamScanPath(opts.ClamScanPath)
	if err != nil {
		layer.Skipped = true
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "clamav-skipped",
			Message: err.Error(),
			Layer:   LayerClamAV,
		})
		return layer
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	args := []string{"-r", workDir, "--no-summary"}
	if dbDir := clamDatabaseDir(opts.ClamAVDatabaseDir); dbDir != "" {
		args = append(args, "-d", dbDir)
	}
	cmd := exec.CommandContext(ctx, clamscan, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())

	if ctx.Err() == context.DeadlineExceeded {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "clamav-timeout",
			Message: "clamscan timed out after 60s",
			Layer:   LayerClamAV,
		})
		return layer
	}

	// Cannot execute (wrong arch, missing DLL, etc.) — skip layer, do not block upload.
	if err != nil && isClamExecFailure(err, out, errOut) {
		layer.Skipped = true
		msg := strings.TrimSpace(errOut)
		if msg == "" {
			msg = err.Error()
		}
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "clamav-skipped",
			Message: "clamscan unavailable: " + msg,
			Layer:   LayerClamAV,
		})
		return layer
	}

	if errOut != "" && isClamDatabaseMissing(errOut) {
		layer.Skipped = true
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "clamav-skipped",
			Message: "virus definitions missing; run: .\\scripts\\setup-clamav.ps1",
			Layer:   LayerClamAV,
		})
		return layer
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, "FOUND") {
			layer.OK = false
			layer.Findings = append(layer.Findings, Finding{
				Rule:    "clamav-infected",
				File:    clamFoundFile(line, workDir),
				Message: line,
				Layer:   LayerClamAV,
			})
		}
	}

	// Exit code 1 with no parseable output still means infection or scan issue.
	if err != nil && !layer.OK {
		return layer
	}
	if err != nil && strings.Contains(err.Error(), "exit status 1") && len(layer.Findings) == 0 {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "clamav-detected",
			Message: "clamscan reported threats",
			Layer:   LayerClamAV,
		})
	}

	if err != nil && !layer.OK {
		return layer
	}
	if err != nil {
		layer.Skipped = true
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "clamav-skipped",
			Message: fmt.Sprintf("clamscan error: %v (%s)", err, errOut),
			Layer:   LayerClamAV,
		})
	}

	return layer
}

func clamDatabaseDir(override string) string {
	if override != "" {
		return override
	}
	if abs, err := filepath.Abs(filepath.Join("storage", "clamav", "database")); err == nil {
		return abs
	}
	return ""
}

func isClamDatabaseMissing(stderr string) bool {
	msg := strings.ToLower(stderr)
	return strings.Contains(msg, "can't open file") ||
		strings.Contains(msg, "cl_load") ||
		strings.Contains(msg, "no such file or directory")
}

func resolveClamScanPath(override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", fmt.Errorf("clamscan override not found: %s", override)
		}
		return override, nil
	}

	if path, err := exec.LookPath("clamscan"); err == nil {
		return path, nil
	}

	if runtime.GOOS == "windows" {
		for _, candidate := range []string{
			`C:\Program Files\ClamAV\clamscan.exe`,
			`C:\Program Files (x86)\ClamAV\clamscan.exe`,
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("clamscan not found; install ClamAV or set ClamScanPath")
}

func isClamExecFailure(err error, stdout, stderr string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error() + " " + stderr + " " + stdout)
	needles := []string{
		"not a valid win32",
		"valid application for this os",
		"cannot find",
		"is not recognized",
		"the system cannot find",
		"failed to run",
		"access is denied",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

func clamFoundFile(line, workDir string) string {
	// Format: "/path/to/file: Win.Test.EICAR-... FOUND"
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 0 {
		return ""
	}
	abs := strings.TrimSpace(parts[0])
	rel, err := filepath.Rel(workDir, abs)
	if err != nil {
		return filepath.Base(abs)
	}
	return filepath.ToSlash(rel)
}
