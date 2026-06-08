package verifier

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var forbiddenExtensions = map[string]struct{}{
	".bat": {}, ".cmd": {}, ".com": {}, ".dll": {}, ".exe": {},
	".jar": {}, ".js": {}, ".msi": {}, ".ps1": {}, ".scr": {},
	".sh": {}, ".so": {}, ".vbs": {}, ".wsf": {},
}

func safeZipEntryName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty zip entry name")
	}
	if strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
		return "", fmt.Errorf("absolute zip entry: %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("path traversal in zip entry: %q", name)
	}
	return clean, nil
}

func unzipChecked(zipPath, destDir string, opts Options) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	if len(r.File) > opts.MaxFiles {
		return fmt.Errorf("too many files in zip: %d (max %d)", len(r.File), opts.MaxFiles)
	}

	var total int64
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rel, err := safeZipEntryName(f.Name)
		if err != nil {
			return err
		}
		if strings.Contains(rel, "/") {
			return fmt.Errorf("nested paths not allowed: %q", rel)
		}

		ext := strings.ToLower(filepath.Ext(rel))
		if _, blocked := forbiddenExtensions[ext]; blocked {
			return fmt.Errorf("forbidden file type in zip: %q", rel)
		}

		if f.UncompressedSize64 > uint64(opts.MaxFileBytes) {
			return fmt.Errorf("file too large: %q", rel)
		}
		total += int64(f.UncompressedSize64)
		if total > opts.MaxUncompressedBytes {
			return fmt.Errorf("uncompressed payload too large (max %d bytes)", opts.MaxUncompressedBytes)
		}

		if err := extractZipFile(f, destDir, rel, opts.MaxFileBytes); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, destDir, rel string, maxFileBytes int64) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry %q: %w", rel, err)
	}
	defer rc.Close()

	outPath := filepath.Join(destDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("mkdir for %q: %w", rel, err)
	}

	dst, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %q: %w", rel, err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, io.LimitReader(rc, maxFileBytes+1))
	if err != nil {
		return fmt.Errorf("write %q: %w", rel, err)
	}
	if n > maxFileBytes {
		return fmt.Errorf("file too large while extracting: %q", rel)
	}
	return nil
}
