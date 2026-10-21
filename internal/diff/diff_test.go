package diff

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const deployment = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: shop
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: web
        image: nginx:1.27
`

func TestCompare(t *testing.T) {
	base, head := t.TempDir(), t.TempDir()
	writeTree(t, base, map[string]string{
		"web.prod.euw2/shop/Deployment/web.yaml": deployment,
		"web.prod.euw2/shop/Service/web.yaml":    "apiVersion: v1\nkind: Service\nmetadata:\n  name: web\n  namespace: shop\n",
		"web.prod.euw2/Namespace/shop.yaml":      "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: shop\n",
		"web.prod.euw2/notes.txt":                "ignored",
	})
	modified := deployment[:len(deployment)-len("        image: nginx:1.27\n")] + "        image: nginx:1.28\n"
	writeTree(t, head, map[string]string{
		"web.prod.euw2/shop/Deployment/web.yaml": modified,
		"web.prod.euw2/shop/HTTPRoute/web.yaml":  "apiVersion: gateway.networking.k8s.io/v1\nkind: HTTPRoute\nmetadata:\n  name: web\n  namespace: shop\n",
		"web.prod.euw2/Namespace/shop.yaml":      "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: shop\n",
		"web.prod.euw2/notes.txt":                "ignored but different",
	})

	r, err := Compare(base, head, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(r.Files); got != 3 {
		t.Fatalf("files = %d, want 3 (modified, added, deleted)", got)
	}
	if r.Added != 1 || r.Modified != 1 || r.Deleted != 1 {
		t.Errorf("counts = %d added, %d modified, %d deleted", r.Added, r.Modified, r.Deleted)
	}

	// Files are in path order.
	byPath := map[string]File{}
	for _, f := range r.Files {
		byPath[f.Path] = f
	}
	dep := byPath["web.prod.euw2/shop/Deployment/web.yaml"]
	if dep.Status != StatusModified || dep.Additions != 1 || dep.Deletions != 1 {
		t.Errorf("deployment: status %s +%d -%d", dep.Status, dep.Additions, dep.Deletions)
	}
	if len(dep.Hunks) != 1 {
		t.Fatalf("deployment hunks = %d", len(dep.Hunks))
	}
	hk := dep.Hunks[0]
	if hk.BaseStart != 10 || hk.HeadStart != 10 {
		t.Errorf("hunk starts = %d/%d, want 10/10", hk.BaseStart, hk.HeadStart)
	}
	var del, add *Line
	for i := range hk.Lines {
		switch hk.Lines[i].Type {
		case LineDeleted:
			del = &hk.Lines[i]
		case LineAdded:
			add = &hk.Lines[i]
		}
	}
	if del == nil || add == nil {
		t.Fatal("expected one deleted and one added line")
	}
	if del.Base != 13 || add.Head != 13 {
		t.Errorf("line numbers: del base %d, add head %d", del.Base, add.Head)
	}
	// "        image: nginx:1.27" vs "...1.28": only the last digit differs.
	if !slices.Equal(del.Highlight, []int{24, 25}) || !slices.Equal(add.Highlight, []int{24, 25}) {
		t.Errorf("highlights = %v / %v", del.Highlight, add.Highlight)
	}
	if dep.BaseText == nil || dep.HeadText == nil {
		t.Error("modified file should carry full text under the limit")
	}
	if dep.Resource == nil || dep.Resource.App != "web" || dep.Resource.Environment != "prod" || dep.Resource.Cluster != "euw2" ||
		dep.Resource.Kind != "Deployment" || dep.Resource.Namespace != "shop" || dep.Resource.Name != "web" || dep.Resource.APIVersion != "apps/v1" {
		t.Errorf("resource = %+v", dep.Resource)
	}

	added := byPath["web.prod.euw2/shop/HTTPRoute/web.yaml"]
	if added.Status != StatusAdded || added.Additions != 5 || added.Deletions != 0 || len(added.Hunks) != 1 {
		t.Errorf("added: %+v", added)
	}
	if added.Hunks[0].Lines[0].Head != 1 || added.Hunks[0].Lines[0].Base != 0 {
		t.Errorf("added line numbering: %+v", added.Hunks[0].Lines[0])
	}
	deleted := byPath["web.prod.euw2/shop/Service/web.yaml"]
	if deleted.Status != StatusDeleted || deleted.Deletions != 5 || deleted.Resource.Kind != "Service" {
		t.Errorf("deleted: %+v", deleted)
	}
	if r.Additions != 6 || r.Deletions != 6 {
		t.Errorf("totals = +%d -%d", r.Additions, r.Deletions)
	}
}

func TestCompareIdentical(t *testing.T) {
	base, head := t.TempDir(), t.TempDir()
	writeTree(t, base, map[string]string{"a/Kind/x.yaml": "kind: X\n"})
	writeTree(t, head, map[string]string{"a/Kind/x.yaml": "kind: X\n"})
	r, err := Compare(base, head, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Files) != 0 {
		t.Errorf("files = %v, want none", r.Files)
	}
}

func TestFullTextLimit(t *testing.T) {
	base, head := t.TempDir(), t.TempDir()
	writeTree(t, base, map[string]string{"a/Kind/x.yaml": "kind: X\na: 1\n"})
	writeTree(t, head, map[string]string{"a/Kind/x.yaml": "kind: X\na: 2\n"})
	r, err := Compare(base, head, Options{Context: DefaultContext, FullTextLimit: 0})
	if err != nil {
		t.Fatal(err)
	}
	if r.Files[0].BaseText != nil {
		t.Error("full text kept despite a zero limit")
	}
	r, err = Compare(base, head, Options{Context: DefaultContext, FullTextLimit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Files[0].BaseText != nil {
		t.Error("full text kept for a file above the limit")
	}
	r, err = Compare(base, head, Options{Context: DefaultContext, FullTextLimit: -1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Files[0].BaseText == nil {
		t.Error("a negative limit should mean the default, which keeps a two-line file")
	}
}

func TestZeroContext(t *testing.T) {
	base, head := t.TempDir(), t.TempDir()
	writeTree(t, base, map[string]string{"a/Kind/x.yaml": "kind: X\na: 1\nb: 2\nc: 3\n"})
	writeTree(t, head, map[string]string{"a/Kind/x.yaml": "kind: X\na: 1\nb: 20\nc: 3\n"})
	r, err := Compare(base, head, Options{Context: 0})
	if err != nil {
		t.Fatal(err)
	}
	hk := r.Files[0].Hunks[0]
	for _, l := range hk.Lines {
		if l.Type == LineContext {
			t.Fatalf("context line %q in a zero-context diff", l.Text)
		}
	}
	if hk.BaseStart != 3 || hk.BaseLines != 1 || hk.HeadStart != 3 || hk.HeadLines != 1 {
		t.Errorf("hunk = %+v", hk)
	}
}

func TestHighlightPairs(t *testing.T) {
	tests := []struct {
		a, b         string
		wantA, wantB []int
	}{
		{"image: nginx:1.27", "image: nginx:1.28", []int{16, 17}, []int{16, 17}},
		{"replicas: 2", "replicas: 10", []int{10, 11}, []int{10, 12}},
		{"abc", "xyz", nil, nil},   // nothing in common
		{"same", "same", nil, nil}, // nothing changed
		{"key: a", "key: ab", []int{6, 6}, []int{6, 7}},
		// Offsets count code points: "café" is four of them, not five bytes.
		{"description: café au lait", "description: café au thé", []int{21, 25}, []int{21, 24}},
		{"note: 🚀 v1", "note: 🚀 v2", []int{9, 10}, []int{9, 10}},
	}
	for _, tt := range tests {
		t.Run(tt.a+" -> "+tt.b, func(t *testing.T) {
			del := []Line{{Text: tt.a}}
			add := []Line{{Text: tt.b}}
			highlightPairs(del, add)
			if !slices.Equal(del[0].Highlight, tt.wantA) || !slices.Equal(add[0].Highlight, tt.wantB) {
				t.Errorf("highlight = %v / %v, want %v / %v", del[0].Highlight, add[0].Highlight, tt.wantA, tt.wantB)
			}
		})
	}
}

func TestResourceFor(t *testing.T) {
	tests := []struct {
		path, content string
		want          Resource
	}{
		{"app.stage.c1/Kind/name.yaml", "", Resource{App: "app", Environment: "stage", Cluster: "c1", Kind: "Kind", Name: "name"}},
		{"app.stage.c1/ns/Kind/name.yml", "", Resource{App: "app", Environment: "stage", Cluster: "c1", Namespace: "ns", Kind: "Kind", Name: "name"}},
		{"plain-app/Kind/name.yaml", "", Resource{App: "plain-app", Kind: "Kind", Name: "name"}},
		{"app.stage.c1/Kind/name.yaml", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: real\n  namespace: real-ns\n",
			Resource{App: "app", Environment: "stage", Cluster: "c1", APIVersion: "v1", Kind: "ConfigMap", Namespace: "real-ns", Name: "real"}},
		{"odd.yaml", "kind: Thing\napiVersion: x/v1\nmetadata:\n  name: t\n", Resource{APIVersion: "x/v1", Kind: "Thing", Name: "t"}},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := resourceFor(tt.path, tt.content)
			if got == nil || *got != tt.want {
				t.Errorf("resourceFor(%q) = %+v, want %+v", tt.path, got, tt.want)
			}
		})
	}
	if resourceFor("x.yaml", "") != nil {
		t.Error("expected nil for a path with no structure and no body")
	}
}

func TestOpenTreeZip(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"a/Kind/x.yaml": "kind: X\n"})
	archive := filepath.Join(t.TempDir(), "tree.zip")
	zipDir(t, dir, archive)

	got, cleanup, err := OpenTree(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(got, "a", "Kind", "x.yaml")); err != nil {
		t.Errorf("extracted file missing: %v", err)
	}
	cleanup()
	if _, err := os.Stat(got); !os.IsNotExist(err) {
		t.Error("cleanup left the temporary directory")
	}

	if _, _, err := OpenTree(filepath.Join(dir, "a", "Kind", "x.yaml")); err == nil {
		t.Error("a plain file that is not a zip should be refused")
	}
}
