package verifier

import "testing"

func TestValidateRequirementsPinned(t *testing.T) {
	findings := validateRequirementsContent("requests==2.31.0\ncolorama==0.4.6\n")
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestValidateRequirementsUnpinned(t *testing.T) {
	findings := validateRequirementsContent("requests>=2.0\n")
	if len(findings) == 0 {
		t.Fatal("expected rejection for unpinned requirement")
	}
}

func TestValidateRequirementsNoVersion(t *testing.T) {
	findings := validateRequirementsContent("requests\n")
	if len(findings) == 0 {
		t.Fatal("expected rejection for bare package name")
	}
}

func TestValidateRequirementsGitURL(t *testing.T) {
	findings := validateRequirementsContent("git+https://github.com/foo/bar.git@v1==1.0.0\n")
	if len(findings) == 0 {
		t.Fatal("expected rejection for git URL")
	}
}

func TestValidateRequirementsIndexURL(t *testing.T) {
	findings := validateRequirementsContent("--index-url https://evil.pypi/simple\nrequests==1.0.0\n")
	if len(findings) == 0 {
		t.Fatal("expected rejection for custom index")
	}
}

func TestVerifyUnpinnedRequirementsRejected(t *testing.T) {
	dir := t.TempDir()
	zipPath := dir + "/src.zip"
	workDir := dir + "/work"

	writeZip(t, zipPath, map[string]string{
		"main.py":          "def handler():\n    return {'ok': True}\n",
		"requirements.txt": "requests\n",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatalf("expected rejected for unpinned requirements, got %+v", result)
	}
}

func TestVerifyPinnedRequirementsPolicyOK(t *testing.T) {
	dir := t.TempDir()
	zipPath := dir + "/src.zip"
	workDir := dir + "/work"

	writeZip(t, zipPath, map[string]string{
		"main.py":          "def handler():\n    return {'ok': True}\n",
		"requirements.txt": "colorama==0.4.6\n",
	})

	result, err := Verify(zipPath, workDir, structuralOnlyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("expected verified (pip-audit skipped), got %+v", result)
	}
}
