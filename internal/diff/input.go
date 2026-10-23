package diff

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// OpenTree returns a directory holding the manifest tree at path, which is
// either a directory, returned as it is, or a zip archive, extracted into a
// temporary directory. cleanup removes anything OpenTree created and is safe
// to call either way.
func OpenTree(path string) (dir string, cleanup func(), err error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", func() {}, err
	}
	if info.IsDir() {
		return path, func() {}, nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".zip") {
		return "", func() {}, fmt.Errorf("%s is neither a directory nor a .zip archive", path)
	}
	dir, err = os.MkdirTemp("", "argocd-diff-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	if err := extractZip(path, dir); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("extract %s: %w", path, err)
	}
	return dir, cleanup, nil
}

// What an archive may unpack to. Manifest trees are far below this; the
// budget exists so that a hostile or corrupt archive cannot fill the disk.
const (
	maxZipEntries = 100_000
	maxZipBytes   = 1 << 30
)

// extractZip unpacks archive into dest, refusing entries that would escape
// it and stopping once the budget above is spent.
func extractZip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	if len(r.File) > maxZipEntries {
		return fmt.Errorf("archive has %d entries, more than the %d allowed", len(r.File), maxZipEntries)
	}
	root := filepath.Clean(dest) + string(os.PathSeparator)
	var remaining int64 = maxZipBytes
	for _, f := range r.File {
		target := filepath.Join(dest, f.Name) //nolint:gosec // checked against root below
		if !strings.HasPrefix(target, root) {
			return fmt.Errorf("entry %q escapes the archive root", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		written, err := writeEntry(f, target, remaining)
		if err != nil {
			return err
		}
		remaining -= written
		if remaining <= 0 {
			return fmt.Errorf("archive unpacks to more than the %d bytes allowed", maxZipBytes)
		}
	}
	return nil
}

// writeEntry copies one entry to target, at most limit bytes, and returns
// how many it wrote.
func writeEntry(f *zip.File, target string, limit int64) (int64, error) {
	in, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, err
	}
	defer func() { _ = out.Close() }()
	return io.Copy(out, io.LimitReader(in, limit))
}
