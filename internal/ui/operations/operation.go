package operations

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
)

type RenderPosition int

const (
	RenderPositionNil RenderPosition = iota
	RenderPositionAfter
	RenderPositionBefore
	RenderBeforeChangeId
	RenderBeforeCommitId
	RenderOverDescription
)

type Operation interface {
	tea.Model
	Render(commit *jj.Commit, renderPosition RenderPosition) string
	Name() string
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit)
}

type SegmentRenderer interface {
	RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string
}
