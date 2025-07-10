package graph

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowIterator struct {
	Palette             common.Palette
	HighlightBackground lipgloss.AdaptiveColor
	SearchText          string
	Selections          map[string]bool
	Op                  operations.Operation
	Width               int
	Rows                []parser.Row
	isHighlighted       bool
	isSelected          bool
	current             int
	Cursor              int
	highlightSeq        string
}

func NewDefaultRowIterator(rows []parser.Row, width int) *DefaultRowIterator {
	highlightBackground := lipgloss.AdaptiveColor{
		Light: config.Current.UI.HighlightLight,
		Dark:  config.Current.UI.HighlightDark,
	}
	highlightColor := highlightBackground.Light
	if lipgloss.HasDarkBackground() {
		highlightColor = highlightBackground.Dark
	}
	highlightSeq := lipgloss.ColorProfile().Color(highlightColor).Sequence(true)
	return &DefaultRowIterator{
		Palette:             common.DefaultPalette,
		HighlightBackground: highlightBackground,
		Op:                  &operations.Default{},
		Width:               width,
		Rows:                rows,
		Selections:          make(map[string]bool),
		current:             -1,
		highlightSeq:        highlightSeq,
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
			if s.isHighlighted {
				segment = segment.WithBackground(s.highlightSeq)
			}

			if s.isHighlighted && s.SearchText != "" && strings.Contains(segment.Text, s.SearchText) {
				for _, part := range segment.Reverse(s.SearchText) {
					fmt.Fprint(&lw, part.String())
				}
			} else {
				fmt.Fprint(&lw, segment.String())
			}
		}
		if segmentedLine.Flags&parser.Revision == parser.Revision && row.IsAffected {
			style := common.DefaultPalette.Dimmed
			if s.isHighlighted {
				style = common.DefaultPalette.Dimmed.Background(s.HighlightBackground)
			}
			fmt.Fprint(&lw, style.Render(" (affected by last operation)"))
		}
		line := lw.String()
		fmt.Fprint(r, line)
		if s.isHighlighted {
			lineWidth := lipgloss.Width(line)
			gap := s.Width - lineWidth
			if gap > 0 {
				fmt.Fprintf(r, "\033[%sm%s\033[0m", s.highlightSeq, strings.Repeat(" ", gap))
			}
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
		for _, segment := range segmentedLine.Segments {
			fmt.Fprint(r, segment.String())
		}
		fmt.Fprint(r, "\n")
	}
}

func (s *DefaultRowIterator) writeSection(r io.Writer, extended parser.GraphRowLine, indent int, highlight bool, section string) {
	lines := strings.Split(section, "\n")
	for _, sectionLine := range lines {
		lw := strings.Builder{}
		for _, segment := range extended.Segments {
			if s.isHighlighted && highlight {
				segment = segment.WithBackground(s.highlightSeq)
			}
			fmt.Fprint(&lw, segment.String())
		}

		fmt.Fprint(&lw, sectionLine)
		line := lw.String()
		fmt.Fprint(r, line)
		if s.isHighlighted && highlight {
			lineWidth := lipgloss.Width(line)
			gap := s.Width - lineWidth
			if gap > 0 {
				fmt.Fprintf(r, "\033[%sm%s\033[0m", s.highlightSeq, strings.Repeat(" ", gap))
			}
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
			selectedMarker = s.Palette.Added.Background(s.HighlightBackground).Render("✓ ")
		} else {
			selectedMarker = s.Palette.Added.Render("✓ ")
		}
	}
	return opMarker + selectedMarker
}

func (s *DefaultRowIterator) RenderBeforeCommitId(commit *jj.Commit) string {
	return s.Op.Render(commit, operations.RenderBeforeCommitId)
}
