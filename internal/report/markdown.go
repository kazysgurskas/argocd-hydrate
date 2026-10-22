package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/kazysgurskas/argocd-hydrate/internal/diff"
)

// MarkdownOptions tune the Markdown summary.
type MarkdownOptions struct {
	// CollapseAfter is the number of application rows above which the table
	// is wrapped in a collapsed <details> block. Zero means DefaultCollapseAfter;
	// negative means never collapse.
	CollapseAfter int
}

// DefaultCollapseAfter is the CollapseAfter used when MarkdownOptions leaves
// it zero.
const DefaultCollapseAfter = 8

// WriteMarkdown writes a GitHub-flavoured Markdown summary of result to w,
// meant for a pull request comment: a totals line and a table with one row
// per application, project and cluster.
func WriteMarkdown(w io.Writer, result *diff.Result, opts MarkdownOptions) error {
	if result == nil {
		result = &diff.Result{}
	}
	collapseAfter := opts.CollapseAfter
	if collapseAfter == 0 {
		collapseAfter = DefaultCollapseAfter
	}

	var b strings.Builder
	if len(result.Files) == 0 {
		b.WriteString("**No rendered changes.** The manifests of the affected applications are identical before and after this pull request.\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	fmt.Fprintf(&b, "**%s changed**: %s (+%d / −%d)\n",
		plural(len(result.Files), "manifest"),
		strings.Join(nonZero(
			count(result.Added, "added"),
			count(result.Modified, "modified"),
			count(result.Deleted, "deleted"),
		), ", "),
		result.Additions, result.Deletions)

	rows := appRows(result)
	collapse := collapseAfter > 0 && len(rows) > collapseAfter
	b.WriteString("\n")
	if collapse {
		fmt.Fprintf(&b, "<details>\n<summary>%s</summary>\n\n", plural(len(rows), "application"))
	}
	b.WriteString("| Application | Project | Cluster | Added | Modified | Deleted | Lines |\n")
	b.WriteString("|---|---|---|---:|---:|---:|---:|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | +%d / −%d |\n",
			cell(r.app), cell(r.project), cell(r.cluster),
			blankZero(r.added), blankZero(r.modified), blankZero(r.deleted),
			r.additions, r.deletions)
	}
	if collapse {
		b.WriteString("\n</details>\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

type appRow struct {
	app, project, cluster    string
	added, modified, deleted int
	additions, deletions     int
}

// appRows aggregates files per application, project and cluster, in
// that order. Files whose path does not follow the hydrate layout fall under
// their first path segment.
func appRows(result *diff.Result) []appRow {
	type key struct{ app, project, cluster string }
	byKey := map[key]*appRow{}
	for _, f := range result.Files {
		k := key{}
		if f.Resource != nil {
			k = key{f.Resource.App, f.Resource.Project, f.Resource.Cluster}
		}
		if k.app == "" {
			k.app, _, _ = strings.Cut(f.Path, "/")
		}
		r := byKey[k]
		if r == nil {
			r = &appRow{app: k.app, project: k.project, cluster: k.cluster}
			byKey[k] = r
		}
		switch f.Status {
		case diff.StatusAdded:
			r.added++
		case diff.StatusModified:
			r.modified++
		case diff.StatusDeleted:
			r.deleted++
		}
		r.additions += f.Additions
		r.deletions += f.Deletions
	}
	rows := make([]appRow, 0, len(byKey))
	for _, r := range byKey {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.app != b.app {
			return a.app < b.app
		}
		if a.project != b.project {
			return a.project < b.project
		}
		return a.cluster < b.cluster
	})
	return rows
}

func count(n int, what string) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d %s", n, what)
}

func nonZero(parts ...string) []string {
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func blankZero(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprint(n)
}

// cell escapes a value for a Markdown table cell.
func cell(s string) string {
	if s == "" {
		return ""
	}
	return "`" + strings.ReplaceAll(s, "|", "\\|") + "`"
}
