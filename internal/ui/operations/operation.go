package operations

import (
	"github.com/charmbracelet/lipgloss"
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
	common.Model
	Render(commit *jj.Commit, renderPosition RenderPosition) string
	Name() string
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit)
}

type SegmentRenderer interface {
	RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string
}
