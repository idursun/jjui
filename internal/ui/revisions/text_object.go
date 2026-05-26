package revisions

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	textObjectGraph       = "graph"
	textObjectAuthor      = "author"
	textObjectDate        = "date"
	textObjectBookmark    = "bookmark"
	textObjectDescription = "description"
)

type revisionTextObject struct {
	common.FocusedObject
	line  int
	x     int
	width int
}

func textObjectsForRow(row parser.Row) []revisionTextObject {
	if row.Commit == nil {
		return nil
	}

	objects := make([]revisionTextObject, 0, 5)
	if graph, ok := graphTextObject(row); ok {
		objects = append(objects, graph)
	}

	revisionLineIndex := -1
	for i, line := range row.Lines {
		if line.Flags&parser.Revision == parser.Revision {
			revisionLineIndex = i
			objects = append(objects, inlineTextObjects(row, line, i)...)
			break
		}
	}

	if desc, ok := descriptionTextObject(row, revisionLineIndex); ok {
		objects = append(objects, desc)
	}

	return objects
}

func graphTextObject(row parser.Row) (revisionTextObject, bool) {
	for lineIndex, line := range row.Lines {
		if line.Flags&parser.Revision != parser.Revision {
			continue
		}
		x := 0
		for _, segment := range line.Gutter.Segments {
			if strings.ContainsAny(segment.Text, "@○◆×") {
				return newRevisionTextObject(row, textObjectGraph, segment.Text, 0, lineIndex, x, render.StringWidth(segment.Text)), true
			}
			x += render.StringWidth(segment.Text)
		}
	}
	return revisionTextObject{}, false
}

func inlineTextObjects(row parser.Row, line *parser.GraphRowLine, lineIndex int) []revisionTextObject {
	objects := make([]revisionTextObject, 0, 4)
	x := gutterWidth(line)
	fieldOrdinal := 0
	bookmarkIndex := 0
	bookmarkPlaceholderX := x
	var inlineDescription *revisionTextObject
	seenCommitID := false

	for _, segment := range line.Segments {
		text := segment.Text
		trimmed := strings.TrimSpace(text)
		width := render.StringWidth(text)
		if trimmed == "" {
			x += width
			continue
		}
		if isCommitIdentifier(row, trimmed) {
			if strings.HasPrefix(trimmed, row.Commit.CommitId) {
				seenCommitID = true
			}
			x += width
			continue
		}
		if isDimmed(segment.Style) {
			x += width
			continue
		}

		switch {
		case isDefaultAuthorStyle(segment.Style):
			objects = append(objects, newRevisionTextObject(row, textObjectAuthor, trimmed, 0, lineIndex, x+leadingSpaceWidth(text), render.StringWidth(trimmed)))
			bookmarkPlaceholderX = x + width
		case isDefaultDateStyle(segment.Style):
			objects = append(objects, newRevisionTextObject(row, textObjectDate, trimmed, 0, lineIndex, x+leadingSpaceWidth(text), render.StringWidth(trimmed)))
			bookmarkPlaceholderX = x + width
		case isDefaultBookmarkStyle(segment.Style):
			for _, word := range wordSpans(text) {
				objects = append(objects, newRevisionTextObject(row, textObjectBookmark, word.value, bookmarkIndex, lineIndex, x+render.StringWidth(text[:word.start]), render.StringWidth(word.value)))
				bookmarkIndex++
			}
		case fieldOrdinal == 0:
			// In compact templates without author/date metadata, the subject can
			// appear inline before the commit id. Treat that as description.
			desc := newRevisionTextObject(row, textObjectDescription, trimmed, 0, lineIndex, x+leadingSpaceWidth(text), render.StringWidth(trimmed))
			inlineDescription = &desc
		case seenCommitID && inlineDescription == nil:
			// Single-line templates can put the description after the commit id.
			desc := newRevisionTextObject(row, textObjectDescription, trimmed, 0, lineIndex, x+leadingSpaceWidth(text), render.StringWidth(trimmed))
			inlineDescription = &desc
		}
		fieldOrdinal++
		x += width
	}

	if bookmarkIndex == 0 {
		objects = append(objects, newRevisionTextObject(row, textObjectBookmark, "", 0, lineIndex, bookmarkPlaceholderX, 1))
	}
	if inlineDescription != nil {
		objects = append(objects, *inlineDescription)
	}

	return objects
}

