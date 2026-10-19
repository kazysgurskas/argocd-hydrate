package diff

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// zipDir archives the contents of dir into archive, as `zip -r` would.
func zipDir(t *testing.T, dir, archive string) {
	t.Helper()
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = out.Close() }()
	w := zip.NewWriter(out)
	defer func() { _ = w.Close() }()
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
		defer func() { _ = in.Close() }()
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

func TestExtractZipBudget(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "many.zip")
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(out)
	for i := 0; i <= maxZipEntries; i++ {
		if _, err := w.Create("a/" + strconv.Itoa(i) + ".yaml"); err != nil {
			t.Fatal(err)
		}
	}
	_ = w.Close()
	_ = out.Close()

	if err := extractZip(archive, t.TempDir()); err == nil {
		t.Error("an archive over the entry budget must be refused")
	}
}
