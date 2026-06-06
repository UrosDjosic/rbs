package verifier

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func banditAvailable() bool {
	if _, err := exec.LookPath("bandit"); err == nil {
		return true
	}
	for _, py := range []string{"python", "python3", "py"} {
		path, err := exec.LookPath(py)
		if err != nil {
			continue
		}
		if exec.Command(path, "-m", "bandit", "--version").Run() == nil {
			return true
		}
	}
	return false
}

func TestVerifyBanditSubprocess(t *testing.T) {
	if !banditAvailable() {
		t.Skip("bandit not installed")
	}

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"main.py": "import subprocess\n\ndef handler(event=None):\n    subprocess.call('whoami', shell=True)\n    return {'ok': True}\n",
	})

	result, err := Verify(zipPath, workDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Status != StatusRejected {
		t.Fatalf("expected rejected by bandit, got %+v", result)
	}
}

func TestVerifyBanditBenign(t *testing.T) {
	if !banditAvailable() {
		t.Skip("bandit not installed")
	}

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "src.zip")
	workDir := filepath.Join(dir, "work")

	writeZip(t, zipPath, map[string]string{
		"main.py": "def handler(event=None):\n    return {'ok': True}\n",
	})

	result, err := Verify(zipPath, workDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Status != StatusVerified {
		t.Fatalf("expected verified, got %+v", result)
	}
	if len(result.Layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(result.Layers))
	}
}
