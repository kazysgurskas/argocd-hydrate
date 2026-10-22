package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kazysgurskas/argocd-diff/internal/diff"
)

func TestRenderEmbedsDataAndEscapes(t *testing.T) {
	result := &diff.Result{
		Files: []diff.File{{
			Path:   "app.stage.c1/ns/ConfigMap/x.yaml",
			Status: diff.StatusModified,
			Hunks: []diff.Hunk{{BaseStart: 1, BaseLines: 1, HeadStart: 1, HeadLines: 1, Lines: []diff.Line{
				{Type: diff.LineDeleted, Base: 1, Text: "script: </script><b>&"},
				{Type: diff.LineAdded, Head: 1, Text: "script: ok"},
			}}},
			Additions: 1, Deletions: 1,
		}},
		Additions: 1, Deletions: 1, Modified: 1,
	}
	meta := Meta{Title: "repo #1 <x>", Repo: "owner/repo", PRNumber: 1, GeneratedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}

	var out bytes.Buffer
	if err := Render(&out, result, meta); err != nil {
		t.Fatal(err)
	}
	html := out.String()

	// The data block must not be able to close the script element.
	if strings.Contains(html, "</script><b>") {
		t.Error("a line containing </script> was embedded unescaped")
	}
	if !strings.Contains(html, `<title>repo #1 &lt;x&gt; - manifest diff</title>`) {
		t.Error("title is not HTML-escaped")
	}

	// The embedded JSON round-trips.
	start := strings.Index(html, `<script type="application/json" id="diff-data">`)
	if start < 0 {
		t.Fatal("data script tag missing")
	}
	start += len(`<script type="application/json" id="diff-data">`)
	end := strings.Index(html[start:], "</script>")
	var got struct {
		Meta   Meta        `json:"meta"`
		Result diff.Result `json:"result"`
	}
	if err := json.Unmarshal([]byte(html[start:start+end]), &got); err != nil {
		t.Fatalf("embedded JSON does not parse: %v", err)
	}
	if got.Meta.Repo != "owner/repo" || len(got.Result.Files) != 1 || got.Result.Files[0].Hunks[0].Lines[0].Text != "script: </script><b>&" {
		t.Errorf("round trip lost data: %+v", got)
	}
}

func TestRenderNilResult(t *testing.T) {
	var out bytes.Buffer
	if err := Render(&out, nil, Meta{Title: "t"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"files":null`) && !strings.Contains(out.String(), `"files":[]`) {
		t.Error("nil result should render as an empty file list")
	}
}
