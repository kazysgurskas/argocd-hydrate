package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/config"
)

// getVersion returns the version information filled by LDFLAGS during build
func getVersion() struct {
	Version   string
	GitCommit string
	BuildDate string
} {
	return struct {
		Version   string
		GitCommit string
		BuildDate string
	}{
		Version:   "dev",
		GitCommit: "none",
		BuildDate: "unknown",
	}
}

// New creates a new root command for the argocd-hydrate CLI
func New() *cobra.Command {
	config := config.GetConfig()

	cmd := &cobra.Command{
		Use:   "argocd-hydrate",
		Short: "Hydrate ArgoCD applications into Kubernetes manifests",
		Long: `ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders Kubernetes manifests. It supports applications that use Helm charts
and directory-based sources.`,
		Run: runHydrate,
	}

	// Define flags
	cmd.PersistentFlags().StringVar(&config.ApplicationsFile, "applications", "default",
		"Path to the file containing ArgoCD Application CRDs")
	cmd.PersistentFlags().StringVar(&config.OutputDir, "output", "default",
		"Output directory for the rendered manifests")
	cmd.PersistentFlags().StringVar(&config.ChartsDir, "charts-dir", "default",
		"Directory for storing downloaded Helm charts")

	// Add examples
	cmd.Example = `  # Use default values
  argocd-hydrate

  # Specify custom applications file and output directory
  argocd-hydrate --applications=apps/applications.yaml --output=rendered

  # Specify custom charts directory
  argocd-hydrate --charts-dir=/path/to/charts`

	// Use version information from LDFLAGS
	versionInfo := getVersion()
	versionTemplate := fmt.Sprintf("Version: %s\nGit Commit: %s\nBuild Date: %s",
		versionInfo.Version, versionInfo.GitCommit, versionInfo.BuildDate)
	cmd.Version = versionTemplate

	return cmd
}
