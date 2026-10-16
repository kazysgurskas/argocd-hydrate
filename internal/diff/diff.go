// Package diff compares two trees of rendered Kubernetes manifests, as
// argocd-hydrate writes them, and describes every changed file as a list of
// hunks with per-line metadata, ready for a report.
package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// Status is what happened to a file between base and head.
type Status string

const (
	StatusAdded    Status = "added"
	StatusDeleted  Status = "deleted"
	StatusModified Status = "modified"
)

// LineType is the role of a line within a hunk.
type LineType string

const (
	LineContext LineType = "ctx"
	LineAdded   LineType = "add"
	LineDeleted LineType = "del"
)

// Line is one line of a hunk. Base and Head are 1-based line numbers in the
// respective file, 0 when the line does not exist there. Highlight marks the
// byte range of Text that differs from the paired line on the other side, for
// modified lines whose beginning and end are unchanged.
type Line struct {
	Type      LineType `json:"t"`
	Base      int      `json:"b,omitempty"`
	Head      int      `json:"h,omitempty"`
	Text      string   `json:"x"`
	Highlight []int    `json:"hl,omitempty"`
}

// Hunk is a run of changed lines with surrounding context, as in a unified
// diff. The starts are 1-based line numbers; the counts are lines covered on
// each side.
type Hunk struct {
	BaseStart int    `json:"bs"`
	BaseLines int    `json:"bl"`
	HeadStart int    `json:"hs"`
	HeadLines int    `json:"hl"`
	Lines     []Line `json:"lines"`
}

// File is one changed manifest.
type File struct {
	// Path is the file's path relative to the tree root, with forward
	// slashes, the same in base and head.
	Path      string    `json:"path"`
	Status    Status    `json:"status"`
	Resource  *Resource `json:"resource,omitempty"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
	Hunks     []Hunk    `json:"hunks"`
	// BaseText and HeadText carry the whole file for modified files small
	// enough (see Options.FullTextLimit), so a viewer can expand context
	// between hunks. Nil otherwise.
	BaseText []string `json:"base,omitempty"`
	HeadText []string `json:"head,omitempty"`
}

// Result is the comparison of two trees.
type Result struct {
	Files     []File `json:"files"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Added     int    `json:"added"`
	Deleted   int    `json:"deleted"`
	Modified  int    `json:"modified"`
}

// Options tune a comparison.
type Options struct {
	// Context is the number of unchanged lines kept around each change.
	// Zero means DefaultContext.
	Context int
	// FullTextLimit is the largest file, in lines, whose full text is kept
	// on modified files. Zero means DefaultFullTextLimit; negative means
	// never.
	FullTextLimit int
}

const (
	DefaultContext       = 3
	DefaultFullTextLimit = 2000
)

