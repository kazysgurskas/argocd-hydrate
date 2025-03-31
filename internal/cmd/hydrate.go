package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/types"
	"github.com/kazysgurskas/argocd-hydrate/internal/renderer"
	"github.com/kazysgurskas/argocd-hydrate/pkg/util"
)

// runHydrate is the main function for the hydrate command
func runHydrate(cmd *cobra.Command, args []string) {
	// Load applications
	applications, err := application.LoadApplications(types.Config.ApplicationsFile)
	if err != nil {
		fmt.Printf("Error loading applications: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d ArgoCD application(s) in %s\n", len(applications), types.Config.ApplicationsFile)

	// Ensure base output directory exists
	if err := os.MkdirAll(types.Config.OutputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory %s: %v\n", types.Config.OutputDir, err)
		os.Exit(1)
	}

	// Process each application
	for _, app := range applications {
		fmt.Printf("Processing application: %s\n", app.Metadata.Name)

		// Create application output directory
		appOutputDir := filepath.Join(types.Config.OutputDir, app.Metadata.Name)
		if err := os.MkdirAll(appOutputDir, 0755); err != nil {
			fmt.Printf("Error creating application directory %s: %v\n", appOutputDir, err)
			os.Exit(1)
		}

		// Render the application
		manifests, err := renderer.HydrateFromApplication(app)
		if err != nil {
			fmt.Printf("Error hydrating application %s: %v\n", app.Metadata.Name, err)
			os.Exit(1)
		}

		// Skip if no manifests were generated
		if len(manifests) == 0 {
			fmt.Printf("No manifests generated for application %s\n", app.Metadata.Name)
			continue
		}

		// Keep track of the resources written
		resourceCounter := make(map[string]int)

		// Write each manifest to a separate file
		for _, manifest := range manifests {
			// Create directory for this resource type
			resourceTypeDir := filepath.Join(appOutputDir, manifest.Kind)
			if err := os.MkdirAll(resourceTypeDir, 0755); err != nil {
				fmt.Printf("Error creating resource type directory %s: %v\n", resourceTypeDir, err)
				os.Exit(1)
			}

			// Form output file path
			resourceName := util.SanitizeFileName(manifest.Name)

			// Check if we've already seen a resource with this name and kind
			// If so, append a counter to make the filename unique
			key := manifest.Kind + "_" + resourceName
			if count, exists := resourceCounter[key]; exists {
				resourceCounter[key] = count + 1
				resourceName = fmt.Sprintf("%s-%d", resourceName, count)
			} else {
				resourceCounter[key] = 1
			}

			outputFile := filepath.Join(resourceTypeDir, resourceName+".yaml")

			// Write manifest to file
			if err := os.WriteFile(outputFile, []byte(manifest.Content), 0644); err != nil {
				fmt.Printf("Error writing manifest %s/%s: %v\n", manifest.Kind, manifest.Name, err)
				os.Exit(1)
			}
		}

		fmt.Printf("Successfully hydrated application %s with %d manifests\n", app.Metadata.Name, len(manifests))
	}
}
