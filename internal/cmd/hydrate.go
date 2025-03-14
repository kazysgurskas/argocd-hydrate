package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/types"
	"github.com/kazysgurskas/argocd-hydrate/internal/renderer"
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

	// Ensure output directory exists
	if err := os.MkdirAll(types.Config.OutputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory %s: %v\n", types.Config.OutputDir, err)
		os.Exit(1)
	}

	// Process each application
	for _, app := range applications {
		fmt.Printf("Processing application: %s\n", app.Metadata.Name)

		// Render the application
		manifests, err := renderer.HydrateFromApplication(app)
		if err != nil {
			fmt.Printf("Error hydrating application %s: %v\n", app.Metadata.Name, err)
			continue
		}

		// Skip if no manifests were generated
		if strings.TrimSpace(manifests) == "" {
			fmt.Printf("No manifests generated for application %s\n", app.Metadata.Name)
			continue
		}

		// Write manifests to file
		outputFile := filepath.Join(types.Config.OutputDir, app.Metadata.Name+".yaml")
		if err := os.WriteFile(outputFile, []byte(manifests), 0644); err != nil {
			fmt.Printf("Error writing manifests for application %s: %v\n", app.Metadata.Name, err)
			continue
		}

		fmt.Printf("Successfully hydrated application %s to %s\n", app.Metadata.Name, outputFile)
	}
}
