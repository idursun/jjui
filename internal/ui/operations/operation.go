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
	Name() string
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit) tea.Cmd
}

type SegmentRenderer interface {
	RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string
}

// DisplayListRenderer is an optional interface for operations that support
// rendering directly to a DisplayList. This enables mouse interactions.
type DisplayListRenderer interface {
	// RenderToDisplayList renders the operation content to the DisplayList.
	// - rect: the area to render in (relative coordinates for the render buffer)
	// - screenOffset: the offset to add for mouse interactions (absolute screen position)
	// Returns the height in lines that was rendered.
	RenderToDisplayList(dl *render.DisplayList, commit *jj.Commit, pos RenderPosition, rect cellbuf.Rectangle, screenOffset cellbuf.Position) int

	// SupportsDisplayList returns true if DisplayList rendering should be used
	// for the given position. Operations can support DisplayList for some
	// positions but not others.
	SupportsDisplayList(pos RenderPosition) bool
}
