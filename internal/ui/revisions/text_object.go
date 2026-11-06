package revisions

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	textObjectNode        = "node"
	textObjectChangeId    = "change_id"
	textObjectCommitId    = "commit_id"
	textObjectAuthor      = "author"
	textObjectTimestamp   = "timestamp"
	textObjectBookmark    = "bookmark"
	textObjectTag         = "tag"
	textObjectDescription = "description"
	textObjectDivergent   = "divergent"
)

type textObject struct {
	Kind  string
	Value string
	// Index is the ordinal of the object among objects of the same kind.
	Index int
	// Remote and Conflicted are only set for bookmarks and tags.
	Remote     string
	Conflicted bool
	line       int
	x          int
	// width spans the object on its first line.
	width int
}

// textObjectKind maps a `--color=debug` labelled segment to the kind of text
// object it belongs to.
func textObjectKind(segment *screen.Segment) string {
	switch {
	case segment.HasLabel("node"):
		return textObjectNode
	case segment.HasLabel("change_id"), segment.HasLabel("change_offset"):
		return textObjectChangeId
	case segment.HasLabel("commit_id"):
		return textObjectCommitId
	case segment.HasLabel("timestamp"):
		return textObjectTimestamp
	case segment.HasLabel("author"):
		return textObjectAuthor
	case segment.HasLabel("bookmarks"), segment.HasLabel("local_bookmarks"), segment.HasLabel("remote_bookmarks"):
		return textObjectBookmark
	case segment.HasLabel("tags"):
		return textObjectTag
	case segment.HasLabel("description"):
		return textObjectDescription
	case segment.HasLabel("divergent"):
		return textObjectDivergent
	}
	return ""
}

func textObjectsForRow(row parser.Row) []textObject {
	var objects []textObject
	counts := map[string]int{}
	last := func() *textObject {
		if len(objects) == 0 {
			return nil
		}
		return &objects[len(objects)-1]
	}
	add := func(kind, text string, line, x int) {
		objects = append(objects, textObject{Kind: kind, Value: text, Index: counts[kind], line: line, x: x, width: render.StringWidth(text)})
		counts[kind]++
	}
	extend := func(o *textObject, line, start int, text string) {
		if o.line == line {
			o.width = start + render.StringWidth(text) - o.x
		}
	}

	gap := false
	for lineIndex, line := range row.Lines {
		x := 0
		segments := append(append([]*screen.Segment{}, line.Gutter.Segments...), line.Segments...)
		for _, segment := range segments {
			width := render.StringWidth(segment.Text)
			kind := textObjectKind(segment)
			text := strings.TrimSpace(segment.Text)
			if kind == "" || text == "" {
				gap = gap || segment.Text != ""
				x += width
				continue
			}
			joinWithSpace := gap
			gap = false
			start := x + render.StringWidth(segment.Text[:strings.Index(segment.Text, text)])
			x += width

			// Bookmarks and tags render as a name followed by @remote and
			// markers such as ?? or *.
			if kind == textObjectBookmark || kind == textObjectTag {
				prev := last()
				switch {
				case segment.HasLabel("name"):
					add(kind, text, lineIndex, start)
					objects[len(objects)-1].Value = jj.UnquoteRefName(text)
				case prev == nil || prev.Kind != kind:
				default:
					extend(prev, lineIndex, start, text)
					if segment.HasLabel("remote") {
						prev.Remote = jj.UnquoteRefName(text)
					} else if text == "??" {
						prev.Conflicted = true
					}
				}
				continue
			}
			// Consecutive spans of the same kind (e.g. the local part, @ and the
			// domain of an email) form one object.
			if prev := last(); prev != nil && prev.Kind == kind && kind != textObjectNode {
				if prev.line != lineIndex {
					prev.Value += "\n"
				} else if joinWithSpace {
					prev.Value += " "
				}
				prev.Value += text
				extend(prev, lineIndex, start, text)
				continue
			}
			add(kind, text, lineIndex, start)
		}
	}
	return objects
}

// resolveTextObject finds the object matching the wanted kind and index,
// falling back to another object of the same kind and then to the node.
func resolveTextObject(objects []textObject, kind string, index int) *textObject {
	var sameKind, node *textObject
	for i := range objects {
		o := &objects[i]
		if o.Kind == kind && o.Index == index {
			return o
		}
		if o.Kind == kind && sameKind == nil {
			sameKind = o
		}
		if o.Kind == textObjectNode && node == nil {
			node = o
		}
	}
	if sameKind != nil {
		return sameKind
	}
	return node
}

func textObjectCursor() *tea.Cursor {
	cursor := tea.NewCursor(0, 0)
	cursor.Shape = tea.CursorBlock
	cursor.Blink = true
	return cursor
}
