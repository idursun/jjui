package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowIterator struct {
	SearchText    string
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
	TextStyle     lipgloss.Style
	SelectedStyle lipgloss.Style
}

func NewDefaultRowIterator(rows []parser.Row, width int) *DefaultRowIterator {
	return &DefaultRowIterator{
		Op:            &operations.Default{},
		Width:         width,
		Rows:          rows,
		Selections:    make(map[string]bool),
		current:       -1,
		dimmedStyle:   common.DefaultPalette.Get("dimmed"),
		checkStyle:    common.DefaultPalette.Get("success").Inline(true),
		TextStyle:     common.DefaultPalette.Get("text").Inline(true),
		SelectedStyle: common.DefaultPalette.Get("selected").Inline(true),
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

func (s *DefaultRowIterator) Render(r io.Writer) {
	row := s.Rows[s.current]
	// will render by extending the previous connections
	if before := s.RenderBefore(row.Commit); before != "" {
		extended := parser.GraphRowLine{}
		if row.Previous != nil {
			extended = row.Previous.Last(parser.Highlightable).Extend(row.Indent)
		}
		s.writeSection(r, extended, row.Indent, false, before)
	}
	var lastLine *parser.GraphRowLine
	for segmentedLine := range row.RowLinesIter(parser.Including(parser.Highlightable)) {
		lastLine = segmentedLine
		lw := strings.Builder{}
		if segmentedLine.Flags&parser.Revision != parser.Revision && s.isHighlighted {
			if decoration := s.Op.Render(row.Commit, operations.RenderOverDescription); decoration != "" {
				extended := segmentedLine.Chop(row.Indent)
				s.writeSection(r, extended, row.Indent, true, decoration)
				continue
			}
		}
		// if it is a revision line
		for i, segment := range segmentedLine.Segments {
			if i == segmentedLine.ChangeIdIdx {
				if decoration := s.RenderBeforeChangeId(row.Commit); decoration != "" {
					fmt.Fprint(&lw, decoration)
				}
			}

			if s.isHighlighted && i == segmentedLine.CommitIdIdx {
				if decoration := s.RenderBeforeCommitId(row.Commit); decoration != "" {
					fmt.Fprint(&lw, decoration)
				}
			}

			style := segment.Style
			if s.isHighlighted {
				style = style.Inherit(s.SelectedStyle)
			} else {
				style = style.Inherit(s.TextStyle)
			}

			start, end := segment.FindSubstringRange(s.SearchText)
			if start != -1 {
				mid := lipgloss.NewRange(start, end, style.Reverse(true))
				fmt.Fprint(&lw, lipgloss.StyleRanges(style.Render(segment.Text), mid))
			} else {
				fmt.Fprint(&lw, style.Render(segment.Text))
			}
		}
		if segmentedLine.Flags&parser.Revision == parser.Revision && row.IsAffected {
			style := s.dimmedStyle
			if s.isHighlighted {
				style = s.dimmedStyle.Background(s.SelectedStyle.GetBackground())
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		if s.isHighlighted && segmentedLine.Flags&parser.Highlightable == parser.Highlightable {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.SelectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.TextStyle.GetBackground())))
		}
		fmt.Fprint(r, "\n")
	}

	if row.Commit.IsRoot() {
		return
	}

	afterSection := s.RenderAfter(row.Commit)
	if afterSection != "" && lastLine != nil {
		s.writeSection(r, lastLine.Extend(row.Indent), row.Indent, false, afterSection)
	}

	for segmentedLine := range row.RowLinesIter(parser.Excluding(parser.Highlightable)) {
		var lw strings.Builder
		for _, segment := range segmentedLine.Segments {
			fmt.Fprint(&lw, segment.String())
		}
		line := lw.String()
		if s.isHighlighted && segmentedLine.Flags&parser.Highlightable == parser.Highlightable {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.SelectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.TextStyle.GetBackground())))
		}
		fmt.Fprint(r, "\n")
	}
}

func (s *DefaultRowIterator) writeSection(r io.Writer, extended parser.GraphRowLine, indent int, highlight bool, section string) {
	lines := strings.Split(section, "\n")
	for _, sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range extended.Segments {
			style := segment.Style
			if s.isHighlighted && highlight {
				fmt.Fprint(&lw, segment.Style.Inherit(s.SelectedStyle).Render(segment.Text))
				style = style.Background(s.SelectedStyle.GetBackground())
			} else {
				fmt.Fprint(&lw, segment.Style.Inherit(s.TextStyle).Render(segment.Text))
			}
		}

		fmt.Fprint(&lw, sectionLine)
		line := lw.String()
		if s.isHighlighted && highlight {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.SelectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(s.Width, 0, line, lipgloss.WithWhitespaceBackground(s.TextStyle.GetBackground())))
		}
		fmt.Fprintln(r)
		extended = extended.Extend(indent)
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
			selectedMarker = s.checkStyle.Background(s.SelectedStyle.GetBackground()).Render("✓ ")
		} else {
			selectedMarker = s.checkStyle.Background(s.TextStyle.GetBackground()).Render("✓ ")
		}
	}
	return opMarker + selectedMarker
}

func (s *DefaultRowIterator) RenderBeforeCommitId(commit *jj.Commit) string {
	return s.Op.Render(commit, operations.RenderBeforeCommitId)
}
