package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const LayerDependencyAudit = "dependency_audit"

type pipAuditReport struct {
	Dependencies []pipAuditDependency `json:"dependencies"`
}

type pipAuditDependency struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Vulns   []pipAuditVuln  `json:"vulns"`
}

type pipAuditVuln struct {
	ID          string   `json:"id"`
	FixVersions []string `json:"fix_versions"`
}

func scanPipAudit(workDir string, opts Options) LayerResult {
	layer := LayerResult{Name: LayerDependencyAudit, OK: true}
	if opts.SkipPipAudit {
		layer.Skipped = true
		return layer
	}

	reqPath := filepath.Join(workDir, "requirements.txt")
	info, err := os.Stat(reqPath)
	if err != nil {
		if os.IsNotExist(err) {
			layer.Skipped = true
			return layer
		}
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "pip-audit-read",
			File:    "requirements.txt",
			Message: err.Error(),
			Layer:   LayerDependencyAudit,
		})
		return layer
	}
	if info.Size() == 0 {
		layer.Skipped = true
		return layer
	}
	if !RequirementsHasPackages(reqPath) {
		layer.Skipped = true
		return layer
	}

	cmd, err := resolvePipAuditCommand(reqPath, opts.PipAuditPath)
	if err != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "pip-audit-missing",
			Message: err.Error(),
			Layer:   LayerDependencyAudit,
		})
		return layer
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "pip-audit-timeout",
			Message: "pip-audit timed out after 90s",
			Layer:   LayerDependencyAudit,
		})
		return layer
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		out = strings.TrimSpace(stderr.String())
	}
	if out == "" && runErr != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "pip-audit-exec",
			Message: runErr.Error(),
			Layer:   LayerDependencyAudit,
		})
		return layer
	}

	var report pipAuditReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "pip-audit-parse",
			Message: fmt.Sprintf("invalid pip-audit JSON: %v", err),
			Layer:   LayerDependencyAudit,
		})
		return layer
	}

	for _, dep := range report.Dependencies {
		for _, vuln := range dep.Vulns {
			fix := ""
			if len(vuln.FixVersions) > 0 {
				fix = fmt.Sprintf(" (fix: %s)", strings.Join(vuln.FixVersions, ", "))
			}
			layer.OK = false
			layer.Findings = append(layer.Findings, Finding{
				Rule:    "known-vulnerability",
				File:    "requirements.txt",
				Message: fmt.Sprintf("%s==%s: %s%s", dep.Name, dep.Version, vuln.ID, fix),
				Layer:   LayerDependencyAudit,
			})
		}
	}

	if !layer.OK {
		return layer
	}
	if runErr != nil {
		// pip-audit may exit non-zero even when JSON is empty and clean on some versions
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			layer.OK = false
			layer.Findings = append(layer.Findings, Finding{
				Rule:    "pip-audit-exec",
				Message: msg,
				Layer:   LayerDependencyAudit,
			})
		}
	}
	return layer
}

// RequirementsHasPackages reports whether requirements.txt contains non-comment package lines.
func RequirementsHasPackages(reqPath string) bool {
	raw, err := os.ReadFile(reqPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			return true
		}
	}
	return false
}

func resolvePipAuditCommand(reqPath string, override string) (*exec.Cmd, error) {
	args := []string{"-r", reqPath, "--format", "json", "--progress-spinner", "off"}
	if override != "" {
		return exec.Command(override, args...), nil
	}
	if path, err := exec.LookPath("pip-audit"); err == nil {
		return exec.Command(path, args...), nil
	}
	for _, py := range []string{"python", "python3", "py"} {
		bin, err := exec.LookPath(py)
		if err != nil {
			continue
		}
		help := exec.Command(bin, "-m", "pip_audit", "--help")
		if help.Run() == nil {
			return exec.Command(bin, append([]string{"-m", "pip_audit"}, args...)...), nil
		}
	}
	return nil, fmt.Errorf("pip-audit not found; install with: pip install pip-audit")
}
