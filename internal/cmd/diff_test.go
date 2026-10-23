package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRNumberFromRef(t *testing.T) {
	tests := map[string]int{
		"refs/pull/42/merge": 42,
		"refs/pull/7/head":   7,
		"refs/heads/master":  0,
		"refs/pull/x/merge":  0,
		"refs/pull/42":       0,
		"":                   0,
	}
	for ref, want := range tests {
		t.Run(ref, func(t *testing.T) {
			if got := prNumberFromRef(ref); got != want {
				t.Errorf("prNumberFromRef(%q) = %d, want %d", ref, got, want)
			}
		})
	}
}

func TestPRNumberFromEnv(t *testing.T) {
	t.Setenv("GITHUB_PR_NUMBER", "")
	t.Setenv("GITHUB_REF", "refs/pull/314228/merge")
	if got := prNumberFromEnv(); got != 314228 {
		t.Errorf("from GITHUB_REF: %d", got)
	}
	t.Setenv("GITHUB_PR_NUMBER", "9")
	if got := prNumberFromEnv(); got != 9 {
		t.Errorf("GITHUB_PR_NUMBER should win: %d", got)
	}
}

func TestApplicationsOf(t *testing.T) {
	base, head := t.TempDir(), t.TempDir()
	app := func(name, project, cluster string) string {
		return "---\napiVersion: argoproj.io/v1alpha1\nkind: Application\nmetadata:\n  name: " + name +
			"\nspec:\n  project: " + project + "\n  destination:\n    name: " + cluster + "\n"
	}
	if err := os.WriteFile(filepath.Join(base, "applications.yaml"), []byte(app("a", "prod", "c1")+app("gone", "prod", "c1")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(head, "applications.yaml"), []byte(app("a", "prod", "c2")+app("new", "stage", "c3")), 0o644); err != nil {
		t.Fatal(err)
	}
	apps, err := applicationsOf(base, "", head, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 3 || apps["a"].Cluster != "c2" || apps["gone"].Project != "prod" || apps["new"].Cluster != "c3" {
		t.Errorf("apps = %+v", apps)
	}
	// No applications.yaml in a tree is fine; an explicit missing file is not.
	if apps, err := applicationsOf(t.TempDir(), "", t.TempDir(), ""); err != nil || len(apps) != 0 {
		t.Errorf("no files: %v %v", apps, err)
	}
	if _, err := applicationsOf(base, filepath.Join(base, "nope.yaml"), head, ""); err == nil {
		t.Error("explicit missing file should fail")
	}
}

// TestRootDiffsTwoTrees pins the root command's job: argocd-diff with
// --base and --head writes the report and the summary.
func TestRootDiffsTwoTrees(t *testing.T) {
	dir := t.TempDir()
	for side, replicas := range map[string]string{"base": "1", "head": "2"} {
		path := filepath.Join(dir, side, "web.stage.c1", "shop", "Deployment", "web.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  namespace: shop\nspec:\n  replicas: " + replicas + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	html, summary := filepath.Join(dir, "diff.html"), filepath.Join(dir, "diff.md")

	root := New()
	root.SetOut(io.Discard)
	root.SetArgs([]string{"--base", filepath.Join(dir, "base"), "--head", filepath.Join(dir, "head"), "--output", html, "--summary", summary})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(html)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<script type="application/json" id="diff-data">`) {
		t.Error("report lacks the embedded diff data")
	}
	md, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(md), "**1 manifest changed**") {
		t.Errorf("summary:\n%s", md)
	}
}

func TestRootRequiresBaseAndHead(t *testing.T) {
	root := New()
	root.SetOut(io.Discard)
	root.SetArgs(nil)
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "required flag") {
		t.Errorf("want a required-flag error, got %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRootHydratesAndDiffsCheckouts pins the one-command form: --applications
// names the Application file inside each checkout, whose source paths are
// relative to that checkout, and the rendered trees land under --rendered.
func TestRootHydratesAndDiffsCheckouts(t *testing.T) {
	dir := t.TempDir()
	for side, replicas := range map[string]string{"base": "1", "head": "2"} {
		root := filepath.Join(dir, side)
		writeFile(t, filepath.Join(root, "apps", "web", "deployment.yaml"),
			"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  namespace: shop\nspec:\n  replicas: "+replicas+"\n")
		writeFile(t, filepath.Join(root, "manifests", "applications.yaml"),
			"apiVersion: argoproj.io/v1alpha1\nkind: Application\nmetadata:\n  name: web.stage.c1\nspec:\n  project: stage\n  destination:\n    name: c1\n    namespace: shop\n  source:\n    repoURL: https://github.com/example/gitops\n    path: apps/web\n    directory: {}\n")
	}
	rendered := filepath.Join(dir, "rendered")
	html, summary := filepath.Join(dir, "diff.html"), filepath.Join(dir, "diff.md")
	// Left by an earlier run; must not reach the diff.
	stale := filepath.Join(rendered, "head", "gone.stage.c1", "Kind", "x.yaml")
	writeFile(t, stale, "kind: Kind\nmetadata:\n  name: x\n")

	root := New()
	root.SetOut(io.Discard)
	root.SetArgs([]string{"--base", filepath.Join(dir, "base"), "--head", filepath.Join(dir, "head"),
		"--applications", "manifests/applications.yaml", "--rendered", rendered, "--output", html, "--summary", summary})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("a stale file under --rendered survived")
	}
	md, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"**1 manifest changed**", "| `web.stage.c1` | `stage` | `c1` |"} {
		if !strings.Contains(string(md), want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}
	head, err := os.ReadFile(filepath.Join(rendered, "head", "web.stage.c1", "shop", "Deployment", "web.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(head), "replicas: 2") {
		t.Errorf("rendered head tree:\n%s", head)
	}
	if _, err := os.Stat(html); err != nil {
		t.Error(err)
	}
}

func TestRootRefusesPerTreeApplicationsWhenHydrating(t *testing.T) {
	root := New()
	root.SetOut(io.Discard)
	root.SetArgs([]string{"--base", t.TempDir(), "--head", t.TempDir(),
		"--applications", "manifests/applications.yaml", "--head-applications", "x.yaml"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "head-applications") {
		t.Errorf("want a mutually exclusive flags error, got %v", err)
	}
}
