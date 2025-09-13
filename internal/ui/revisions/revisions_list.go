package revisions

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	models2 "github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ list.IItemRenderer = (*RevisionList)(nil)

type RevisionList struct {
	*list.CheckableList[*models2.RevisionItem]
	renderer      *list.ListRenderer[*models2.RevisionItem]
	Tracer        parser.LaneTracer
	getOpFn       func() operations.Operation
	dimmedStyle   lipgloss.Style
	checkStyle    lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func (r *RevisionList) GetItemHeight(index int) int {
	return len(r.Items[index].Lines)
}

func (r *RevisionList) RenderItem(w io.Writer, index int) {
	row := r.Items[index]
	inLane := r.Tracer.IsInSameLane(index)
	isHighlighted := index == r.Cursor
	// render: before
	if before := r.RenderBefore(row.Commit); before != "" {
		extended := models2.GraphGutter{}
		if row.Previous != nil {
			extended = row.Previous.Extend()
		}
		r.writeSection(w, index, extended, extended, false, before)
	}

	// render: description overlay
	descriptionOverlay := r.getOpFn().Render(row.Commit, operations.RenderOverDescription)
	requiresDescriptionRendering := descriptionOverlay != "" && isHighlighted
	descriptionRendered := false

	// Each line has a flag:
	// Revision: the line contains a change id and commit id (which is assumed to be the first line of the row)
	// Highlightable: the line can be highlighted (e.g. revision line and description line)
	// Elided: this is usually the last line of the row, it is not highlightable
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]
		if segmentedLine.Flags&models2.Elided == models2.Elided {
			break
		}
		lw := strings.Builder{}
		if segmentedLine.Flags&models2.Revision != models2.Revision && isHighlighted {
			if requiresDescriptionRendering {
				r.writeSection(w, index, segmentedLine.Gutter, row.Extend(), true, descriptionOverlay)
				descriptionRendered = true
				// skip all remaining highlightable lines
				for lineIndex < len(row.Lines) {
					if row.Lines[lineIndex].Flags&models2.Highlightable == models2.Highlightable {
						lineIndex++
						continue
					} else {
						break
					}
				}
				continue
			}
		}

		// render: gutter
		for i, segment := range segmentedLine.Gutter.Segments {
			gutterInLane := r.Tracer.IsGutterInLane(index, lineIndex, i)
			text := r.Tracer.UpdateGutterText(index, lineIndex, i, segment.Text)
			style := segment.Style
			if gutterInLane {
				style = style.Inherit(r.textStyle)
			} else {
				style = style.Inherit(r.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}

		// render: before change id
		if segmentedLine.Flags&models2.Revision == models2.Revision {
			if decoration := r.RenderBeforeChangeId(index, row); decoration != "" {
				fmt.Fprint(&lw, decoration)
			}
		}

		// render: after change id
		for _, segment := range segmentedLine.Segments {
			if isHighlighted && segment.Text == row.Commit.CommitId {
				if decoration := r.RenderBeforeCommitId(row.Commit); decoration != "" {
					fmt.Fprint(&lw, decoration)
				}
			}

			style := segment.Style
			if isHighlighted {
				style = style.Inherit(r.selectedStyle)
			} else if inLane {
				style = style.Inherit(r.textStyle)
			} else {
				style = style.Inherit(r.dimmedStyle).Faint(true)
			}

			op := r.getOpFn()
			if sr, ok := op.(operations.SegmentRenderer); ok {
				rendered := sr.RenderSegment(style, segment, row)
				if rendered != "" {
					fmt.Fprint(&lw, style.Render(rendered))
					continue
				}
			}

			// if the SegmentRenderer did not render anything, fall back to default rendering
			fmt.Fprint(&lw, style.Render(segment.Text))
		}

		// render: affected by last operation
		if segmentedLine.Flags&models2.Revision == models2.Revision && row.IsAffected {
			style := r.dimmedStyle
			if isHighlighted {
				style = r.dimmedStyle.Background(r.selectedStyle.GetBackground())
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		if isHighlighted && segmentedLine.Flags&models2.Highlightable == models2.Highlightable {
			fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.textStyle.GetBackground())))
		}
		fmt.Fprint(r.renderer, "\n")
	}

	// render: description overlay if not yet rendered
	if requiresDescriptionRendering && !descriptionRendered {
		r.writeSection(r.renderer, index, row.Extend(), row.Extend(), true, descriptionOverlay)
	}

	if row.Commit.IsRoot() {
		return
	}

	// render: after
	if afterSection := r.RenderAfter(row.Commit); afterSection != "" {
		extended := row.Extend()
		r.writeSection(r.renderer, index, extended, extended, false, afterSection)
	}

	// render: remaining lines (non-highlightable)
	for lineIndex, segmentedLine := range row.RowLinesIter(models2.Excluding(models2.Highlightable)) {
		var lw strings.Builder
		for i, segment := range segmentedLine.Gutter.Segments {
			gutterInLane := r.Tracer.IsGutterInLane(index, lineIndex, i)
			text := r.Tracer.UpdateGutterText(index, lineIndex, i, segment.Text)
			style := segment.Style
			if gutterInLane {
				style = style.Inherit(r.textStyle)
			} else {
				style = style.Inherit(r.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}
		for _, segment := range segmentedLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(r.textStyle).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.textStyle.GetBackground())))
		fmt.Fprint(r.renderer, "\n")
	}

}

func (r *RevisionList) writeSection(w io.Writer, index int, current models2.GraphGutter, extended models2.GraphGutter, highlight bool, section string) {
	isHighlighted := index == r.Cursor
	lines := strings.Split(section, "\n")
	for _, sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range current.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(r.textStyle).Render(segment.Text))
		}

		fmt.Fprint(&lw, sectionLine)
		line := lw.String()
		if isHighlighted && highlight {
			fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.textStyle.GetBackground())))
		}
		fmt.Fprintln(w)
		current = extended
	}
}

func (r *RevisionList) RenderBefore(commit *models2.Commit) string {
	return r.getOpFn().Render(commit, operations.RenderPositionBefore)
}

func (r *RevisionList) RenderAfter(commit *models2.Commit) string {
	return r.getOpFn().Render(commit, operations.RenderPositionAfter)
}

func (r *RevisionList) RenderBeforeChangeId(index int, item *models2.RevisionItem) string {
	commit := item.Commit
	isSelected := item.IsChecked()
	isHighlighted := r.Cursor == index
	opMarker := r.getOpFn().Render(commit, operations.RenderBeforeChangeId)
	selectedMarker := ""
	if isSelected {
		if isHighlighted {
			selectedMarker = r.checkStyle.Background(r.selectedStyle.GetBackground()).Render("✓")
		} else {
			selectedMarker = r.checkStyle.Background(r.textStyle.GetBackground()).Render("✓")
		}
	}
	if opMarker == "" && selectedMarker == "" {
		return ""
	}
	var sections []string

	space := r.textStyle.Render(" ")
	if isHighlighted {
		space = r.selectedStyle.Render(" ")
	}

	if opMarker != "" {
		sections = append(sections, opMarker, space)
	}
	if selectedMarker != "" {
		sections = append(sections, selectedMarker, space)
	}
	return lipgloss.JoinHorizontal(0, sections...)
}

func (r *RevisionList) RenderBeforeCommitId(commit *models2.Commit) string {
	return r.getOpFn().Render(commit, operations.RenderBeforeCommitId)
}
