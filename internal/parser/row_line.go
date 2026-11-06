package parser

import (
	"strings"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
)

type GraphRowLine struct {
	Segments []*screen.Segment
	Gutter   GraphGutter
	Flags    RowLineFlags
}

func (gr *GraphRowLine) GetSegments() []*screen.Segment {
	return gr.Segments
}

func NewGraphRowLine(segments []*screen.Segment) GraphRowLine {
	return GraphRowLine{
		Segments: segments,
		Gutter:   GraphGutter{Segments: make([]*screen.Segment, 0)},
	}
}

func (gr *GraphRowLine) ParseRowPrefixes() (int, *jj.Commit) {
	prefixesIdx := -1
	for i, segment := range gr.Segments {
		if strings.Contains(segment.Text, jj.JJUIPrefix) {
			prefixesIdx = i
			break
		}
	}

	if prefixesIdx == -1 {
		return -1, nil
	}
	prefixParts := strings.Split(gr.Segments[prefixesIdx].Text, jj.JJUIPrefix)
	if len(prefixParts) != 3 && len(prefixParts) != 4 && len(prefixParts) != 5 && len(prefixParts) != 6 && len(prefixParts) != 7 && len(prefixParts) != 8 {
		return -1, nil
	}
	beforePrefix := prefixParts[0]
	commit := &jj.Commit{
		ChangeId: strings.TrimSpace(prefixParts[1]),
		CommitId: strings.TrimSpace(prefixParts[2]),
	}
	if len(prefixParts) == 6 || len(prefixParts) == 7 || len(prefixParts) == 8 {
		commit.HasBookmarks = strings.TrimSpace(prefixParts[3]) == "true"
		commit.HasLocalBookmarks = strings.TrimSpace(prefixParts[4]) == "true"
		commit.HasWorkspace = strings.TrimSpace(prefixParts[5]) == "true"
	}
	switch len(prefixParts) {
	case 4:
		commit.Root = strings.TrimSpace(prefixParts[3]) == "true"
	case 5:
		commit.Root = strings.TrimSpace(prefixParts[3]) == "true"
		commit.IsEmpty = strings.TrimSpace(prefixParts[4]) == "true"
	case 7, 8:
		commit.Root = strings.TrimSpace(prefixParts[6]) == "true"
		if len(prefixParts) == 8 {
			commit.IsEmpty = strings.TrimSpace(prefixParts[7]) == "true"
		}
	}

	// Remove the structured metadata, while keeping everything before the
	// prefixes.
	gr.Segments[prefixesIdx] = &screen.Segment{Text: beforePrefix}

	return prefixesIdx + 1, commit
}

func (gr *GraphRowLine) chop(indent int) {
	if len(gr.Segments) == 0 {
		return
	}
	segments := gr.Segments
	gr.Segments = make([]*screen.Segment, 0)

	for i, s := range segments {
		extended := screen.Segment{
			Style:  s.Style,
			Labels: s.Labels,
		}
		var textBuilder strings.Builder
		for _, p := range s.Text {
			if indent <= 0 {
				break
			}
			textBuilder.WriteRune(p)
			indent--
		}
		extended.Text = textBuilder.String()
		gr.Gutter.Segments = append(gr.Gutter.Segments, &extended)
		if len(extended.Text) < len(s.Text) {
			gr.Segments = append(gr.Segments, &screen.Segment{
				Text:   s.Text[len(extended.Text):],
				Style:  s.Style,
				Labels: s.Labels,
			})
		}
		if indent <= 0 && len(segments)-i-1 > 0 {
			gr.Segments = segments[i+1:]
			break
		}
	}

	// break gutter into segments per rune
	segments = gr.Gutter.Segments
	gr.Gutter.Segments = make([]*screen.Segment, 0)
	for _, s := range segments {
		for _, p := range s.Text {
			extended := screen.Segment{
				Text:   string(p),
				Style:  s.Style,
				Labels: s.Labels,
			}
			gr.Gutter.Segments = append(gr.Gutter.Segments, &extended)
		}
	}

	// Pad with spaces if indent is not fully consumed
	if indent > 0 && len(gr.Gutter.Segments) > 0 {
		style := gr.Gutter.Segments[len(gr.Gutter.Segments)-1].Style
		for range indent {
			gr.Gutter.Segments = append(gr.Gutter.Segments, &screen.Segment{
				Text:  " ",
				Style: style,
			})
		}
	}
}

func (gr *GraphRowLine) containsRune(r rune) bool {
	for _, segment := range gr.Gutter.Segments {
		if strings.ContainsRune(segment.Text, r) {
			return true
		}
	}
	return false
}

// ParseRevisionNode detects a revision line in `--color=debug` output by its
// graph node label and returns the index of the first content segment.
func (gr *GraphRowLine) ParseRevisionNode() (int, *jj.Commit) {
	for i, segment := range gr.Segments {
		if !segment.HasLabel("node") {
			continue
		}
		if segment.HasLabel("elided") {
			return -1, nil
		}
		for j := i + 1; j < len(gr.Segments); j++ {
			if len(gr.Segments[j].Labels) > 0 {
				return j, &jj.Commit{
					IsWorkingCopy: segment.HasLabel("working_copy"),
					IsImmutable:   segment.HasLabel("immutable"),
				}
			}
		}
		return -1, nil
	}
	return -1, nil
}

// applyLabels fills in commit metadata from the labelled segments of a line.
// The first change id and commit id rendered for the revision win.
func applyLabels(commit *jj.Commit, segments []*screen.Segment) {
	// Ids are reduced to their `prefix` span so they match the
	// `shortest()` ids used by jjui's other commands.
	idOf := func(start int, label string) (string, int) {
		var full, prefix strings.Builder
		i := start
		for ; i < len(segments) && segments[i].HasLabel(label); i++ {
			full.WriteString(segments[i].Text)
			if segments[i].HasLabel("prefix") {
				prefix.WriteString(segments[i].Text)
			}
		}
		if prefix.Len() > 0 {
			return prefix.String(), i
		}
		return full.String(), i
	}
	for i := 0; i < len(segments); i++ {
		s := segments[i]
		switch {
		case s.HasLabel("change_id") && commit.ChangeId == "":
			id, next := idOf(i, "change_id")
			if next < len(segments) && segments[next].HasLabel("change_offset") {
				id += segments[next].Text
				next++
			}
			commit.ChangeId = id
			i = next - 1
			continue
		case s.HasLabel("commit_id") && commit.CommitId == "":
			id, next := idOf(i, "commit_id")
			commit.CommitId = id
			i = next - 1
			continue
		}
		if s.HasLabel("bookmarks") && s.HasLabel("name") {
			commit.HasBookmarks = true
			isRemote := i+1 < len(segments) && segments[i+1].HasLabel("bookmarks") && segments[i+1].Text == "@"
			if !isRemote {
				commit.HasLocalBookmarks = true
			}
		}
		if s.HasLabel("working_copies") || s.HasLabel("working_copy") {
			commit.HasWorkspace = true
		}
		if s.HasLabel("root") {
			commit.Root = true
		}
		if s.HasLabel("empty") {
			commit.IsEmpty = true
		}
		if s.HasLabel("hidden") {
			commit.Hidden = true
		}
	}
}
