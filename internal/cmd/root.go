package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/types"
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "argocd-hydrate",
	Short: "Hydrate ArgoCD applications into Kubernetes manifests",
	Long: `ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders Kubernetes manifests. It supports applications that use Helm charts
and directory-based sources.`,
	Run: runHydrate,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Define flags
	rootCmd.PersistentFlags().StringVar(&types.Config.ApplicationsFile, "applications", types.Config.ApplicationsFile,
		"Path to the file containing ArgoCD Application CRDs")
	rootCmd.PersistentFlags().StringVar(&types.Config.OutputDir, "output", types.Config.OutputDir,
		"Output directory for the rendered manifests")
	rootCmd.PersistentFlags().StringVar(&types.Config.ChartsDir, "charts-dir", types.Config.ChartsDir,
		"Directory for storing downloaded Helm charts")

	// Add examples
	rootCmd.Example = `  # Use default values
  argocd-hydrate

  # Specify custom applications file and output directory
  argocd-hydrate --applications=apps/applications.yaml --output=rendered

  # Specify custom charts directory
  argocd-hydrate --charts-dir=/path/to/charts`

	// Add version info
	rootCmd.Version = "0.1.0"
}
