package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/diff"
	"github.com/kazysgurskas/argocd-hydrate/internal/report"
	"github.com/kazysgurskas/argocd-hydrate/internal/version"
)

type diffOptions struct {
	base, head, output string
	context            int
	fullTextLimit      int
	meta               report.Meta
}

func newDiffCommand() *cobra.Command {
	o := &diffOptions{}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare two rendered manifest trees and write an HTML report",
		Long: `Compare two trees of rendered manifests, as "argocd-hydrate" writes them, and
write a self-contained HTML report of what changed. Each input is a directory
or a zip archive of one.

Pull request details are optional and default to the GITHUB_* variables that
GitHub Actions sets, so inside a workflow only --base and --head are needed.`,
		Example: `  argocd-hydrate diff --base base-manifests --head head-manifests
  argocd-hydrate diff --base base.zip --head head.zip --output diff.html \
      --repo hostinger/kubernetes-apps --pr-number 42 --pr-title "Bump nginx" \
      --base-ref master --head-ref feat/nginx --author alice`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDiff(cmd, o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.base, "base", "", "Base manifest tree: a directory or a .zip (required)")
	f.StringVar(&o.head, "head", "", "Head manifest tree: a directory or a .zip (required)")
	f.StringVar(&o.output, "output", "diff.html", "Output HTML file")
	f.IntVar(&o.context, "context", diff.DefaultContext, "Unchanged lines kept around each change")
	f.IntVar(&o.fullTextLimit, "full-text-limit", diff.DefaultFullTextLimit,
		"Largest modified file, in lines, whose full text is embedded so context can be expanded in the viewer (-1: never)")
	f.StringVar(&o.meta.Title, "title", "", "Report title (default: the pull request, or the input paths)")
	f.StringVar(&o.meta.Repo, "repo", envOr("GITHUB_REPOSITORY", ""), "Repository as owner/name")
	f.IntVar(&o.meta.PRNumber, "pr-number", envInt("GITHUB_PR_NUMBER"), "Pull request number")
	f.StringVar(&o.meta.PRTitle, "pr-title", "", "Pull request title")
	f.StringVar(&o.meta.BaseRef, "base-ref", envOr("GITHUB_BASE_REF", ""), "Base branch or ref")
	f.StringVar(&o.meta.HeadRef, "head-ref", envOr("GITHUB_HEAD_REF", ""), "Head branch or ref")
	f.StringVar(&o.meta.Author, "author", "", "Pull request author")
	f.StringVar(&o.meta.PRURL, "pr-url", "", "Pull request URL (default: derived from --repo and --pr-number on github.com)")
	_ = cmd.MarkFlagRequired("base")
	_ = cmd.MarkFlagRequired("head")
	return cmd
}

func runDiff(cmd *cobra.Command, o *diffOptions) error {
	baseDir, cleanupBase, err := diff.OpenTree(o.base)
	if err != nil {
		return fmt.Errorf("base: %w", err)
	}
	defer cleanupBase()
	headDir, cleanupHead, err := diff.OpenTree(o.head)
	if err != nil {
		return fmt.Errorf("head: %w", err)
	}
	defer cleanupHead()

	result, err := diff.Compare(baseDir, headDir, diff.Options{Context: o.context, FullTextLimit: o.fullTextLimit})
	if err != nil {
		return err
	}

	o.meta.GeneratedAt = time.Now().UTC()
	o.meta.Tool = "argocd-hydrate " + version.Version
	if o.meta.Title == "" {
		switch {
		case o.meta.Repo != "" && o.meta.PRNumber > 0:
			o.meta.Title = fmt.Sprintf("%s #%d", o.meta.Repo, o.meta.PRNumber)
		default:
			o.meta.Title = fmt.Sprintf("%s → %s", o.base, o.head)
		}
	}
	if o.meta.PRURL == "" && o.meta.PRNumber > 0 && strings.Count(o.meta.Repo, "/") == 1 {
		o.meta.PRURL = fmt.Sprintf("https://github.com/%s/pull/%d", o.meta.Repo, o.meta.PRNumber)
	}

	out, err := os.Create(o.output)
	if err != nil {
		return fmt.Errorf("create %s: %w", o.output, err)
	}
	defer func() { _ = out.Close() }()
	if err := report.Render(out, result, o.meta); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s: %d files changed (%d added, %d modified, %d deleted), +%d -%d\n",
		o.output, len(result.Files), result.Added, result.Modified, result.Deleted, result.Additions, result.Deletions)
	return nil
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func envInt(name string) int {
	n, _ := strconv.Atoi(os.Getenv(name))
	return n
}
