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

type moveDirection string

const (
	moveForward   moveDirection = "forward"
	moveBackwards moveDirection = "backwards"
	moveSideways  moveDirection = "sideways"
)

// moveDirectionMsg is the direction of a move onto target.
type moveDirectionMsg struct {
	target    string
	direction moveDirection
}

type MoveBookmarkOperation struct {
	context       *context.MainContext
	bookmarkNames []string
	target        *jj.Commit
	// direction is the move onto directionTarget, which lags the target while
	// jj is asked.
	direction       moveDirection
	directionTarget string
	styles          struct {
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

func (m *MoveBookmarkOperation) Init() tea.Cmd { return m.detectDirection() }

func (m *MoveBookmarkOperation) detectDirection() tea.Cmd {
	if m.target == nil {
		return nil
	}
	target := m.target.GetChangeId()
	return func() tea.Msg {
		return moveDirectionMsg{target: target, direction: m.queryDirection(target)}
	}
}

// queryDirection is forward unless a bookmark would move backwards or sideways.
func (m *MoveBookmarkOperation) queryDirection(target string) moveDirection {
	output, err := m.context.RunCommandImmediate(jj.BookmarkMoveDirections(target, m.bookmarkNames))
	if err != nil {
		return ""
	}
	direction := moveForward
	for line := range strings.SplitSeq(string(output), "\n") {
		switch strings.TrimSpace(line) {
		case "s":
			return moveSideways
		case "b":
			direction = moveBackwards
		}
	}
	return direction
}

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
	return m.detectDirection()
}

func (m *MoveBookmarkOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Apply:
		if m.target == nil {
			return nil, true
		}
		direction := m.direction
		if m.directionTarget != m.target.GetChangeId() {
			direction = m.queryDirection(m.target.GetChangeId())
		}
		var extraFlags []string
		if intent.Force || direction != moveForward {
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
	case moveDirectionMsg:
		// Replies for targets the cursor has moved past can arrive late.
		if m.target != nil && msg.target == m.target.GetChangeId() {
			m.direction, m.directionTarget = msg.direction, msg.target
		}
		return nil
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
	parts := []string{
		m.styles.targetMarker.Render("<< onto >>"),
		m.styles.dimmed.Render(" move bookmark "),
		m.styles.changeId.Render(strings.Join(m.bookmarkNames, ", ")),
	}
	if m.directionTarget == commit.GetChangeId() && (m.direction == moveBackwards || m.direction == moveSideways) {
		parts = append(parts, m.styles.dimmed.Render(" "+string(m.direction)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m *MoveBookmarkOperation) Name() string {
	return fmt.Sprintf("move bookmark %s", strings.Join(m.bookmarkNames, ", "))
}
