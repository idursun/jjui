package revisions

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row              parser.Row
	isHighlighted    bool
	selectedStyle    lipgloss.Style
	textStyle        lipgloss.Style
	dimmedStyle      lipgloss.Style
	matchedStyle     lipgloss.Style
	isGutterInLane   func(lineIndex, segmentIndex int) bool
	updateGutterText func(lineIndex, segmentIndex int, text string) string
	inLane           bool
	op               operations.Operation
	SearchText       string
	AceJumpPrefix    *string
	isChecked        bool
}

func (ir itemRenderer) writeSection(w io.Writer, current parser.GraphGutter, extended parser.GraphGutter, highlight bool, section string, width int) {
	isHighlighted := ir.isHighlighted
	section = strings.TrimSuffix(section, "\n")
	lines := strings.SplitSeq(section, "\n")
	for sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range current.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(ir.textStyle).Render(segment.Text))
		}

		fmt.Fprint(&lw, sectionLine)
		line := lw.String()
		if isHighlighted && highlight {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.textStyle.GetBackground())))
		}
		fmt.Fprintln(w)
		current = extended
	}
}

func (ir itemRenderer) Render(w io.Writer, width int) {
	ir.renderBeforeSection(w, width)

	descriptionOverlay := ""
	if ir.isHighlighted {
		descriptionOverlay = ir.op.Render(ir.row.Commit, operations.RenderOverDescription)
	}

	descriptionRendered := ir.renderMainLines(w, width, descriptionOverlay)

	if descriptionOverlay != "" && !descriptionRendered {
		ir.writeSection(w, ir.row.Extend(), ir.row.Extend(), true, descriptionOverlay, width)
	}

	ir.renderAfterSection(w, width)
	ir.renderNonHighlightableLines(w, width)
}

// renderBeforeSection renders content before the main revision lines by extending
// the previous row's graph connections.
// This is used for operation-specific content that appear above the revision.
func (ir itemRenderer) renderBeforeSection(w io.Writer, width int) {
	before := ir.op.Render(ir.row.Commit, operations.RenderPositionBefore)
	if before == "" {
		return
	}

	extended := parser.GraphGutter{}
	if ir.row.Previous != nil {
		extended = ir.row.Previous.Extend()
	}
	ir.writeSection(w, extended, extended, false, before, width)
}

// renderMainLines renders the main highlightable lines of a revision row, including
// the revision line (with ChangeID/CommitID) and description lines.
// If a description overlay is provided for a highlighted row, it replaces the
// normal description lines.
// Returns true if the description overlay was rendered, false otherwise.
func (ir itemRenderer) renderMainLines(w io.Writer, width int, descriptionOverlay string) bool {
	descriptionRendered := false
	needDescription := descriptionOverlay != ""

	// Each line has a flag:
	// Revision: the line contains a change id and commit id (which is assumed to be the first line of the row)
	// Highlightable: the line can be highlighted (e.g. revision line and description line)
	// Elided: this is usually the last line of the row, it is not highlightable
	for lineIndex := 0; lineIndex < len(ir.row.Lines); lineIndex++ {
		segmentedLine := ir.row.Lines[lineIndex]

		if segmentedLine.Flags&parser.Elided == parser.Elided {
			break
		}
		if !ir.isHighlighted || !needDescription || descriptionRendered {
			ir.renderLine(w, width, lineIndex, *segmentedLine)
			continue
		}
		if segmentedLine.Flags&parser.Highlightable != parser.Highlightable {
			ir.renderLine(w, width, lineIndex, *segmentedLine)
			continue
		}
		if segmentedLine.Flags&parser.Revision == parser.Revision {
			ir.renderLine(w, width, lineIndex, *segmentedLine)
			continue
		}

		// All conditions met, render description
		ir.writeSection(w, segmentedLine.Gutter, ir.row.Extend(), true, descriptionOverlay, width)
		descriptionRendered = true
		lineIndex = ir.skipHighlightableLines(lineIndex)
	}

	return descriptionRendered
}

