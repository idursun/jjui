package bookmark

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*CreateBookmarkOperation)(nil)
var _ operations.TracksSelectedRevision = (*CreateBookmarkOperation)(nil)
var _ common.Focusable = (*CreateBookmarkOperation)(nil)
var _ dispatch.ScopeProvider = (*CreateBookmarkOperation)(nil)

type CreateBookmarkOperation struct {
	context *context.MainContext
	target  *jj.Commit
	styles  struct {
		targetMarker lipgloss.Style
		dimmed       lipgloss.Style
	}
}

func NewCreateBookmarkOperation(context *context.MainContext) *CreateBookmarkOperation {
	op := &CreateBookmarkOperation{context: context}
	op.styles.targetMarker = common.DefaultPalette.Get("revisions selected")
	op.styles.dimmed = common.DefaultPalette.Get("revisions dimmed")
	return op
}

func (c *CreateBookmarkOperation) Init() tea.Cmd { return nil }

func (c *CreateBookmarkOperation) IsFocused() bool { return true }

func (c *CreateBookmarkOperation) Scopes() []dispatch.Scope {
	return []dispatch.Scope{
		{
			Name:    actions.ScopeBookmarkMove,
			Leak:    dispatch.LeakAll,
			Handler: c,
		},
	}
}

func (c *CreateBookmarkOperation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	c.target = commit
	return nil
}

func (c *CreateBookmarkOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Apply:
		_ = intent
		if c.target == nil {
			return nil, true
		}
		return tea.Sequence(
			common.Close,
			func() tea.Msg {
				return common.StartSetBookmarkMsg{Revision: c.target.GetChangeId(), ReturnFocusToBookmarkView: true}
			},
		), true
	case intents.Cancel:
		return tea.Sequence(common.Close, common.FocusBookmarkView()), true
	}
	return nil, false
}

func (c *CreateBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := c.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (c *CreateBookmarkOperation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (c *CreateBookmarkOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderPositionBefore || c.target == nil || commit == nil || c.target.GetChangeId() != commit.GetChangeId() {
		return ""
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		c.styles.targetMarker.Render("<< create on >>"),
		c.styles.dimmed.Render(" bookmark"),
	)
}

func (c *CreateBookmarkOperation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ layout.Rectangle, _ layout.Position) int {
	return 0
}

func (c *CreateBookmarkOperation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (c *CreateBookmarkOperation) Name() string {
	return fmt.Sprintf("create bookmark")
}
