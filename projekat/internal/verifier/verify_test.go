package verifier

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func structuralOnlyOpts() *Options {
	return &Options{SkipBandit: true, SkipPipAudit: true}
}

func TestVerifyBenign(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"main.py":          "def handler(event=None):\n    return {'ok': True}\n",
		"requirements.txt": "",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Status != StatusVerified {
		t.Fatalf("expected verified, got %+v", result)
	}
}

func TestVerifyMissingMain(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"requirements.txt": "requests\n",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Status != StatusRejected {
		t.Fatalf("expected rejected, got %+v", result)
	}
}

func TestVerifyPathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"../evil.py": "print('x')",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatalf("expected rejected, got %+v", result)
	}
}

func TestVerifyForbiddenExtension(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"main.py": "def handler():\n    pass\n",
		"run.bat": "@echo off\n",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatalf("expected rejected, got %+v", result)
	}
}

func TestVerifyPEMagicInPy(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"main.py": "MZfake payload",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatalf("expected rejected for PE magic bytes, got %+v", result)
	}
}
