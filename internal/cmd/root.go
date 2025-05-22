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
	// Create a new configuration with default values
	cfg := config.NewConfig()

	// Set this as the global configuration
	config.SetConfig(cfg)

	cmd := &cobra.Command{
		Use:   "argocd-hydrate",
		Short: "Hydrate ArgoCD applications into Kubernetes manifests",
		Long: `ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders Kubernetes manifests. It supports applications that use Helm charts
and directory-based sources.`,
		Run: runHydrate,
	}

	// Define flags - these modify the configuration we just created and set as global
	cmd.PersistentFlags().StringVar(&cfg.ApplicationsFile, "applications", cfg.ApplicationsFile,
		"Path to the file containing ArgoCD Application CRDs")
	cmd.PersistentFlags().StringVar(&cfg.OutputDir, "output", cfg.OutputDir,
		"Output directory for the rendered manifests")
	cmd.PersistentFlags().StringVar(&cfg.ChartsDir, "charts-dir", cfg.ChartsDir,
		"Directory for storing downloaded Helm charts")
	cmd.PersistentFlags().StringVar(&cfg.KubeVersion, "kube-version", cfg.KubeVersion,
		"Kubernetes version to use for rendering Helm charts")

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
