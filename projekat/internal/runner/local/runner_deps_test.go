package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblak/internal/function"
	"oblak/internal/runner"
)

func TestInvokeWithDependencies(t *testing.T) {
	src := filepath.Join("..", "..", "..", "samples", "benign", "with_deps")
	workDir := t.TempDir()
	for _, name := range []string{"main.py", "requirements.txt"} {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Skipf("sample not available: %v", err)
		}
		if err := os.WriteFile(filepath.Join(workDir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := function.InstallDependencies(context.Background(), workDir, ""); err != nil {
		t.Skipf("pip install unavailable: %v", err)
	}

	lr := NewLocalRunner("")
	result, err := lr.Invoke(context.Background(), runner.InvokeRequest{
		WorkDir: workDir,
		Payload: []byte(`{"name":"oblak"}`),
	})
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", result.ExitCode, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"colorama_version": "0.4.6"`) {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}
