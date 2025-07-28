package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowIterator struct {
	SearchText    string
	AceJumpPrefix *string
	Selections    map[string]bool
	Op            operations.Operation
	Width         int
	Rows          []parser.Row
	isHighlighted bool
	isSelected    bool
	current       int
	Cursor        int
	dimmedStyle   lipgloss.Style
	checkStyle    lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

type Option func(*DefaultRowIterator)

func NewDefaultRowIterator(rows []parser.Row, options ...Option) *DefaultRowIterator {
	iterator := &DefaultRowIterator{
		Op:         &operations.Default{},
		Rows:       rows,
		Selections: make(map[string]bool),
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
		prefix = " "
	}
	return func(s *DefaultRowIterator) {
		s.textStyle = common.DefaultPalette.Get(prefix + "text").Inline(true)
		s.selectedStyle = common.DefaultPalette.Get(prefix + "selected").Inline(true)
		s.dimmedStyle = common.DefaultPalette.Get(prefix + "dimmed")
		s.checkStyle = common.DefaultPalette.Get(prefix + "success").Inline(true)
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
	if v, ok := s.Selections[s.Rows[s.current].Commit.GetChangeId()]; ok {
		s.isSelected = v
	}
	return true
}

func (s *DefaultRowIterator) RowHeight() int {
	return len(s.Rows[s.current].Lines)
}

func (s *DefaultRowIterator) aceJumpIndex(segment *screen.Segment, row parser.Row) int {
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
	// will render by extending the previous connections
	if before := s.RenderBefore(row.Commit); before != "" {
		extended := parser.GraphGutter{}
		if row.Previous != nil {
			extended = row.Previous.Extend()
		}
		s.writeSection(r, extended, extended, false, before)
	}
	for segmentedLine := range row.RowLinesIter(parser.Including(parser.Highlightable)) {
		lw := strings.Builder{}
		if segmentedLine.Flags&parser.Revision != parser.Revision && s.isHighlighted {
			if decoration := s.Op.Render(row.Commit, operations.RenderOverDescription); decoration != "" {
				s.writeSection(r, segmentedLine.Gutter, row.Extend(), true, decoration)
				continue
			}
		}

		for _, segment := range segmentedLine.Gutter.Segments {
			if s.isHighlighted {
				fmt.Fprint(&lw, segment.Style.Inherit(s.selectedStyle).Render(segment.Text))
			} else {
				fmt.Fprint(&lw, segment.Style.Inherit(s.textStyle).Render(segment.Text))
			}
		}

		if segmentedLine.Flags&parser.Revision == parser.Revision {
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
			} else {
				style = style.Inherit(s.textStyle)
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
		if segmentedLine.Flags&parser.Revision == parser.Revision && row.IsAffected {
			style := s.dimmedStyle
			if s.isHighlighted {
				style = s.dimmedStyle.Background(s.selectedStyle.GetBackground())
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		if s.isHighlighted && segmentedLine.Flags&parser.Highlightable == parser.Highlightable {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.textStyle.GetBackground())))
		}
		fmt.Fprint(r, "\n")
	}

	if row.Commit.IsRoot() {
		return
	}

	if afterSection := s.RenderAfter(row.Commit); afterSection != "" {
		extended := row.Extend()
		s.writeSection(r, extended, extended, false, afterSection)
	}

	for segmentedLine := range row.RowLinesIter(parser.Excluding(parser.Highlightable)) {
		var lw strings.Builder
		for _, segment := range segmentedLine.Gutter.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(s.textStyle).Render(segment.Text))
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
func (s *DefaultRowIterator) writeSection(r io.Writer, current parser.GraphGutter, extended parser.GraphGutter, highlight bool, section string) {
	lines := strings.Split(section, "\n")
	for _, sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range current.Segments {
			if s.isHighlighted && highlight {
				fmt.Fprint(&lw, segment.Style.Inherit(s.selectedStyle).Render(segment.Text))
			} else {
				fmt.Fprint(&lw, segment.Style.Inherit(s.textStyle).Render(segment.Text))
			}
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
