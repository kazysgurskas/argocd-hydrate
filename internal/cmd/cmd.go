package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Flags
	applicationsFile string
	outputDir        string

	// Root command
	rootCmd = &cobra.Command{
		Use:   "argocd-hydrate",
		Short: "Hydrate ArgoCD applications into Kubernetes manifests",
		Long: `ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders them into Kubernetes manifests. It supports applications that use Helm charts and directory-based sources.`,
		Run: runHydrate,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Define flags
	rootCmd.PersistentFlags().StringVar(&applicationsFile, "applications", "manifests/applications.yaml", "Path to the file containing ArgoCD Application CRDs")
	rootCmd.PersistentFlags().StringVar(&outputDir, "output", "manifests", "Output directory for the rendered manifests")

	// Add examples
	rootCmd.Example = `  # Use default values
  argocd-hydrate

  # Specify custom applications file and output directory
  argocd-hydrate --applications=apps/applications.yaml --output=rendered

  # Process a specific application file and output to a custom directory
  argocd-hydrate --applications=my-app.yaml --output=k8s-output`

	// Add version info
	rootCmd.Version = "0.0.1"
}
