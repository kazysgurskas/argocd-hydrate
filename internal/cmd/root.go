package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/config"
	"github.com/kazysgurskas/argocd-hydrate/internal/version"
)

// New creates the root command. Running it with no subcommand hydrates, as
// earlier versions did; `hydrate` and `diff` are the explicit forms.
func New() *cobra.Command {
	cfg := config.NewConfig()
	config.SetConfig(cfg)

	root := &cobra.Command{
		Use:   "argocd-hydrate",
		Short: "Hydrate ArgoCD applications into Kubernetes manifests, and diff the results",
		Long: `ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders Kubernetes manifests. It supports applications that use Helm charts
and directory-based sources. "argocd-hydrate diff" compares two rendered trees and writes an HTML report.`,
		Run: runHydrate,
		Example: `  # Hydrate with the default values
  argocd-hydrate

  # Specify custom applications file and output directory
  argocd-hydrate --applications=apps/applications.yaml --output=rendered

  # Compare two rendered trees (directories or zip files) into an HTML report
  argocd-hydrate diff --base base-manifests --head head-manifests --output diff.html`,
		Version: fmt.Sprintf("%s (commit %s, built %s)", version.Version, version.GitCommit, version.BuildDate),
	}
	addHydrateFlags(root, cfg)

	hydrate := &cobra.Command{
		Use:   "hydrate",
		Short: "Render ArgoCD applications into Kubernetes manifests",
		Run:   runHydrate,
	}
	addHydrateFlags(hydrate, cfg)

	root.AddCommand(hydrate, newDiffCommand())
	return root
}

func addHydrateFlags(cmd *cobra.Command, cfg *config.Configuration) {
	cmd.Flags().StringVar(&cfg.ApplicationsFile, "applications", cfg.ApplicationsFile,
		"Path to the file containing ArgoCD Application CRDs")
	cmd.Flags().StringVar(&cfg.OutputDir, "output", cfg.OutputDir,
		"Output directory for the rendered manifests")
	cmd.Flags().StringVar(&cfg.ChartsDir, "charts-dir", cfg.ChartsDir,
		"Directory for storing downloaded Helm charts")
	cmd.Flags().StringVar(&cfg.KubeVersion, "kube-version", cfg.KubeVersion,
		"Kubernetes version to use for rendering Helm charts")
}
