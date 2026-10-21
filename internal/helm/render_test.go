package helm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeChart writes a one-template chart that prints its values and the
// Kubernetes version it is rendered for.
func writeChart(t *testing.T, kubeVersion string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"Chart.yaml":  "apiVersion: v2\nname: tiny\nversion: 0.1.0\nkubeVersion: " + kubeVersion + "\n",
		"values.yaml": "greeting: hello\nnested:\n  a: 1\n  b: 2\n",
		"templates/configmap.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}
  namespace: {{ .Release.Namespace }}
data:
  greeting: {{ .Values.greeting }}
  a: {{ .Values.nested.a | quote }}
  b: {{ .Values.nested.b | quote }}
  kube: {{ .Capabilities.KubeVersion.Version }}
`,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func writeValues(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRenderMergesValueFilesInOrder(t *testing.T) {
	chart := writeChart(t, `""`)
	first := writeValues(t, "greeting: hi\nnested:\n  a: 10\n")
	second := writeValues(t, "nested:\n  b: 20\n")

	out, err := Render(chart, "rel", "ns", "1.35.6", []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"name: rel", "namespace: ns",
		"greeting: hi", `a: "10"`, `b: "20"`,
		"kube: v1.35.6",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderRejectsIncompatibleKubeVersion(t *testing.T) {
	chart := writeChart(t, `">=1.40.0"`)
	_, err := Render(chart, "rel", "ns", "1.35.6", nil)
	if err == nil || !strings.Contains(err.Error(), "--kube-version") {
		t.Errorf("want an error that says to pass --kube-version, got %v", err)
	}
}

func TestRenderRejectsMalformedKubeVersion(t *testing.T) {
	chart := writeChart(t, `""`)
	if _, err := Render(chart, "rel", "ns", "latest", nil); err == nil {
		t.Error("expected an error for a version that is not X.Y.Z")
	}
}
