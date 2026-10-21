package report

import (
	"strings"
	"testing"

	"github.com/kazysgurskas/argocd-hydrate/internal/diff"
)

func file(app, environment, cluster, namespace, kind, name string, status diff.Status, add, del int) diff.File {
	path := app + "." + environment + "." + cluster + "/"
	if namespace != "" {
		path += namespace + "/"
	}
	path += kind + "/" + name + ".yaml"
	return diff.File{
		Path:   path,
		Status: status,
		Resource: &diff.Resource{
			App: app, Environment: environment, Cluster: cluster,
			Namespace: namespace, Kind: kind, Name: name,
		},
		Additions: add,
		Deletions: del,
	}
}

func render(t *testing.T, r *diff.Result, opts MarkdownOptions) string {
	t.Helper()
	var b strings.Builder
	if err := WriteMarkdown(&b, r, opts); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestMarkdownNoChanges(t *testing.T) {
	out := render(t, &diff.Result{}, MarkdownOptions{})
	if !strings.HasPrefix(out, "**No rendered changes.**") {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if strings.Contains(out, "|") {
		t.Fatalf("table rendered for an empty result:\n%s", out)
	}
}

func TestMarkdownSummary(t *testing.T) {
	r := &diff.Result{
		Files: []diff.File{
			file("shop", "prod", "euw2", "shop", "Deployment", "web", diff.StatusModified, 3, 1),
			file("shop", "prod", "euw2", "shop", "Secret", "web-env", diff.StatusModified, 1, 1),
			file("shop", "prod", "euw2", "shop", "Service", "web-old", diff.StatusDeleted, 0, 12),
			file("shop", "stage", "euw3", "shop", "HTTPRoute", "web", diff.StatusAdded, 20, 0),
			file("ops", "prod", "euw2", "", "Namespace", "ops", diff.StatusAdded, 5, 0),
			{Path: "stray.yaml", Status: diff.StatusAdded, Additions: 2},
		},
		Added: 3, Modified: 2, Deleted: 1, Additions: 31, Deletions: 14,
	}
	out := render(t, r, MarkdownOptions{})

	for _, want := range []string{
		"**6 manifests changed**: 3 added, 2 modified, 1 deleted (+31 / −14)",
		"| `ops` | `prod` | `euw2` | 1 |  |  | +5 / −0 |",
		"| `shop` | `prod` | `euw2` |  | 2 | 1 | +4 / −14 |",
		"| `shop` | `stage` | `euw3` | 1 |  |  | +20 / −0 |",
		"| `stray.yaml` |  |  | 1 |  |  | +2 / −0 |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<details>") {
		t.Errorf("table collapsed below the threshold:\n%s", out)
	}
}

func TestMarkdownCollapsesLargeTables(t *testing.T) {
	r := &diff.Result{}
	for i := 0; i < 10; i++ {
		r.Files = append(r.Files, file("app"+string(rune('a'+i)), "prod", "euw2", "ns", "ConfigMap", "cm", diff.StatusModified, 1, 1))
	}
	r.Modified = 10
	out := render(t, r, MarkdownOptions{CollapseAfter: 8})
	if !strings.Contains(out, "<details>\n<summary>10 applications</summary>") {
		t.Errorf("table not collapsed:\n%s", out)
	}
	out = render(t, r, MarkdownOptions{CollapseAfter: -1})
	if strings.Contains(out, "<details>") {
		t.Errorf("table collapsed with CollapseAfter -1:\n%s", out)
	}
}
