package diff

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// zipDir archives the contents of dir into archive, as `zip -r` would.
func zipDir(t *testing.T, dir, archive string) {
	t.Helper()
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	w := zip.NewWriter(out)
	defer w.Close()
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		f, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(f, in)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExtractZipRefusesEscape(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "evil.zip")
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(out)
	f, _ := w.Create("../escape.yaml")
	_, _ = f.Write([]byte("kind: X\n"))
	_ = w.Close()
	_ = out.Close()

	if err := extractZip(archive, t.TempDir()); err == nil {
		t.Error("an entry escaping the root must be refused")
	}
}
