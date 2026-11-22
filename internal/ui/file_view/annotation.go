package file_view

import (
	"bytes"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/screen"
)

// line is one line of `jj file annotate`, printed with the user's
// templates.file_annotate and split by the labels jj attaches to its fields.
type line struct {
	// changeId and commitId are empty when the template does not print them.
	changeId string
	commitId string
	number   int
	content  string
	// gutter is what the template prints before the content. When it repeats
	// the line above, all but the line number and what follows is blanked.
	gutter string
	// first marks the line where the gutter changes.
	first bool
}

func parseAnnotation(output []byte) []line {
	var (
		lines    []line
		previous string
	)
	for segments := range screen.BreakNewLinesIter(screen.ParseLabelledFromReader(bytes.NewReader(output))) {
		l, key := parseLine(segments)
		l.first = len(lines) == 0 || key != previous
		previous = key
		if !l.first {
			l.gutter = blankBeforeNumber(segments)
		}
		lines = append(lines, l)
	}
	return lines
}

// parseLine returns the line and its gutter without the line number, which
// tells whether the gutter repeats the line above.
func parseLine(segments []*screen.Segment) (line, string) {
	var (
		l       line
		gutter  strings.Builder
		key     strings.Builder
		content strings.Builder
		inBody  bool
	)
	for _, s := range segments {
		has := func(label string) bool { return slices.Contains(s.Labels, label) }
		if has("content") {
			inBody = true
		}
		if inBody {
			content.WriteString(s.Text)
			continue
		}
		gutter.WriteString(s.String())
		switch {
		case has("line_number"):
			l.number, _ = strconv.Atoi(strings.TrimSpace(s.Text))
			continue
		case has("change_id"):
			l.changeId += s.Text
		case has("commit_id"):
			l.commitId += s.Text
		}
		key.WriteString(s.Text)
	}
	if !inBody {
		return line{content: plain(segments)}, ""
	}
	l.gutter = gutter.String()
	l.content = content.String()
	return l, key.String()
}

// blankBeforeNumber renders the gutter with everything before the line
// number replaced by spaces of the same width.
func blankBeforeNumber(segments []*screen.Segment) string {
	var b strings.Builder
	number := false
	for _, s := range segments {
		if slices.Contains(s.Labels, "content") {
			break
		}
		number = number || slices.Contains(s.Labels, "line_number")
		if number {
			b.WriteString(s.String())
		} else {
			b.WriteString(strings.Repeat(" ", ansi.StringWidth(s.Text)))
		}
	}
	return b.String()
}

func plain(segments []*screen.Segment) string {
	var b strings.Builder
	for _, s := range segments {
		b.WriteString(s.Text)
	}
	return b.String()
}

// id is what the line's change is selected by, preferring the change id.
func (l line) id() string {
	if l.changeId != "" {
		return l.changeId
	}
	return l.commitId
}
