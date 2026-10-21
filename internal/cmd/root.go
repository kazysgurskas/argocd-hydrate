// Package cmd defines the argocd-hydrate command line.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/version"
)

// New returns the root command. Run with no subcommand it hydrates, as
// earlier versions did; hydrate and diff are the explicit forms. Errors are
// returned from Execute rather than printed, so the caller reports them
// once.
func New() *cobra.Command {
	o := &hydrateOptions{}
	root := &cobra.Command{
		Use:   "argocd-hydrate",
		Short: "Hydrate ArgoCD applications into Kubernetes manifests, and diff the results",
		Long: `ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders Kubernetes manifests. It supports applications that use Helm charts
and directory-based sources. "argocd-hydrate diff" compares two rendered trees and writes an HTML report.`,
		Example: `  # Hydrate with the default values
  argocd-hydrate

  # Specify custom applications file and output directory
  argocd-hydrate --applications=apps/applications.yaml --output=rendered

  # Compare two rendered trees (directories or zip files) into an HTML report
  argocd-hydrate diff --base base-manifests --head head-manifests --output diff.html`,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version.Version, version.GitCommit, version.BuildDate),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(*cobra.Command, []string) error {
			return runHydrate(o)
		},
	}
	addHydrateFlags(root, o)

	hydrate := &cobra.Command{
		Use:   "hydrate",
		Short: "Render ArgoCD applications into Kubernetes manifests",
		RunE: func(*cobra.Command, []string) error {
			return runHydrate(o)
		},
	}
	addHydrateFlags(hydrate, o)

	root.AddCommand(hydrate, newDiffCommand())
	return root
}
