// Package cmd defines the argocd-diff command line.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-diff/internal/version"
)

// New returns the root command, which diffs two trees of rendered manifests.
// hydrate, which renders such a tree, is its subcommand. Errors are
// returned from Execute rather than printed, so the caller reports them
// once.
func New() *cobra.Command {
	o := &diffOptions{}
	root := &cobra.Command{
		Use:   "argocd-diff",
		Short: "Diff rendered ArgoCD application manifests",
		Long: `Compare two trees of rendered manifests, as "argocd-diff hydrate" writes
them, and write a self-contained HTML report of what changed. Each input is a
directory or a zip archive of one.

Pull request details are optional. Inside GitHub Actions the repository, base
and head refs come from GITHUB_REPOSITORY, GITHUB_BASE_REF and GITHUB_HEAD_REF,
and the number from GITHUB_REF on pull_request events (refs/pull/<n>/merge) or
from GITHUB_PR_NUMBER if the workflow exports it, so only --base and --head are
needed there.`,
		Example: `  # Render the two trees to compare, from each checkout
  argocd-diff hydrate --applications manifests/applications.yaml --output base-manifests
  argocd-diff hydrate --applications manifests/applications.yaml --output head-manifests

  argocd-diff --base base-manifests --head head-manifests
  argocd-diff --base base-manifests --head head-manifests --summary diff.md
  argocd-diff --base base.zip --head head.zip --output diff.html \
      --repo hostinger/kubernetes-apps --pr-number 42 --pr-title "Bump nginx" \
      --base-ref master --head-ref feat/nginx --author alice`,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version.Version, version.GitCommit, version.BuildDate),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDiff(cmd, o)
		},
	}
	addDiffFlags(root, o)
	root.AddCommand(newHydrateCommand())
	return root
}
