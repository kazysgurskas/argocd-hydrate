package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-diff/internal/diff"
	"github.com/kazysgurskas/argocd-diff/internal/report"
	"github.com/kazysgurskas/argocd-diff/internal/version"
)

type diffOptions struct {
	base, head, output string
	summary            string
	baseApplications   string
	headApplications   string
	context            int
	fullTextLimit      int
	meta               report.Meta
}

func addDiffFlags(cmd *cobra.Command, o *diffOptions) {
	f := cmd.Flags()
	f.StringVar(&o.base, "base", "", "Base manifest tree: a directory or a .zip (required)")
	f.StringVar(&o.head, "head", "", "Head manifest tree: a directory or a .zip (required)")
	f.StringVar(&o.output, "output", "diff.html", "Output HTML file")
	f.StringVar(&o.summary, "summary", "", "Also write a Markdown summary, for a pull request comment, to this file")
	f.StringVar(&o.baseApplications, "base-applications", "", "ArgoCD Application resources of the base tree, for project and cluster labels (default: applications.yaml in the base tree, if present)")
	f.StringVar(&o.headApplications, "head-applications", "", "ArgoCD Application resources of the head tree (default: applications.yaml in the head tree, if present)")
	f.IntVar(&o.context, "context", diff.DefaultContext, "Unchanged lines kept around each change")
	f.IntVar(&o.fullTextLimit, "full-text-limit", diff.DefaultFullTextLimit,
		"Largest modified file, in lines, whose full text is embedded so context can be expanded in the viewer (-1: never)")
	f.StringVar(&o.meta.Title, "title", "", "Report title (default: the pull request, or the input paths)")
	f.StringVar(&o.meta.Repo, "repo", envOr("GITHUB_REPOSITORY", ""), "Repository as owner/name")
	f.IntVar(&o.meta.PRNumber, "pr-number", prNumberFromEnv(), "Pull request number")
	f.StringVar(&o.meta.PRTitle, "pr-title", "", "Pull request title")
	f.StringVar(&o.meta.BaseRef, "base-ref", envOr("GITHUB_BASE_REF", ""), "Base branch or ref")
	f.StringVar(&o.meta.HeadRef, "head-ref", envOr("GITHUB_HEAD_REF", ""), "Head branch or ref")
	f.StringVar(&o.meta.Author, "author", "", "Pull request author")
	f.StringVar(&o.meta.PRURL, "pr-url", "", "Pull request URL (default: derived from --repo and --pr-number on github.com)")
	_ = cmd.MarkFlagRequired("base")
	_ = cmd.MarkFlagRequired("head")
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

	apps, err := applicationsOf(baseDir, o.baseApplications, headDir, o.headApplications)
	if err != nil {
		return err
	}
	result, err := diff.Compare(baseDir, headDir, diff.Options{Context: o.context, FullTextLimit: o.fullTextLimit, Applications: apps})
	if err != nil {
		return err
	}

	o.meta.GeneratedAt = time.Now().UTC()
	o.meta.Tool = "argocd-diff " + version.Version
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
	if o.summary != "" {
		if err := writeSummary(o.summary, result); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s: %d files changed (%d added, %d modified, %d deleted), +%d -%d\n",
		o.output, len(result.Files), result.Added, result.Modified, result.Deleted, result.Additions, result.Deletions)
	return nil
}

func writeSummary(path string, result *diff.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := report.WriteMarkdown(f, result, report.MarkdownOptions{}); err != nil {
		_ = f.Close()
		return fmt.Errorf("write summary: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

// applicationsOf reads the Application resources of both trees: the given
// files, or applications.yaml at each tree root when it exists. Head wins
// where both trees define the same Application.
func applicationsOf(baseDir, basePath, headDir, headPath string) (diff.Applications, error) {
	apps := diff.Applications{}
	for _, tree := range []struct{ dir, path string }{{baseDir, basePath}, {headDir, headPath}} {
		path := tree.path
		if path == "" {
			path = filepath.Join(tree.dir, "applications.yaml")
		}
		loaded, err := diff.LoadApplications(path)
		if err != nil {
			if tree.path == "" && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("applications: %w", err)
		}
		apps = apps.Merge(loaded)
	}
	return apps, nil
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

// prNumberFromEnv is the pull request number GitHub Actions implies:
// GITHUB_PR_NUMBER when a workflow exports it, otherwise the number in a
// GITHUB_REF of the form refs/pull/<n>/merge. Zero when neither applies.
func prNumberFromEnv() int {
	if n, err := strconv.Atoi(os.Getenv("GITHUB_PR_NUMBER")); err == nil && n > 0 {
		return n
	}
	return prNumberFromRef(os.Getenv("GITHUB_REF"))
}

func prNumberFromRef(ref string) int {
	rest, ok := strings.CutPrefix(ref, "refs/pull/")
	if !ok {
		return 0
	}
	num, _, ok := strings.Cut(rest, "/")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
