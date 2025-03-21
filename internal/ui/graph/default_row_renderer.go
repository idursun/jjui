package graph

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowRenderer struct {
	Palette             common.Palette
	HighlightBackground lipgloss.AdaptiveColor
	IsHighlighted       bool
	IsSelected          bool
	IsAffected          bool
	Op                  operations.Operation
}

func (s *DefaultRowRenderer) RenderBefore(*jj.Commit) string {
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionBefore {
		return s.Op.Render()
	}
	return ""
}

func (s *DefaultRowRenderer) RenderAfter(*jj.Commit) string {
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionAfter {
		return s.Op.Render()
	}
	return ""
}

func (s *DefaultRowRenderer) RenderBeforeChangeId() string {
	opMarker := ""
	if s.IsHighlighted {
		if s.Op.RenderPosition() == operations.RenderPositionGlyph {
			opMarker = s.Op.Render()
		}
	}
	selectedMarker := ""
	if s.IsSelected {
		selectedMarker = s.Palette.Added.Render("✓")
	}
	return opMarker + selectedMarker
}

func (s *DefaultRowRenderer) RenderBeforeCommitId() string {
	if s.Op.RenderPosition() == operations.RenderPositionBookmark {
		return s.Op.Render()
	}
	return ""
}