// skipHighlightableLines is used when a description overlay is rendered to
// skip the original description lines.
func (ir itemRenderer) skipHighlightableLines(startIndex int) int {
	for startIndex < len(ir.row.Lines) {
		if ir.row.Lines[startIndex].Flags&parser.Highlightable == parser.Highlightable {
			startIndex++
		} else {
			break
		}
	}
	return startIndex
}

// renderLine renders a single line of a revision row, including the graph gutter,
// different segments (ChangeID, CommitID, description, etc)
// and an optional marker indicating if the revision was affected by the last
// operation.
func (ir itemRenderer) renderLine(w io.Writer, width int, lineIndex int, segmentedLine parser.GraphRowLine) {
	lw := strings.Builder{}
	ir.renderGutter(&lw, lineIndex, segmentedLine)
	ir.renderSegments(&lw, segmentedLine)
	ir.renderAffectedMarker(&lw, segmentedLine)

	line := lw.String()
	if ir.isHighlighted && segmentedLine.Flags&parser.Highlightable == parser.Highlightable {
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.selectedStyle.GetBackground())))
	} else {
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.textStyle.GetBackground())))
	}
	fmt.Fprint(w, "\n")
}

// renderGutter renders the graph gutter portion
// For revision lines, it also renders the checkbox and any operation-specific
// content before the ChangeID.
func (ir itemRenderer) renderGutter(lw *strings.Builder, lineIndex int, segmentedLine parser.GraphRowLine) {
	for i, segment := range segmentedLine.Gutter.Segments {
		gutterInLane := ir.isGutterInLane(lineIndex, i)
		text := ir.updateGutterText(lineIndex, i, segment.Text)
		style := segment.Style
		if gutterInLane {
			style = style.Inherit(ir.textStyle)
		} else {
			style = style.Inherit(ir.dimmedStyle).Faint(true)
		}
		fmt.Fprint(lw, style.Render(text))
	}

	beforeChangeID := ir.op.Render(ir.row.Commit, operations.RenderBeforeChangeId)
	if segmentedLine.Flags&parser.Revision == parser.Revision {
		if ir.isChecked {
			fmt.Fprint(lw, ir.selectedStyle.Render("✓ "))
		}
		if beforeChangeID != "" {
			fmt.Fprint(lw, beforeChangeID)
		}
	}
}

// renderSegments renders the content segments (ChangeID, CommitID, description)
// It supports operation-specific segment rendering, search text highlighting
func (ir itemRenderer) renderSegments(lw *strings.Builder, segmentedLine parser.GraphRowLine) {
	beforeCommitID := ir.op.Render(ir.row.Commit, operations.RenderBeforeCommitId)

	for _, segment := range segmentedLine.Segments {
		if beforeCommitID != "" && segment.Text == ir.row.Commit.CommitId {
			fmt.Fprint(lw, beforeCommitID)
		}

		style := ir.getSegmentStyle(*segment)

		if sr, ok := ir.op.(operations.SegmentRenderer); ok {
			rendered := sr.RenderSegment(style, segment, ir.row)
			if rendered != "" {
				fmt.Fprint(lw, style.Render(rendered))
				continue
			}
		}

		text := segment.Text
		if ir.SearchText != "" && strings.TrimSpace(text) != "" &&
			strings.Contains(strings.ToLower(text), ir.SearchText) {
			ir.renderQuickSearchHighlight(lw, text, style)
		} else {
			fmt.Fprint(lw, style.Render(text))
		}
	}
}

func (ir itemRenderer) getSegmentStyle(segment screen.Segment) lipgloss.Style {
	style := segment.Style
	if ir.isHighlighted {
		style = style.Inherit(ir.selectedStyle)
	} else if ir.inLane {
		style = style.Inherit(ir.textStyle)
	} else {
		style = style.Inherit(ir.dimmedStyle).Faint(true)
	}
	return style
}

