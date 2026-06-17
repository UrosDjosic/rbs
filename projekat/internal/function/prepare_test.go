package function

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallDependenciesSkipsWithoutRequirements(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstallDependencies(context.Background(), dir, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, DepsDirName)); !os.IsNotExist(err) {
		t.Fatalf("deps dir should not exist: %v", err)
	}
}

func TestInstallDependenciesInstallsPinnedPackage(t *testing.T) {
	src := filepath.Join("..", "..", "samples", "benign", "with_deps")
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

	if err := InstallDependencies(context.Background(), workDir, ""); err != nil {
		t.Skipf("pip install unavailable or failed: %v", err)
	}

	coloramaInit := filepath.Join(workDir, DepsDirName, "colorama", "__init__.py")
	if _, err := os.Stat(coloramaInit); err != nil {
		t.Fatalf("colorama not installed to deps: %v", err)
	}
}

func TestInvalidateExt4Cache(t *testing.T) {
	dir := t.TempDir()
	fnID := "fn-test"
	verID := "ver-test"
	cacheDir := filepath.Join(dir, "cache", "functions", fnID)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(cacheDir, verID+".ext4")
	if err := os.WriteFile(imagePath, []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InvalidateExt4Cache(dir, fnID, verID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("cached image should be removed: %v", err)
	}
}
