package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"oblak/internal/verifier"
)

func main() {
	zipPath := flag.String("zip", "", "path to uploaded src.zip")
	workDir := flag.String("workdir", "", "directory to unpack into (default: <zip-dir>/work)")
	skipBandit := flag.Bool("skip-bandit", false, "skip Bandit static analysis layer")
	skipClamAV := flag.Bool("skip-clamav", false, "skip ClamAV scan layer")
	skipPipAudit := flag.Bool("skip-pip-audit", false, "skip pip-audit dependency scan layer")
	clamScanPath := flag.String("clamscan", "", "path to clamscan.exe (optional)")
	flag.Parse()

	if *zipPath == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/verifier --zip path/to/src.zip [--workdir path]")
		os.Exit(2)
	}

	zipPathClean := filepath.Clean(*zipPath)
	dest := *workDir
	if dest == "" {
		dest = filepath.Join(filepath.Dir(zipPathClean), "work")
	}

	opts := &verifier.Options{
		SkipBandit:   *skipBandit,
		SkipClamAV:   *skipClamAV,
		SkipPipAudit: *skipPipAudit,
		ClamScanPath: *clamScanPath,
	}
	result, err := verifier.Verify(zipPathClean, dest, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify error:", err)
		os.Exit(2)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)

	if result.OK {
		os.Exit(0)
	}
	os.Exit(1)
}
