package operations

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/render"
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
	RenderToDisplayContext(dl *render.DisplayContext, commit *jj.Commit, pos RenderPosition, rect cellbuf.Rectangle, screenOffset cellbuf.Position) int
	DesiredHeight(commit *jj.Commit, pos RenderPosition) int
	Name() string
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit) tea.Cmd
}

type SegmentRenderer interface {
	RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string
}