func descriptionTextObject(row parser.Row, revisionLineIndex int) (revisionTextObject, bool) {
	for lineIndex, line := range row.Lines {
		if lineIndex == revisionLineIndex || line.Flags&parser.Highlightable != parser.Highlightable || line.Flags&parser.Revision == parser.Revision {
			continue
		}
		text, firstX := lineText(line)
		value := strings.TrimSpace(text)
		if value == "" {
			continue
		}
		return newRevisionTextObject(row, textObjectDescription, value, 0, lineIndex, firstX, render.StringWidth(value)), true
	}
	return revisionTextObject{}, false
}

func newRevisionTextObject(row parser.Row, kind string, value string, index int, line int, x int, width int) revisionTextObject {
	return revisionTextObject{
		FocusedObject: common.FocusedObject{
			Kind:     kind,
			Value:    value,
			ChangeId: row.Commit.GetChangeId(),
			CommitId: row.Commit.CommitId,
			Index:    index,
		},
		line:  line,
		x:     x,
		width: max(width, 1),
	}
}

func gutterWidth(line *parser.GraphRowLine) int {
	width := 0
	for _, segment := range line.Gutter.Segments {
		width += render.StringWidth(segment.Text)
	}
	return width
}

func lineText(line *parser.GraphRowLine) (string, int) {
	var b strings.Builder
	x := gutterWidth(line)
	firstX := x
	seenText := false
	for _, segment := range line.Segments {
		if !seenText {
			firstX = x + leadingSpaceWidth(segment.Text)
			seenText = strings.TrimSpace(segment.Text) != ""
		}
		b.WriteString(segment.Text)
		x += render.StringWidth(segment.Text)
	}
	return b.String(), firstX
}

func isCommitIdentifier(row parser.Row, text string) bool {
	return text == row.Commit.ChangeId ||
		text == row.Commit.GetChangeId() ||
		text == row.Commit.CommitId
}

func isDimmed(style lipgloss.Style) bool {
	return styleForeground(style) == "8"
}

func isDefaultAuthorStyle(style lipgloss.Style) bool {
	return styleForeground(style) == "3"
}

func isDefaultDateStyle(style lipgloss.Style) bool {
	color := styleForeground(style)
	return color == "6" || color == "14"
}

func isDefaultBookmarkStyle(style lipgloss.Style) bool {
	color := styleForeground(style)
	return color == "5" || color == "13"
}

func styleForeground(style lipgloss.Style) string {
	if fg := style.GetForeground(); fg != nil {
		return fmt.Sprint(fg)
	}
	return ""
}

func leadingSpaceWidth(text string) int {
	for i, r := range text {
		if !unicode.IsSpace(r) {
			return render.StringWidth(text[:i])
		}
	}
	return 0
}

type wordSpan struct {
	start int
	value string
}

func wordSpans(text string) []wordSpan {
	var spans []wordSpan
	inWord := false
	start := 0
	for i, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				spans = append(spans, wordSpan{start: start, value: text[start:i]})
				inWord = false
			}
			continue
		}
		if !inWord {
			start = i
			inWord = true
		}
	}
	if inWord {
		spans = append(spans, wordSpan{start: start, value: text[start:]})
	}
	return spans
}

func textObjectCursor() *tea.Cursor {
	cursor := tea.NewCursor(0, 0)
	cursor.Shape = tea.CursorBlock
	cursor.Blink = true
	return cursor
}
