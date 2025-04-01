package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
)

// ProcessDirectory processes a directory source
func ProcessDirectory(source *application.Source) (string, error) {
	dirPath := source.Path

	// Check if directory exists
	dirInfo, err := os.Stat(dirPath)
	if err != nil || !dirInfo.IsDir() {
		return "", fmt.Errorf("invalid directory path: %s: %w", dirPath, err)
	}

	// Determine if we should recursively find YAML files
	shouldRecurse := source.ShouldRecurseDirectory()

	// Get all yaml files
	var yamlFiles []string
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process files that end with .yaml
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			// If not recursive, only include files in the root directory
			if !shouldRecurse && filepath.Dir(path) != dirPath {
				return nil
			}
			yamlFiles = append(yamlFiles, path)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	// Sort the files for consistent output
	sort.Strings(yamlFiles)

	fmt.Printf("Processing %d YAML files from directory %s (recurse: %v)\n",
		len(yamlFiles), dirPath, shouldRecurse)

	var manifests []string
	for _, file := range yamlFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", file, err)
		}
		yamlContent := strings.TrimSpace(string(content))

		if !strings.HasPrefix(yamlContent, "---") {
			yamlContent = fmt.Sprintf("---\n%s", yamlContent)
		}
		manifests = append(manifests, yamlContent)
	}

	return strings.Join(manifests, "\n"), nil
}
