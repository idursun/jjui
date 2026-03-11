package bookmark

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*MoveBookmarkOperation)(nil)
var _ operations.TracksSelectedRevision = (*MoveBookmarkOperation)(nil)
var _ common.Focusable = (*MoveBookmarkOperation)(nil)

type MoveBookmarkOperation struct {
	context      *context.MainContext
	bookmarkName string
	target       *jj.Commit
	styles       struct {
		targetMarker lipgloss.Style
		dimmed       lipgloss.Style
		changeId     lipgloss.Style
	}
}

func NewMoveBookmarkOperation(context *context.MainContext, bookmarkName string) *MoveBookmarkOperation {
	op := &MoveBookmarkOperation{
		context:      context,
		bookmarkName: bookmarkName,
	}
	op.styles.targetMarker = common.DefaultPalette.Get("revisions selected")
	op.styles.dimmed = common.DefaultPalette.Get("revisions dimmed")
	op.styles.changeId = common.DefaultPalette.Get("revisions text")
	return op
}

func (m *MoveBookmarkOperation) Init() tea.Cmd { return nil }

func (m *MoveBookmarkOperation) IsFocused() bool { return true }

func (m *MoveBookmarkOperation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	m.target = commit
	return nil
}

func (m *MoveBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		switch msg := msg.(type) {
		case intents.Apply:
			if m.target == nil {
				return nil
			}
			var extraFlags []string
			if msg.Force {
				extraFlags = append(extraFlags, "--allow-backwards")
			}
			return m.context.RunCommand(
				jj.BookmarkMove(m.target.GetChangeId(), m.bookmarkName, extraFlags...),
				common.CloseApplied,
				common.Refresh,
				common.FocusBookmarkView(),
			)
		case intents.Cancel:
			return tea.Sequence(common.Close, common.FocusBookmarkView())
		}
	}
	return nil
}

func (m *MoveBookmarkOperation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (m *MoveBookmarkOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderPositionBefore || m.target == nil || commit == nil || m.target.GetChangeId() != commit.GetChangeId() {
		return ""
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.styles.targetMarker.Render("<< onto >>"),
		m.styles.dimmed.Render(" move bookmark "),
		m.styles.changeId.Render(m.bookmarkName),
	)
}

func (m *MoveBookmarkOperation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ layout.Rectangle, _ layout.Position) int {
	return 0
}

func (m *MoveBookmarkOperation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (m *MoveBookmarkOperation) Name() string {
	return fmt.Sprintf("move bookmark %s", m.bookmarkName)
}
