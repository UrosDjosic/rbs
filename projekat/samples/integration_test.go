package samples_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"oblak/internal/function"
	"oblak/internal/runner"
	"oblak/internal/runner/local"
	"oblak/internal/verifier"
)

type sampleCase struct {
	path       string
	wantOK     bool
	failLayer  string
	needBandit bool
	needClamAV bool
	invoke     string
}

var allSamples = []sampleCase{
	{path: "benign/hello_world", wantOK: true},
	{path: "benign/add", wantOK: true, invoke: `{"a":2,"b":2}`},
	{path: "benign/with_deps", wantOK: true, invoke: `{"name":"oblak"}`},
	{path: "malicious/unpinned_requirements", wantOK: false, failLayer: verifier.LayerRequirementsPolicy},
	{path: "malicious/missing_main", wantOK: false, failLayer: verifier.LayerStructuralAV},
	{path: "malicious/forbidden_script", wantOK: false, failLayer: verifier.LayerStructuralAV},
	{path: "malicious/nested_main", wantOK: false, failLayer: verifier.LayerStructuralAV},
	{path: "malicious/eval_exec", wantOK: false, failLayer: verifier.LayerStaticBandit, needBandit: true},
	{path: "malicious/subprocess_shell", wantOK: false, failLayer: verifier.LayerStaticBandit, needBandit: true},
	{path: "malicious/clamav_marker", wantOK: false, failLayer: verifier.LayerClamAV, needClamAV: true},
}

func TestAllSamplesVerify(t *testing.T) {
	root := samplesRoot(t)
	projectRoot := findProjectRoot(t, root)
	clamDB := filepath.Join(projectRoot, "storage", "clamav", "database")

	for _, tc := range allSamples {
		tc := tc
		name := strings.ReplaceAll(tc.path, "/", "_")
		t.Run(name, func(t *testing.T) {
			if tc.needBandit && !banditAvailable() {
				t.Skip("bandit not installed")
			}
			if tc.needClamAV && !clamavAvailable(clamDB) {
				t.Skip("clamav not available with project database")
			}

			sampleDir := filepath.Join(root, filepath.FromSlash(tc.path))
			if _, err := os.Stat(sampleDir); err != nil {
				t.Fatalf("sample missing: %v", err)
			}

			dir := t.TempDir()
			zipPath := filepath.Join(dir, "src.zip")
			workDir := filepath.Join(dir, "work")
			if err := zipSampleDir(sampleDir, zipPath); err != nil {
				t.Fatal(err)
			}

			opts := &verifier.Options{ClamAVDatabaseDir: clamDB}
			result, err := verifier.Verify(zipPath, workDir, opts)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("status=%s ok=%v layers=%s", result.Status, result.OK, formatLayers(result))
			if result.OK != tc.wantOK {
				t.Fatalf("verify: want ok=%v got ok=%v findings=%s", tc.wantOK, result.OK, formatFindings(result))
			}
			if !tc.wantOK && tc.failLayer != "" {
				if !layerFailed(result, tc.failLayer) {
					t.Fatalf("expected layer %q to fail, got: %s", tc.failLayer, formatLayers(result))
				}
			}

			if tc.wantOK && tc.invoke != "" {
				t.Run("invoke", func(t *testing.T) {
					if err := function.InstallDependencies(context.Background(), workDir, ""); err != nil {
						t.Fatalf("pip install: %v", err)
					}
					lr := local.NewLocalRunner("")
					res, err := lr.Invoke(context.Background(), runner.InvokeRequest{
						WorkDir: workDir,
						Payload: []byte(tc.invoke),
					})
					if err != nil {
						t.Fatal(err)
					}
					if res.ExitCode != 0 {
						t.Fatalf("invoke exit=%d stdout=%q stderr=%q", res.ExitCode, res.Stdout, res.Stderr)
					}
					t.Logf("stdout: %s", strings.TrimSpace(res.Stdout))
				})
			}
		})
	}
}

func findProjectRoot(t *testing.T, start string) string {
	t.Helper()
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func clamavAvailable(dbDir string) bool {
	if _, err := os.Stat(filepath.Join(dbDir, "main.cvd")); err != nil {
		return false
	}
	clamscan := `C:\Program Files\ClamAV\clamscan.exe`
	if _, err := os.Stat(clamscan); err != nil {
		if _, err := exec.LookPath("clamscan"); err != nil {
			return false
		}
	}
	return true
}

func samplesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(wd) == "samples" {
		return wd
	}
	return filepath.Join(wd, "samples")
}

func zipSampleDir(dir, zipPath string) error {
	dir = filepath.Clean(dir)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, ".git/") {
			return nil
		}
		if rel == "payload.json" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = rel
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(zipPath, buf.Bytes(), 0o644)
}

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

func layerFailed(result *verifier.Result, name string) bool {
	for _, layer := range result.Layers {
		if layer.Name == name && !layer.OK && !layer.Skipped {
			return true
		}
	}
	return false
}

func formatLayers(result *verifier.Result) string {
	parts := make([]string, 0, len(result.Layers))
	for _, l := range result.Layers {
		state := "ok"
		if l.Skipped {
			state = "skipped"
		} else if !l.OK {
			state = "FAIL"
		}
		parts = append(parts, l.Name+":"+state)
	}
	return strings.Join(parts, " ")
}

func formatFindings(result *verifier.Result) string {
	var msgs []string
	for _, l := range result.Layers {
		for _, f := range l.Findings {
			msgs = append(msgs, l.Name+"/"+f.Rule+": "+f.Message)
		}
	}
	if len(msgs) == 0 {
		b, _ := json.Marshal(result.Layers)
		return string(b)
	}
	return strings.Join(msgs, "; ")
}
