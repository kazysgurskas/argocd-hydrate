package hydrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/helm"
)

// HydrateFromApplication hydrates ArgoCD application into Kubernetes manifests
func HydrateFromApplication(app application.Application) (string, error) {
	var manifests []string

	// Extract key information from the Application CRD
	name := app.Metadata.Name
	namespace := app.GetEffectiveNamespace()

	// Get all sources for the application
	sources := app.GetSources()

	for _, source := range sources {
		// Skip sources that are just for reference values
		if source.IsValueSource() {
			continue
		}

		var sourceManifests string
		var err error

		if source.IsHelmChart() {
			sourceManifests, err = processHelmChart(source, name, namespace)
		} else if source.IsDirectory() {
			sourceManifests, err = processDirectory(source)
		} else {
			// Dump the source for debugging
			sourceYaml, _ := yaml.Marshal(source)
			fmt.Printf("Unsupported source type for application %s. Source details:\n%s\n", name, string(sourceYaml))
			return "", fmt.Errorf("unsupported source type for application %s", name)
		}

		if err != nil {
			return "", err
		}

		if sourceManifests != "" {
			manifests = append(manifests, sourceManifests)
		}
	}

	if len(manifests) == 0 {
		fmt.Printf("WARNING: No manifests generated for application %s\n", name)
	}

	return strings.Join(manifests, "\n"), nil
}

// processHelmChart processes a Helm chart source
func processHelmChart(source *application.Source, appName, namespace string) (string, error) {
	releaseName := source.GetEffectiveReleaseName(appName)

	chartPath, err := helm.PullChart(source.RepoURL, source.Chart, source.TargetRevision)
	if err != nil {
		return "", err
	}

	var valueFilesPaths []string
	for _, valueFile := range source.Helm.ValueFiles {
		// Remove $values prefix if present
		if strings.HasPrefix(valueFile, "$values/") {
			valueFile = strings.Replace(valueFile, "$values/", "", 1)
		}
		valueFilesPaths = append(valueFilesPaths, valueFile)
	}

	fmt.Printf("Rendering chart %s (version %s) with release name %s in namespace %s\n",
		source.Chart, source.TargetRevision, releaseName, namespace)

	renderedManifest, err := helm.RenderHelmChart(chartPath, releaseName, namespace, source.TargetRevision, valueFilesPaths)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(renderedManifest), nil
}

// processDirectory processes a directory source
func processDirectory(source *application.Source) (string, error) {
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
