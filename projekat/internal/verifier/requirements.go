package verifier

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const LayerRequirementsPolicy = "requirements_policy"

const maxRequirementLines = 30

// pinnedRequirement matches: package==1.2.3 or package[extra]==1.2.3
var pinnedRequirement = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*(\[[^\]]+\])?==[^#\s;]+$`)

var forbiddenRequirementPatterns = []struct {
	sub  string
	rule string
	msg  string
}{
	{"-e ", "editable", "editable installs (-e) are not allowed"},
	{"--editable", "editable", "editable installs are not allowed"},
	{"--index-url", "index-url", "custom package indexes are not allowed"},
	{"--extra-index-url", "extra-index-url", "custom package indexes are not allowed"},
	{"-i ", "index-url", "custom package indexes are not allowed"},
	{"--index ", "index-url", "custom package indexes are not allowed"},
	{"git+", "vcs-url", "VCS/git URLs are not allowed"},
	{"hg+", "vcs-url", "VCS URLs are not allowed"},
	{"svn+", "vcs-url", "VCS URLs are not allowed"},
	{"bzr+", "vcs-url", "VCS URLs are not allowed"},
	{"http://", "url", "direct URL dependencies are not allowed"},
	{"https://", "url", "direct URL dependencies are not allowed"},
	{"file:", "local-path", "local path dependencies are not allowed"},
	{"../", "local-path", "path traversal in requirements is not allowed"},
	{`..\`, "local-path", "path traversal in requirements is not allowed"},
	{"-r ", "includes", "recursive requirement files (-r) are not allowed"},
	{"--requirement", "includes", "recursive requirement files are not allowed"},
	{"-c ", "constraints", "constraint files are not allowed"},
	{"--constraint", "constraints", "constraint files are not allowed"},
}

func scanRequirementsPolicy(workDir string) LayerResult {
	layer := LayerResult{Name: LayerRequirementsPolicy, OK: true}

	reqPath := filepath.Join(workDir, "requirements.txt")
	info, err := os.Stat(reqPath)
	if err != nil {
		if os.IsNotExist(err) {
			layer.Skipped = true
			return layer
		}
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "requirements-read",
			File:    "requirements.txt",
			Message: fmt.Sprintf("cannot read requirements.txt: %v", err),
			Layer:   LayerRequirementsPolicy,
		})
		return layer
	}
	if info.Size() == 0 {
		layer.Skipped = true
		return layer
	}

	raw, err := os.ReadFile(reqPath)
	if err != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "requirements-read",
			File:    "requirements.txt",
			Message: fmt.Sprintf("cannot read requirements.txt: %v", err),
			Layer:   LayerRequirementsPolicy,
		})
		return layer
	}

	findings := validateRequirementsContent(string(raw))
	if len(findings) > 0 {
		layer.OK = false
		layer.Findings = append(layer.Findings, findings...)
	}
	return layer
}

func validateRequirementsContent(content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")
	packageLines := 0

	for i, line := range lines {
		lineNum := i + 1
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		packageLines++
		if packageLines > maxRequirementLines {
			findings = append(findings, Finding{
				Rule:    "requirements-limit",
				File:    "requirements.txt",
				Message: fmt.Sprintf("too many dependency lines (max %d)", maxRequirementLines),
				Layer:   LayerRequirementsPolicy,
			})
			break
		}

		lower := strings.ToLower(line)
		for _, p := range forbiddenRequirementPatterns {
			if strings.Contains(lower, p.sub) {
				findings = append(findings, Finding{
					Rule:    "requirements-" + p.rule,
					File:    "requirements.txt",
					Message: fmt.Sprintf("line %d: %s", lineNum, p.msg),
					Layer:   LayerRequirementsPolicy,
				})
			}
		}

		if strings.ContainsAny(line, "<>!~") && !strings.Contains(line, "==") {
			findings = append(findings, Finding{
				Rule:    "requirements-unpinned",
				File:    "requirements.txt",
				Message: fmt.Sprintf("line %d: only exact pins (==) are allowed, got %q", lineNum, line),
				Layer:   LayerRequirementsPolicy,
			})
			continue
		}

		if !strings.Contains(line, "==") {
			findings = append(findings, Finding{
				Rule:    "requirements-unpinned",
				File:    "requirements.txt",
				Message: fmt.Sprintf("line %d: package must be pinned with ==, got %q", lineNum, line),
				Layer:   LayerRequirementsPolicy,
			})
			continue
		}

		if !pinnedRequirement.MatchString(line) {
			findings = append(findings, Finding{
				Rule:    "requirements-format",
				File:    "requirements.txt",
				Message: fmt.Sprintf("line %d: invalid pinned requirement format, expected name==version, got %q", lineNum, line),
				Layer:   LayerRequirementsPolicy,
			})
		}
	}

	return findings
}
