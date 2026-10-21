package hydrate

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
)

// renderDirectory joins the .yaml files under source.Path into one stream,
// each file as its own document, in path order. Subdirectories count only
// when the source asks to recurse. The source must be a directory source.
func renderDirectory(source *application.Source) (string, error) {
	root := source.Path
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}
	recurse := source.Directory.Recurse

	var paths []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && !recurse {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".yaml") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk %s: %w", root, err)
	}
	sort.Strings(paths)
	slog.Info("rendering directory", "path", root, "files", len(paths), "recurse", recurse)

	documents := make([]string, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		doc := strings.TrimSpace(string(content))
		if !strings.HasPrefix(doc, "---") {
			doc = "---\n" + doc
		}
		documents = append(documents, doc)
	}
	return strings.Join(documents, "\n"), nil
}
