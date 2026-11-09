package bookmark

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

var _ operations.Operation = (*MoveBookmarkOperation)(nil)
var _ common.Focusable = (*MoveBookmarkOperation)(nil)
var _ common.ScopeProvider = (*MoveBookmarkOperation)(nil)

type MoveBookmarkOperation struct {
	context       *context.MainContext
	bookmarkNames []string
	target        *jj.Commit
	styles        struct {
		targetMarker lipgloss.Style
		dimmed       lipgloss.Style
		changeId     lipgloss.Style
	}
}

func NewMoveBookmarkOperation(context *context.MainContext, bookmarkNames []string, target *jj.Commit) *MoveBookmarkOperation {
	op := &MoveBookmarkOperation{
		context:       context,
		bookmarkNames: bookmarkNames,
		target:        target,
	}
	op.styles.targetMarker = theme.DefaultPalette.Get("revisions", "", "", true)
	op.styles.dimmed = theme.DefaultPalette.Get("revisions", "", "dimmed", false)
	op.styles.changeId = theme.DefaultPalette.Get("revisions", "", "text", false)
	return op
}

func (m *MoveBookmarkOperation) Init() tea.Cmd { return nil }

func (m *MoveBookmarkOperation) IsFocused() bool { return true }

func (m *MoveBookmarkOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeBookmarkTarget,
			Leak:    common.LeakAll,
			Handler: m,
		},
	}
}

func (m *MoveBookmarkOperation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	m.target = commit
	return nil
}

func (m *MoveBookmarkOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Apply:
		if m.target == nil {
			return nil, true
		}
		var extraFlags []string
		if intent.Force {
			extraFlags = append(extraFlags, "--allow-backwards")
		}
		return m.context.RunCommand(
			jj.BookmarkMove(m.target.GetChangeId(), m.bookmarkNames, extraFlags...),
			common.CloseApplied,
			common.Refresh,
		), true
	case intents.Cancel:
		return common.Close, true
	}
	return nil, false
}

func (m *MoveBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		selected, ok := msg.Item.(common.SelectedRevision)
		if !ok {
			return nil
		}
		return m.SetSelectedRevision(&jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId})
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
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
		m.styles.changeId.Render(strings.Join(m.bookmarkNames, ", ")),
	)
}

func (m *MoveBookmarkOperation) Name() string {
	return fmt.Sprintf("move bookmark %s", strings.Join(m.bookmarkNames, ", "))
}
