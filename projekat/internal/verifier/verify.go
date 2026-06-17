package verifier

import (
	"fmt"
	"os"
)

// Verify unpacks zipPath into workDir and runs all verifier layers.
func Verify(zipPath, workDir string, opts *Options) (*Result, error) {
	o := opts.normalized()

	if err := os.RemoveAll(workDir); err != nil {
		return nil, fmt.Errorf("clean workdir: %w", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("create workdir: %w", err)
	}

	result := &Result{WorkDir: workDir}

	if err := unzipChecked(zipPath, workDir, o); err != nil {
		result.OK = false
		result.Status = StatusRejected
		result.Layers = []LayerResult{{
			Name: LayerStructuralAV,
			OK:   false,
			Findings: []Finding{{
				Rule:    "unpack",
				Message: err.Error(),
				Layer:   LayerStructuralAV,
			}},
		}}
		return result, nil
	}

	structural := scanStructuralAV(workDir)
	result.Layers = append(result.Layers, structural)
	if !structural.OK {
		result.OK = false
		result.Status = StatusRejected
		return result, nil
	}

	clamav := scanClamAV(workDir, o)
	result.Layers = append(result.Layers, clamav)
	if !clamav.OK && !clamav.Skipped {
		result.OK = false
		result.Status = StatusRejected
		return result, nil
	}

	bandit := scanBandit(workDir, o)
	result.Layers = append(result.Layers, bandit)
	if !bandit.OK {
		result.OK = false
		result.Status = StatusRejected
		return result, nil
	}

	requirements := scanRequirementsPolicy(workDir)
	result.Layers = append(result.Layers, requirements)
	if !requirements.OK && !requirements.Skipped {
		result.OK = false
		result.Status = StatusRejected
		return result, nil
	}

	pipAudit := scanPipAudit(workDir, o)
	result.Layers = append(result.Layers, pipAudit)
	if !pipAudit.OK && !pipAudit.Skipped {
		result.OK = false
		result.Status = StatusRejected
		return result, nil
	}

	result.OK = true
	result.Status = StatusVerified
	return result, nil
}
