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
	// MaxFlags caps the "worth a look" list. Zero means DefaultMaxFlags;
	// negative means no cap.
	MaxFlags int
}

const (
	DefaultCollapseAfter = 8
	DefaultMaxFlags      = 20
)

// riskyKinds are the kinds whose change is called out on its own line, in
// the order they are listed.
var riskyKinds = []string{
	"Secret",
	"Namespace",
	"CustomResourceDefinition",
	"HTTPRoute",
	"PodDisruptionBudget",
}

// WriteMarkdown writes a GitHub-flavoured Markdown summary of result to w,
// meant for a pull request comment: a totals line, anything that deserves a
// second look (deleted resources and changes to sensitive kinds), and a table
// with one row per application, environment and cluster.
func WriteMarkdown(w io.Writer, result *diff.Result, opts MarkdownOptions) error {
	if result == nil {
		result = &diff.Result{}
	}
	collapseAfter := opts.CollapseAfter
	if collapseAfter == 0 {
		collapseAfter = DefaultCollapseAfter
	}
	maxFlags := opts.MaxFlags
	if maxFlags == 0 {
		maxFlags = DefaultMaxFlags
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

	if flags := flagLines(result); len(flags) > 0 {
		b.WriteString("\n**Worth a look**\n\n")
		shown := flags
		if maxFlags > 0 && len(shown) > maxFlags {
			shown = shown[:maxFlags]
		}
		for _, f := range shown {
			b.WriteString("- " + f + "\n")
		}
		if rest := len(flags) - len(shown); rest > 0 {
			fmt.Fprintf(&b, "- … and %s\n", plural(rest, "more"))
		}
	}

	rows := appRows(result)
	collapse := collapseAfter > 0 && len(rows) > collapseAfter
	b.WriteString("\n")
	if collapse {
		fmt.Fprintf(&b, "<details>\n<summary>%s</summary>\n\n", plural(len(rows), "application"))
	}
	b.WriteString("| Application | Environment | Cluster | Added | Modified | Deleted | Lines |\n")
	b.WriteString("|---|---|---|---:|---:|---:|---:|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | +%d / −%d |\n",
			cell(r.app), cell(r.env), cell(r.cluster),
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
	app, env, cluster        string
	added, modified, deleted int
	additions, deletions     int
}

// appRows aggregates files per application, environment and cluster, in
// that order. Files whose path does not follow the hydrate layout fall under
// their first path segment.
func appRows(result *diff.Result) []appRow {
	type key struct{ app, env, cluster string }
	byKey := map[key]*appRow{}
	for _, f := range result.Files {
		k := key{}
		if f.Resource != nil {
			k = key{f.Resource.App, f.Resource.Environment, f.Resource.Cluster}
		}
		if k.app == "" {
			k.app, _, _ = strings.Cut(f.Path, "/")
		}
		r := byKey[k]
		if r == nil {
			r = &appRow{app: k.app, env: k.env, cluster: k.cluster}
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
		if a.env != b.env {
			return a.env < b.env
		}
		return a.cluster < b.cluster
	})
	return rows
}

// flagLines lists what a reviewer should not miss: every deleted resource
// first, then added or modified resources of a risky kind, grouped by kind in
// the order of riskyKinds.
func flagLines(result *diff.Result) []string {
	var lines []string
	for _, f := range result.Files {
		if f.Status == diff.StatusDeleted {
			lines = append(lines, "Deleted "+describe(f))
		}
	}
	for _, kind := range riskyKinds {
		for _, f := range result.Files {
			if f.Status == diff.StatusDeleted || f.Resource == nil || f.Resource.Kind != kind {
				continue
			}
			verb := "Modified"
			if f.Status == diff.StatusAdded {
				verb = "Added"
			}
			lines = append(lines, verb+" "+describe(f))
		}
	}
	return lines
}

// describe renders a file as "`Kind` `namespace/name` in `app.env.cluster`",
// falling back to the path when the resource is unknown.
func describe(f diff.File) string {
	r := f.Resource
	if r == nil || r.Kind == "" {
		return "`" + f.Path + "`"
	}
	name := r.Name
	if r.Namespace != "" {
		name = r.Namespace + "/" + name
	}
	s := "`" + r.Kind + "`"
	if name != "" {
		s += " `" + name + "`"
	}
	where := strings.Trim(strings.Join([]string{r.App, r.Environment, r.Cluster}, "."), ".")
	if where != "" {
		s += " in `" + where + "`"
	}
	return s
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
