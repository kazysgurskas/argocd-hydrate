package hydrate

import (
	"strings"
	"testing"
)

const stream = `---
# Source: web/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: shop
  annotations:
    note: 'a line with --- in the middle is kept'
spec:
  replicas: 2
---
apiVersion: v1
kind: Namespace
metadata:
  name: shop
---
apiVersion: batch/v1
kind: Job
metadata:
  generateName: migrate-
  namespace: shop
---
# A document without a kind is skipped.
metadata:
  name: nothing
---
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: shop
---
apiVersion: v1
kind: Secret
metadata:
  name: web-env
  namespace: shop
type: Opaque
data:
  password: cGFzc3dvcmQ=
stringData:
  token: plain
`

func TestParseManifestsSplitsAndIdentifies(t *testing.T) {
	manifests, err := parseManifests(stream)
	if err != nil {
		t.Fatal(err)
	}
	want := []Manifest{
		{Kind: "Deployment", Name: "web", Namespace: "shop"},
		{Kind: "Namespace", Name: "shop"},
		{Kind: "Job", Name: "migrate-generated", Namespace: "shop"},
		{Kind: "Secret", Name: "web-env", Namespace: "shop"},
	}
	if len(manifests) != len(want) {
		t.Fatalf("got %d manifests, want %d: %+v", len(manifests), len(want), manifests)
	}
	for i, w := range want {
		got := manifests[i]
		if got.Kind != w.Kind || got.Name != w.Name || got.Namespace != w.Namespace {
			t.Errorf("manifest %d = %s %s/%s, want %s %s/%s", i, got.Kind, got.Namespace, got.Name, w.Kind, w.Namespace, w.Name)
		}
		if !strings.HasPrefix(got.Content, "---\n") || !strings.HasSuffix(got.Content, "\n") || strings.HasSuffix(got.Content, "\n\n") {
			t.Errorf("manifest %d content is not one document ending in a single newline: %q", i, got.Content)
		}
	}
	// A document that is not a Secret is kept as written, comment included.
	deployment := "---\n# Source: web/templates/deployment.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  namespace: shop\n  annotations:\n    note: 'a line with --- in the middle is kept'\nspec:\n  replicas: 2\n"
	if manifests[0].Content != deployment {
		t.Errorf("deployment rewritten:\n%s", manifests[0].Content)
	}
}

func TestParseManifestsRedactsSecrets(t *testing.T) {
	manifests, err := parseManifests(stream)
	if err != nil {
		t.Fatal(err)
	}
	secret := manifests[len(manifests)-1].Content
	for _, leaked := range []string{"cGFzc3dvcmQ=", "plain"} {
		if strings.Contains(secret, leaked) {
			t.Errorf("secret value %q leaked:\n%s", leaked, secret)
		}
	}
	for _, kept := range []string{"password: '***REDACTED***'", "token: '***REDACTED***'", "type: Opaque"} {
		if !strings.Contains(secret, kept) {
			t.Errorf("missing %q in:\n%s", kept, secret)
		}
	}
}

func TestParseManifestsRejectsBadYAML(t *testing.T) {
	if _, err := parseManifests("---\nkind: [\n"); err == nil {
		t.Error("expected a parse error")
	}
}

func TestParseManifestsEmptyStream(t *testing.T) {
	manifests, err := parseManifests("")
	if err != nil || len(manifests) != 0 {
		t.Errorf("got %v, %v for an empty stream", manifests, err)
	}
}

func TestManifestPath(t *testing.T) {
	tests := []struct {
		manifest Manifest
		want     string
	}{
		{Manifest{Kind: "Deployment", Name: "web", Namespace: "shop"}, "shop/Deployment/web.yaml"},
		{Manifest{Kind: "Namespace", Name: "shop"}, "Namespace/shop.yaml"},
		{Manifest{Kind: "ClusterRole", Name: "system:node"}, "ClusterRole/system-node.yaml"},
		{Manifest{Kind: "Ingress", Name: "www.example.com", Namespace: "a b"}, "a-b/Ingress/www-example-com.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.manifest.Path(); got != tt.want {
				t.Errorf("Path() = %q, want %q", got, tt.want)
			}
		})
	}
}
