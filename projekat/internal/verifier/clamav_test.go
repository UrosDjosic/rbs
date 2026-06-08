package verifier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanClamAVSkippedWhenMissing(t *testing.T) {
	layer := scanClamAV(t.TempDir(), Options{
		ClamScanPath: filepath.Join(t.TempDir(), "nonexistent-clamscan.exe"),
	})
	if !layer.Skipped {
		t.Fatalf("expected skipped layer, got %+v", layer)
	}
}

func TestScanClamAVSkippedFlag(t *testing.T) {
	layer := scanClamAV(t.TempDir(), Options{SkipClamAV: true})
	if !layer.Skipped || !layer.OK {
		t.Fatalf("expected skipped ok layer, got %+v", layer)
	}
}

func TestResolveClamScanWindowsPath(t *testing.T) {
	const winPath = `C:\Program Files\ClamAV\clamscan.exe`
	if _, err := os.Stat(winPath); err != nil {
		t.Skip("clamscan not installed at default Windows path")
	}
	path, err := resolveClamScanPath("")
	if err != nil {
		t.Fatal(err)
	}
	if path != winPath {
		t.Fatalf("expected %q, got %q", winPath, path)
	}
}
