package renderer

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
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
			sourceManifests, err = ProcessHelmChart(source, name, namespace)
		} else if source.IsDirectory() {
			sourceManifests, err = ProcessDirectory(source)
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