func (ir itemRenderer) renderAffectedMarker(lw *strings.Builder, segmentedLine parser.GraphRowLine) {
	if segmentedLine.Flags&parser.Revision == parser.Revision && ir.row.IsAffected {
		style := ir.dimmedStyle
		if ir.isHighlighted {
			style = ir.dimmedStyle.Background(ir.selectedStyle.GetBackground())
		}
		fmt.Fprint(lw, style.Render(" (affected by last operation)"))
	}
}

// renderAfterSection renders content after the main revision lines by extending
// the row's graph connections.
// This is used for operation-specific content that should appear below the
// revision. Skipped for root commits.
func (ir itemRenderer) renderAfterSection(w io.Writer, width int) {
	if ir.row.Commit.IsRoot() {
		return
	}

	after := ir.op.Render(ir.row.Commit, operations.RenderPositionAfter)
	if after != "" {
		extended := ir.row.Extend()
		ir.writeSection(w, extended, extended, false, after, width)
	}
}

// renderNonHighlightableLines renders non-highlightable lines (additional
// metadata, elided content markers, etc.) that appear after the main revision
// content.
// These lines are always rendered with normal style and cannot be selected.
func (ir itemRenderer) renderNonHighlightableLines(w io.Writer, width int) {
	for lineIndex, segmentedLine := range ir.row.RowLinesIter(parser.Excluding(parser.Highlightable)) {
		var lw strings.Builder
		for i, segment := range segmentedLine.Gutter.Segments {
			gutterInLane := ir.isGutterInLane(lineIndex, i)
			text := ir.updateGutterText(lineIndex, i, segment.Text)

			style := segment.Style
			if gutterInLane {
				style = style.Inherit(ir.textStyle)
			} else {
				style = style.Inherit(ir.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}
		for _, segment := range segmentedLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(ir.textStyle).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(ir.textStyle.GetBackground())))
		fmt.Fprint(w, "\n")
	}
}

func (ir itemRenderer) Height() int {
	h := 0

	// Before section
	before := ir.op.Render(ir.row.Commit, operations.RenderPositionBefore)
	if before != "" {
		h += strings.Count(before, "\n") + 1
	}

	// Main content
	descriptionOverlay := ""
	if ir.isHighlighted {
		descriptionOverlay = ir.op.Render(ir.row.Commit, operations.RenderOverDescription)
	}
	requiresDescriptionRendering := descriptionOverlay != ""

	if requiresDescriptionRendering {
		h += strings.Count(descriptionOverlay, "\n") + 1
		for _, line := range ir.row.Lines {
			// Revision line is always kept
			if line.Flags&parser.Revision == parser.Revision {
				h++
				continue
			}
			// Highlightable lines are replaced by overlay
			if line.Flags&parser.Highlightable == parser.Highlightable {
				continue
			}
			// Elided lines are hidden when overlay is present
			if line.Flags&parser.Elided == parser.Elided {
				continue
			}
			// Keep other lines
			h++
		}
	} else {
		h += len(ir.row.Lines)
	}

	// After section
	if !ir.row.Commit.IsRoot() {
		after := ir.op.Render(ir.row.Commit, operations.RenderPositionAfter)
		if after != "" {
			h += strings.Count(after, "\n") + 1
		}
	}

	return h
}

func (ir itemRenderer) renderQuickSearchHighlight(w io.Writer, textToRender string, style lipgloss.Style) {
	searchTerm := ir.SearchText
	lowercaseText := strings.ToLower(textToRender)

	var lastIdx int

	// Find all occurrences of search term (case-insensitive)
	for {
		searchTermIdx := strings.Index(lowercaseText[lastIdx:], searchTerm)

		if searchTermIdx == -1 {
			if lastIdx < len(textToRender) {
				fmt.Fprint(w, style.Render(textToRender[lastIdx:]))
			}
			break
		}

		betweenHighlights := textToRender[lastIdx : lastIdx+searchTermIdx]
		fmt.Fprint(w, style.Render(betweenHighlights))

		highlights := textToRender[lastIdx+searchTermIdx : lastIdx+searchTermIdx+len(searchTerm)]
		fmt.Fprint(w, ir.matchedStyle.Render(highlights))
		lastIdx += searchTermIdx + len(searchTerm)
	}
}
