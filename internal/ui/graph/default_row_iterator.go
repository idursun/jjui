package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/models"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowIterator struct {
	SearchText    string
	AceJumpPrefix *string
	Op            operations.Operation
	Width         int
	Rows          []models.Row
	current       int
	Cursor        int
	isHighlighted bool
	isSelected    bool
	selections    map[string]bool
	dimmedStyle   lipgloss.Style
	checkStyle    lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	Tracer        parser.LaneTracer
}

type Option func(*DefaultRowIterator)

func NewDefaultRowIterator(rows []models.Row, options ...Option) *DefaultRowIterator {
	iterator := &DefaultRowIterator{
		Op:         &operations.Default{},
		Rows:       rows,
		selections: make(map[string]bool),
		Tracer:     parser.NewNoopTracer(),
		current:    -1,
	}

	for _, opt := range options {
		opt(iterator)
	}

	return iterator
}

func WithWidth(width int) Option {
	return func(s *DefaultRowIterator) {
		s.Width = width
	}
}

func WithStylePrefix(prefix string) Option {
	if prefix != "" {
		prefix += " "
	}
	return func(s *DefaultRowIterator) {
		s.textStyle = common.DefaultPalette.Get(prefix + "text").Inline(true)
		s.selectedStyle = common.DefaultPalette.Get(prefix + "selected").Inline(true)
		s.dimmedStyle = common.DefaultPalette.Get(prefix + "dimmed")
		s.checkStyle = common.DefaultPalette.Get(prefix + "success").Inline(true)
	}
}

func WithSelections(selections map[string]bool) Option {
	return func(s *DefaultRowIterator) {
		s.selections = selections
	}
}

func (s *DefaultRowIterator) IsHighlighted() bool {
	return s.current == s.Cursor
}

func (s *DefaultRowIterator) Next() bool {
	s.current++
	if s.current >= len(s.Rows) {
		return false
	}
	s.isHighlighted = s.current == s.Cursor
	s.isSelected = false
	if v, ok := s.selections[s.Rows[s.current].Commit.GetChangeId()]; ok {
		s.isSelected = v
	}
	return true
}

func (s *DefaultRowIterator) RowHeight() int {
	return len(s.Rows[s.current].Lines)
}

func (s *DefaultRowIterator) aceJumpIndex(segment *screen.Segment, row models.Row) int {
	if s.AceJumpPrefix == nil || row.Commit == nil {
		return -1
	}
	if !(segment.Text == row.Commit.ChangeId || segment.Text == row.Commit.CommitId) {
		return -1
	}
	lowerText, lowerPrefix := strings.ToLower(segment.Text), strings.ToLower(*s.AceJumpPrefix)
	if !strings.HasPrefix(lowerText, lowerPrefix) {
		return -1
	}
	idx := len(lowerPrefix)
	if idx == len(lowerText) {
		idx-- // dont move past last character
	}
	return idx
}

