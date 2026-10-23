// Package report renders a diff.Result as a self-contained HTML page: the
// data is embedded as JSON and a small script builds the viewer, so a page
// with hundreds of changed manifests stays small and renders lazily.
package report

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/kazysgurskas/argocd-diff/internal/diff"
)

//go:embed report.html
var page string

// Meta describes where the compared trees came from.
type Meta struct {
	Title       string    `json:"title"`
	Repo        string    `json:"repo,omitempty"`
	PRNumber    int       `json:"prNumber,omitempty"`
	PRTitle     string    `json:"prTitle,omitempty"`
	PRURL       string    `json:"prUrl,omitempty"`
	BaseRef     string    `json:"baseRef,omitempty"`
	HeadRef     string    `json:"headRef,omitempty"`
	Author      string    `json:"author,omitempty"`
	GeneratedAt time.Time `json:"generatedAt"`
	Tool        string    `json:"tool,omitempty"`
}

type payload struct {
	Meta   Meta         `json:"meta"`
	Result *diff.Result `json:"result"`
}

var tmpl = template.Must(template.New("report").Parse(page))

// Render writes the HTML page for result to w.
func Render(w io.Writer, result *diff.Result, meta Meta) error {
	if result == nil {
		result = &diff.Result{}
	}
	var data bytes.Buffer
	enc := json.NewEncoder(&data)
	enc.SetEscapeHTML(true) // "<" is written as <, so the JSON cannot close the script tag
	if err := enc.Encode(payload{Meta: meta, Result: result}); err != nil {
		return fmt.Errorf("encode diff data: %w", err)
	}
	return tmpl.Execute(w, map[string]any{
		"Title": meta.Title,
		"Data":  template.JS(data.String()), //nolint:gosec // JSON produced just above, HTML-escaped
	})
}
