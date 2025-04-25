package hydrate

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/render"
	"github.com/kazysgurskas/argocd-hydrate/pkg/util"
)

// ManifestInfo represents a single Kubernetes manifest
type ManifestInfo struct {
	Kind    string
	Name    string
	Content string
}

// HydrateFromApplication hydrates ArgoCD application into Kubernetes manifests
func HydrateFromApplication(app application.Application) ([]ManifestInfo, error) {
	var allManifests []ManifestInfo

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

		var sourceManifestsStr string
		var err error

		if source.IsHelmChart() {
			sourceManifestsStr, err = render.ProcessHelmChart(source, name, namespace)
		} else if source.IsDirectory() {
			sourceManifestsStr, err = render.ProcessDirectory(source)
		} else {
			// Dump the source for debugging
			sourceYaml, _ := yaml.Marshal(source)
			fmt.Printf("Unsupported source type for application %s. Source details:\n%s\n", name, string(sourceYaml))
			return nil, fmt.Errorf("unsupported source type for application %s", name)
		}

		if err != nil {
			return nil, err
		}

		if sourceManifestsStr != "" {
			// Parse the manifests into individual documents
			manifests, err := parseManifests(sourceManifestsStr)
			if err != nil {
				return nil, fmt.Errorf("error parsing manifests for application %s: %w", name, err)
			}
			allManifests = append(allManifests, manifests...)
		}
	}

	if len(allManifests) == 0 {
		fmt.Printf("WARNING: No manifests generated for application %s\n", name)
	}

	return allManifests, nil
}

// parseManifests splits a multi-document YAML string into individual ManifestInfo objects
func parseManifests(yamlContent string) ([]ManifestInfo, error) {
	re := regexp.MustCompile(`(?m)^---`)
	docs := re.Split(yamlContent, -1)
	var manifests []ManifestInfo

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Parse the YAML to extract kind and name
		var obj map[string]interface{}
		err := yaml.Unmarshal([]byte(doc), &obj)
		if err != nil {
			return nil, fmt.Errorf("error parsing YAML document: %w", err)
		}

		kind, ok := util.GetNestedString(obj, "kind")
		if !ok || kind == "" {
			// Skip documents without a kind
			continue
		}

		// Get the name from metadata
		metadata, ok := obj["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := metadata["name"].(string)
		if !ok || name == "" {
			// Try to use generateName if name is not present
			generateName, ok := metadata["generateName"].(string)
			if !ok || generateName == "" {
				// Skip documents without a name or generateName
				continue
			}
			name = generateName + "generated"
		}

		// Add document separator back to the content for proper YAML format
		content := "---\n" + doc

		// Ensure content ends with a newline
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}

		manifests = append(manifests, ManifestInfo{
			Kind:    kind,
			Name:    name,
			Content: content,
		})
	}

	return manifests, nil
}
