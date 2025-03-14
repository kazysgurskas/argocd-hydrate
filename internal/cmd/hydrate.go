package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/hydrate"
)

// runHydrate is the main function for the hydrate command
func runHydrate(cmd *cobra.Command, args []string) {
	// Load ArgoCD applications
	applications, err := application.LoadApplications(applicationsFile)
	if err != nil {
		fmt.Printf("Error loading applications: %s\n", err)
		os.Exit(1)
	}

	if len(applications) == 0 {
		fmt.Printf("No valid ArgoCD Application CRDs found in: %s\n", applicationsFile)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %s\n", err)
		os.Exit(1)
	}

	// Process each application
	for _, app := range applications {
		name := app.Metadata.Name
		fmt.Printf("Processing application: %s\n", name)

		// Generate the manifest
		manifests, err := hydrate.HydrateFromApplication(app)
		if err != nil {
			fmt.Printf("Error processing application %s: %s\n", name, err)
			continue
		}

		// Write to output file
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", name))
		if err := os.WriteFile(outputPath, []byte(manifests), 0644); err != nil {
			fmt.Printf("Error writing manifests for %s: %s\n", name, err)
			continue
		}
		fmt.Printf("Manifests for %s written to %s\n", name, outputPath)
	}
}
