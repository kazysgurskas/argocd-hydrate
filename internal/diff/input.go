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
	dir, err = os.MkdirTemp("", "argocd-hydrate-diff-")
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

// extractZip unpacks archive into dest, refusing entries that would escape it.
func extractZip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	root := filepath.Clean(dest) + string(os.PathSeparator)
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
		if err := writeEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func writeEntry(f *zip.File, target string) error {
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in) //nolint:gosec // manifests, bounded by the archive
	return err
}
