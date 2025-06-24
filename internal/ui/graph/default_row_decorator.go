package graph

import (
	"github.com/charmbracelet/lipgloss/v2/compat"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowDecorator struct {
	Palette             common.Palette
	HighlightBackground compat.AdaptiveColor
	SearchText          string
	IsHighlighted       bool
	IsSelected          bool
	Op                  operations.Operation
	Width               int
}

func (s *DefaultRowDecorator) RenderBefore(*jj.Commit) string {
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionBefore {
		return s.Op.Render()
	}
	return ""
}

func (s *DefaultRowDecorator) RenderAfter(*jj.Commit) string {
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionAfter {
		return s.Op.Render()
	}
	return ""
}

func (s *DefaultRowDecorator) RenderBeforeChangeId() string {
	opMarker := ""
	if s.IsHighlighted {
		if s.Op.RenderPosition() == operations.RenderBeforeChangeId {
			opMarker = s.Op.Render()
		}
	}
	selectedMarker := ""
	if s.IsSelected {
		if s.IsHighlighted {
			selectedMarker = s.Palette.Added.Background(s.HighlightBackground).Render("✓ ")
		} else {
			selectedMarker = s.Palette.Added.Render("✓ ")
		}
	}
	return opMarker + selectedMarker
}

func (s *DefaultRowDecorator) RenderBeforeCommitId() string {
	if s.Op.RenderPosition() == operations.RenderBeforeCommitId {
		return s.Op.Render()
	}
	return ""
}
