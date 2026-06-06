package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const LayerStaticBandit = "static_bandit"

type banditReport struct {
	Results []banditIssue `json:"results"`
	Errors  []banditError `json:"errors"`
}

type banditIssue struct {
	Filename        string `json:"filename"`
	LineNumber      int    `json:"line_number"`
	IssueText       string `json:"issue_text"`
	IssueSeverity   string `json:"issue_severity"`
	IssueConfidence string `json:"issue_confidence"`
	TestID          string `json:"test_id"`
	TestName        string `json:"test_name"`
}

type banditError struct {
	Filename string `json:"filename"`
	Message  string `json:"message"`
}

func scanBandit(workDir string, opts Options) LayerResult {
	layer := LayerResult{Name: LayerStaticBandit, OK: true}
	if opts.SkipBandit {
		return layer
	}

	cmd, err := resolveBanditCommand(workDir, opts.BanditPath)
	if err != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "bandit-missing",
			Message: err.Error(),
			Layer:   LayerStaticBandit,
		})
		return layer
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // bandit exits 1 when findings exist; JSON is still valid
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		out = strings.TrimSpace(stderr.String())
	}

	if ctx.Err() == context.DeadlineExceeded {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "bandit-timeout",
			Message: "bandit scan timed out after 30s",
			Layer:   LayerStaticBandit,
		})
		return layer
	}

	if out == "" {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "bandit-exec",
			Message: "bandit produced no output",
			Layer:   LayerStaticBandit,
		})
		return layer
	}

	var report banditReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "bandit-parse",
			Message: fmt.Sprintf("cannot parse bandit JSON: %v", err),
			Layer:   LayerStaticBandit,
		})
		return layer
	}

	for _, e := range report.Errors {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "bandit-error",
			File:    relFile(workDir, e.Filename),
			Message: e.Message,
			Layer:   LayerStaticBandit,
		})
	}

	minSev := strings.ToUpper(opts.BanditMinSeverity)
	for _, issue := range report.Results {
		if !severityAtLeast(issue.IssueSeverity, minSev) {
			continue
		}
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule: "bandit:" + issue.TestID,
			File: relFile(workDir, issue.Filename),
			Message: fmt.Sprintf("%s (severity=%s confidence=%s line=%d)",
				issue.IssueText, issue.IssueSeverity, issue.IssueConfidence, issue.LineNumber),
			Layer: LayerStaticBandit,
		})
	}

	return layer
}

func resolveBanditCommand(workDir, override string) (*exec.Cmd, error) {
	if override != "" {
		return exec.Command(override, "-r", workDir, "-f", "json", "-q"), nil
	}

	if path, err := exec.LookPath("bandit"); err == nil {
		return exec.Command(path, "-r", workDir, "-f", "json", "-q"), nil
	}

	for _, py := range []string{"python", "python3", "py"} {
		path, err := exec.LookPath(py)
		if err != nil {
			continue
		}
		probe := exec.Command(path, "-m", "bandit", "--version")
		if probe.Run() == nil {
			return exec.Command(path, "-m", "bandit", "-r", workDir, "-f", "json", "-q"), nil
		}
	}

	return nil, fmt.Errorf("bandit not found; install with: pip install bandit")
}

func severityAtLeast(sev, min string) bool {
	order := map[string]int{
		"LOW":      1,
		"MEDIUM":   2,
		"HIGH":     3,
		"CRITICAL": 4,
	}
	s := order[strings.ToUpper(sev)]
	m := order[strings.ToUpper(min)]
	if m == 0 {
		m = order["MEDIUM"]
	}
	return s >= m
}

func relFile(workDir, filename string) string {
	if filename == "" {
		return ""
	}
	rel, err := filepath.Rel(workDir, filename)
	if err != nil {
		return filepath.Base(filename)
	}
	return filepath.ToSlash(rel)
}
