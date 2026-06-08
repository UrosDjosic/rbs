package verifier

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var allowedRootFiles = map[string]struct{}{
	"main.py":          {},
	"requirements.txt": {},
}

func scanStructuralAV(workDir string) LayerResult {
	layer := LayerResult{Name: LayerStructuralAV, OK: true}

	entries, err := os.ReadDir(workDir)
	if err != nil {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "read-workdir",
			Message: fmt.Sprintf("cannot read workdir: %v", err),
			Layer:   LayerStructuralAV,
		})
		return layer
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			layer.OK = false
			layer.Findings = append(layer.Findings, Finding{
				Rule:    "no-subdirs",
				File:    e.Name(),
				Message: "subdirectories are not allowed",
				Layer:   LayerStructuralAV,
			})
			continue
		}
		names = append(names, e.Name())
	}

	hasMain := false
	for _, name := range names {
		if _, ok := allowedRootFiles[name]; !ok {
			layer.OK = false
			layer.Findings = append(layer.Findings, Finding{
				Rule:    "allowed-files",
				File:    name,
				Message: "only main.py and requirements.txt are allowed at zip root",
				Layer:   LayerStructuralAV,
			})
			continue
		}
		if name == "main.py" {
			hasMain = true
		}

		ext := strings.ToLower(filepath.Ext(name))
		if _, blocked := forbiddenExtensions[ext]; blocked {
			layer.OK = false
			layer.Findings = append(layer.Findings, Finding{
				Rule:    "forbidden-extension",
				File:    name,
				Message: "forbidden file extension",
				Layer:   LayerStructuralAV,
			})
		}

		finding, blocked := checkMagicBytes(filepath.Join(workDir, name), name)
		if blocked {
			layer.OK = false
			layer.Findings = append(layer.Findings, finding)
		}
	}

	if !hasMain {
		layer.OK = false
		layer.Findings = append(layer.Findings, Finding{
			Rule:    "require-main-py",
			File:    "main.py",
			Message: "main.py is required at zip root",
			Layer:   LayerStructuralAV,
		})
	}

	return layer
}

func checkMagicBytes(path, displayName string) (Finding, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Finding{
			Rule:    "read-file",
			File:    displayName,
			Message: fmt.Sprintf("cannot read file: %v", err),
			Layer:   LayerStructuralAV,
		}, true
	}
	if len(b) >= 2 && bytes.Equal(b[:2], []byte{'M', 'Z'}) {
		return Finding{
			Rule:    "magic-bytes-pe",
			File:    displayName,
			Message: "Windows executable (PE) content is not allowed",
			Layer:   LayerStructuralAV,
		}, true
	}
	if len(b) >= 4 && bytes.Equal(b[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return Finding{
			Rule:    "magic-bytes-elf",
			File:    displayName,
			Message: "ELF executable content is not allowed",
			Layer:   LayerStructuralAV,
		}, true
	}
	return Finding{}, false
}
