package revisions

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common/models"
	"github.com/idursun/jjui/internal/ui/operations"
)

func (r *RevisionList) GetItemHeight(index int) int {
	return len(r.Items[index].Lines)
}

func (r *RevisionList) RenderItem(w io.Writer, index int) {
	row := r.Items[index]
	inLane := r.Tracer.IsInSameLane(index)
	isHighlighted := index == r.Cursor
	if before := r.RenderBefore(row.Commit); before != "" {
		extended := models.GraphGutter{}
		if row.Previous != nil {
			extended = row.Previous.Extend()
		}
		r.writeSection(w, index, extended, extended, false, before)
	}

	descriptionOverlay := r.op.Render(row.Commit, operations.RenderOverDescription)
	requiresDescriptionRendering := descriptionOverlay != "" && isHighlighted
	descriptionRendered := false

	// Each line has a flag:
	// Revision: the line contains a change id and commit id (which is assumed to be the first line of the row)
	// Highlightable: the line can be highlighted (e.g. revision line and description line)
	// Elided: this is usually the last line of the row, it is not highlightable
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]
		if segmentedLine.Flags&models.Elided == models.Elided {
			break
		}
		lw := strings.Builder{}
		if segmentedLine.Flags&models.Revision != models.Revision && isHighlighted {
			if requiresDescriptionRendering {
				r.writeSection(w, index, segmentedLine.Gutter, row.Extend(), true, descriptionOverlay)
				descriptionRendered = true
				// skip all remaining highlightable lines
				for lineIndex < len(row.Lines) {
					if row.Lines[lineIndex].Flags&models.Highlightable == models.Highlightable {
						lineIndex++
						continue
					} else {
						break
					}
				}
				continue
			}
		}

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

		if segmentedLine.Flags&models.Revision == models.Revision {
			if decoration := r.RenderBeforeChangeId(index, row); decoration != "" {
				fmt.Fprint(&lw, decoration)
			}
		}

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

			start, end := segment.FindSubstringRange(r.quickSearch)
			if start != -1 {
				mid := lipgloss.NewRange(start, end, style.Reverse(true))
				fmt.Fprint(&lw, lipgloss.StyleRanges(style.Render(segment.Text), mid))
			} else if aceIdx := r.aceJumpIndex(segment, row.Row); aceIdx > -1 {
				mid := lipgloss.NewRange(aceIdx, aceIdx+1, style.Reverse(true))
				fmt.Fprint(&lw, lipgloss.StyleRanges(style.Render(segment.Text), mid))
			} else {
				fmt.Fprint(&lw, style.Render(segment.Text))
			}
		}
		if segmentedLine.Flags&models.Revision == models.Revision && row.IsAffected {
			style := r.dimmedStyle
			if isHighlighted {
				style = r.dimmedStyle.Background(r.selectedStyle.GetBackground())
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		if isHighlighted && segmentedLine.Flags&models.Highlightable == models.Highlightable {
			fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r.renderer, lipgloss.PlaceHorizontal(r.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(r.textStyle.GetBackground())))
		}
		fmt.Fprint(r.renderer, "\n")
	}

	if requiresDescriptionRendering && !descriptionRendered {
		r.writeSection(r.renderer, index, row.Extend(), row.Extend(), true, descriptionOverlay)
	}

	if row.Commit.IsRoot() {
		return
	}

	if afterSection := r.RenderAfter(row.Commit); afterSection != "" {
		extended := row.Extend()
		r.writeSection(r.renderer, index, extended, extended, false, afterSection)
	}

	for lineIndex, segmentedLine := range row.RowLinesIter(models.Excluding(models.Highlightable)) {
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

func (r *RevisionList) writeSection(w io.Writer, index int, current models.GraphGutter, extended models.GraphGutter, highlight bool, section string) {
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

func (r *RevisionList) RenderBefore(commit *jj.Commit) string {
	return r.op.Render(commit, operations.RenderPositionBefore)
}

func (r *RevisionList) RenderAfter(commit *jj.Commit) string {
	return r.op.Render(commit, operations.RenderPositionAfter)
}

func (r *RevisionList) RenderBeforeChangeId(index int, item *models.RevisionItem) string {
	commit := item.Commit
	isSelected := item.IsChecked()
	isHighlighted := r.Cursor == index
	opMarker := r.op.Render(commit, operations.RenderBeforeChangeId)
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

func (r *RevisionList) RenderBeforeCommitId(commit *jj.Commit) string {
	return r.op.Render(commit, operations.RenderBeforeCommitId)
}

func (r *RevisionList) aceJumpIndex(segment *screen.Segment, row models.Row) int {
	aceJumpPrefix := r.aceJump.Prefix()
	if aceJumpPrefix == nil || row.Commit == nil {
		return -1
	}
	if !(segment.Text == row.Commit.ChangeId || segment.Text == row.Commit.CommitId) {
		return -1
	}
	lowerText, lowerPrefix := strings.ToLower(segment.Text), strings.ToLower(*aceJumpPrefix)
	if !strings.HasPrefix(lowerText, lowerPrefix) {
		return -1
	}
	idx := len(lowerPrefix)
	if idx == len(lowerText) {
		idx-- // dont move past last character
	}
	return idx
}