func (s *DefaultRowIterator) Render(r io.Writer) {
	row := s.Rows[s.current]
	inLane := s.Tracer.IsInSameLane(s.current)

	// will render by extending the previous connections
	if before := s.RenderBefore(row.Commit); before != "" {
		extended := models.GraphGutter{}
		if row.Previous != nil {
			extended = row.Previous.Extend()
		}
		s.writeSection(r, extended, extended, false, before)
	}

	descriptionOverlay := s.Op.Render(row.Commit, operations.RenderOverDescription)
	requiresDescriptionRendering := descriptionOverlay != "" && s.isHighlighted
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
		if segmentedLine.Flags&models.Revision != models.Revision && s.isHighlighted {
			if requiresDescriptionRendering {
				s.writeSection(r, segmentedLine.Gutter, row.Extend(), true, descriptionOverlay)
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
			gutterInLane := s.Tracer.IsGutterInLane(s.current, lineIndex, i)
			text := s.Tracer.UpdateGutterText(s.current, lineIndex, i, segment.Text)
			style := segment.Style
			if gutterInLane {
				style = style.Inherit(s.textStyle)
			} else {
				style = style.Inherit(s.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}

		if segmentedLine.Flags&models.Revision == models.Revision {
			if decoration := s.RenderBeforeChangeId(row.Commit); decoration != "" {
				fmt.Fprint(&lw, decoration)
			}
		}

		for _, segment := range segmentedLine.Segments {
			if s.isHighlighted && segment.Text == row.Commit.CommitId {
				if decoration := s.RenderBeforeCommitId(row.Commit); decoration != "" {
					fmt.Fprint(&lw, decoration)
				}
			}

			style := segment.Style
			if s.isHighlighted {
				style = style.Inherit(s.selectedStyle)
			} else if inLane {
				style = style.Inherit(s.textStyle)
			} else {
				style = style.Inherit(s.dimmedStyle).Faint(true)
			}

			start, end := segment.FindSubstringRange(s.SearchText)
			if start != -1 {
				mid := lipgloss.NewRange(start, end, style.Reverse(true))
				fmt.Fprint(&lw, lipgloss.StyleRanges(style.Render(segment.Text), mid))
			} else if aceIdx := s.aceJumpIndex(segment, row); aceIdx > -1 {
				mid := lipgloss.NewRange(aceIdx, aceIdx+1, style.Reverse(true))
				fmt.Fprint(&lw, lipgloss.StyleRanges(style.Render(segment.Text), mid))
			} else {
				fmt.Fprint(&lw, style.Render(segment.Text))
			}
		}
		if segmentedLine.Flags&models.Revision == models.Revision && row.IsAffected {
			style := s.dimmedStyle
			if s.isHighlighted {
				style = s.dimmedStyle.Background(s.selectedStyle.GetBackground())
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		if s.isHighlighted && segmentedLine.Flags&models.Highlightable == models.Highlightable {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.textStyle.GetBackground())))
		}
		fmt.Fprint(r, "\n")
	}

	if requiresDescriptionRendering && !descriptionRendered {
		s.writeSection(r, row.Extend(), row.Extend(), true, descriptionOverlay)
	}

	if row.Commit.IsRoot() {
		return
	}

	if afterSection := s.RenderAfter(row.Commit); afterSection != "" {
		extended := row.Extend()
		s.writeSection(r, extended, extended, false, afterSection)
	}

	for lineIndex, segmentedLine := range row.RowLinesIter(models.Excluding(models.Highlightable)) {
		var lw strings.Builder
		for i, segment := range segmentedLine.Gutter.Segments {
			gutterInLane := s.Tracer.IsGutterInLane(s.current, lineIndex, i)
			text := s.Tracer.UpdateGutterText(s.current, lineIndex, i, segment.Text)
			style := segment.Style
			if gutterInLane {
				style = style.Inherit(s.textStyle)
			} else {
				style = style.Inherit(s.dimmedStyle).Faint(true)
			}
			fmt.Fprint(&lw, style.Render(text))
		}
		for _, segment := range segmentedLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(s.textStyle).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.textStyle.GetBackground())))
		fmt.Fprint(r, "\n")
	}
}

// current gutter to be used in the first line (needed for overlaying the description)
// extended used to repeat the gutter for each line
func (s *DefaultRowIterator) writeSection(r io.Writer, current models.GraphGutter, extended models.GraphGutter, highlight bool, section string) {
	lines := strings.Split(section, "\n")
	for _, sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range current.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(s.textStyle).Render(segment.Text))
		}

		fmt.Fprint(&lw, sectionLine)
		line := lw.String()
		if s.isHighlighted && highlight {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.textStyle.GetBackground())))
		}
		fmt.Fprintln(r)
		current = extended
	}
}

func (s *DefaultRowIterator) Len() int {
	return len(s.Rows)
}

func (s *DefaultRowIterator) RenderBefore(commit *jj.Commit) string {
	return s.Op.Render(commit, operations.RenderPositionBefore)
}

func (s *DefaultRowIterator) RenderAfter(commit *jj.Commit) string {
	return s.Op.Render(commit, operations.RenderPositionAfter)
}

func (s *DefaultRowIterator) RenderBeforeChangeId(commit *jj.Commit) string {
	opMarker := s.Op.Render(commit, operations.RenderBeforeChangeId)
	selectedMarker := ""
	if s.isSelected {
		if s.isHighlighted {
			selectedMarker = s.checkStyle.Background(s.selectedStyle.GetBackground()).Render("✓")
		} else {
			selectedMarker = s.checkStyle.Background(s.textStyle.GetBackground()).Render("✓")
		}
	}
	if opMarker == "" && selectedMarker == "" {
		return ""
	}
	var sections []string

	space := s.textStyle.Render(" ")
	if s.isHighlighted {
		space = s.selectedStyle.Render(" ")
	}

	if opMarker != "" {
		sections = append(sections, opMarker, space)
	}
	if selectedMarker != "" {
		sections = append(sections, selectedMarker, space)
	}
	return lipgloss.JoinHorizontal(0, sections...)
}

func (s *DefaultRowIterator) RenderBeforeCommitId(commit *jj.Commit) string {
	return s.Op.Render(commit, operations.RenderBeforeCommitId)
}