// Compare diffs the manifest trees at baseDir and headDir. Only .yaml and
// .yml files take part; everything else is ignored. Files are listed in
// path order.
func Compare(baseDir, headDir string, opts Options) (*Result, error) {
	context := opts.Context
	if context <= 0 {
		context = DefaultContext
	}
	fullTextLimit := opts.FullTextLimit
	if fullTextLimit == 0 {
		fullTextLimit = DefaultFullTextLimit
	}

	base, err := readTree(baseDir)
	if err != nil {
		return nil, fmt.Errorf("read base tree: %w", err)
	}
	head, err := readTree(headDir)
	if err != nil {
		return nil, fmt.Errorf("read head tree: %w", err)
	}

	paths := make(map[string]struct{}, len(base)+len(head))
	for p := range base {
		paths[p] = struct{}{}
	}
	for p := range head {
		paths[p] = struct{}{}
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	result := &Result{}
	for _, p := range sorted {
		b, inBase := base[p]
		h, inHead := head[p]
		if inBase && inHead && b == h {
			continue
		}
		f := File{Path: p}
		switch {
		case !inBase:
			f.Status = StatusAdded
			f.Hunks = []Hunk{wholeFileHunk(splitLines(h), LineAdded)}
			f.Resource = resourceFor(p, h)
			result.Added++
		case !inHead:
			f.Status = StatusDeleted
			f.Hunks = []Hunk{wholeFileHunk(splitLines(b), LineDeleted)}
			f.Resource = resourceFor(p, b)
			result.Deleted++
		default:
			f.Status = StatusModified
			bl, hl := splitLines(b), splitLines(h)
			f.Hunks = hunks(bl, hl, context)
			f.Resource = resourceFor(p, h)
			if fullTextLimit > 0 && len(bl) <= fullTextLimit && len(hl) <= fullTextLimit {
				f.BaseText, f.HeadText = bl, hl
			}
			result.Modified++
		}
		for _, hk := range f.Hunks {
			for _, l := range hk.Lines {
				switch l.Type {
				case LineAdded:
					f.Additions++
				case LineDeleted:
					f.Deletions++
				}
			}
		}
		result.Additions += f.Additions
		result.Deletions += f.Deletions
		result.Files = append(result.Files, f)
	}
	return result, nil
}

// readTree returns every manifest under root, keyed by slash-separated
// relative path.
func readTree(root string) (map[string]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	files := make(map[string]string)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isManifest(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	return files, err
}

func isManifest(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

// splitLines splits text into lines without a trailing empty element for the
// final newline.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// wholeFileHunk is the single hunk of an added or deleted file.
func wholeFileHunk(lines []string, t LineType) Hunk {
	h := Hunk{Lines: make([]Line, 0, len(lines))}
	for i, text := range lines {
		l := Line{Type: t, Text: text}
		if t == LineAdded {
			l.Head = i + 1
		} else {
			l.Base = i + 1
		}
		h.Lines = append(h.Lines, l)
	}
	if t == LineAdded {
		h.HeadStart, h.HeadLines = 1, len(lines)
	} else {
		h.BaseStart, h.BaseLines = 1, len(lines)
	}
	return h
}

// hunks computes the unified-diff hunks between two line slices.
func hunks(base, head []string, context int) []Hunk {
	matcher := difflib.NewMatcher(base, head)
	var out []Hunk
	for _, group := range matcher.GetGroupedOpCodes(context) {
		if len(group) == 0 {
			continue
		}
		first, last := group[0], group[len(group)-1]
		h := Hunk{
			BaseStart: first.I1 + 1,
			BaseLines: last.I2 - first.I1,
			HeadStart: first.J1 + 1,
			HeadLines: last.J2 - first.J1,
		}
		for _, op := range group {
			switch op.Tag {
			case 'e':
				for k := 0; k < op.I2-op.I1; k++ {
					h.Lines = append(h.Lines, Line{Type: LineContext, Base: op.I1 + k + 1, Head: op.J1 + k + 1, Text: base[op.I1+k]})
				}
			case 'd':
				h.Lines = append(h.Lines, deletedLines(base, op.I1, op.I2)...)
			case 'i':
				h.Lines = append(h.Lines, addedLines(head, op.J1, op.J2)...)
			case 'r':
				del := deletedLines(base, op.I1, op.I2)
				add := addedLines(head, op.J1, op.J2)
				highlightPairs(del, add)
				h.Lines = append(h.Lines, del...)
				h.Lines = append(h.Lines, add...)
			}
		}
		out = append(out, h)
	}
	return out
}

func deletedLines(base []string, from, to int) []Line {
	lines := make([]Line, 0, to-from)
	for i := from; i < to; i++ {
		lines = append(lines, Line{Type: LineDeleted, Base: i + 1, Text: base[i]})
	}
	return lines
}

func addedLines(head []string, from, to int) []Line {
	lines := make([]Line, 0, to-from)
	for j := from; j < to; j++ {
		lines = append(lines, Line{Type: LineAdded, Head: j + 1, Text: head[j]})
	}
	return lines
}

// highlightPairs marks, on each deleted line and the added line at the same
// position in the replacement, the span that changed: what is left once the
// common prefix and suffix are removed. Lines with nothing in common, or
// identical lines, get no highlight.
func highlightPairs(del, add []Line) {
	n := len(del)
	if len(add) < n {
		n = len(add)
	}
	for i := 0; i < n; i++ {
		a, b := del[i].Text, add[i].Text
		prefix := commonPrefix(a, b)
		suffix := commonSuffix(a[prefix:], b[prefix:])
		if prefix == 0 && suffix == 0 {
			continue
		}
		if prefix+suffix >= len(a) && prefix+suffix >= len(b) {
			continue
		}
		del[i].Highlight = []int{prefix, len(a) - suffix}
		add[i].Highlight = []int{prefix, len(b) - suffix}
	}
}

func commonPrefix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

func commonSuffix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	return n
}
