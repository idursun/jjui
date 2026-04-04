package operations

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
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
	common.ImmediateModel
	Render(commit *jj.Commit, renderPosition RenderPosition) string
	Name() string
}

type EmbeddedOperation interface {
	Operation
	CanEmbed(commit *jj.Commit, pos RenderPosition) bool
	EmbeddedHeight(commit *jj.Commit, pos RenderPosition, width int) int
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit) tea.Cmd
}

type SegmentRenderer interface {
	RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string
}
