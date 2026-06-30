package bookmark

import (
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
)

var _ operations.Operation = (*CreateBookmarkOperation)(nil)
var _ common.Focusable = (*CreateBookmarkOperation)(nil)
var _ common.ScopeProvider = (*CreateBookmarkOperation)(nil)

type CreateBookmarkOperation struct {
	context *context.MainContext
	target  *jj.Commit
	styles  struct {
		targetMarker lipgloss.Style
		dimmed       lipgloss.Style
	}
}

func NewCreateBookmarkOperation(context *context.MainContext, target *jj.Commit) *CreateBookmarkOperation {
	op := &CreateBookmarkOperation{context: context, target: target}
	op.styles.targetMarker = common.DefaultPalette.Get("revisions selected")
	op.styles.dimmed = common.DefaultPalette.Get("revisions dimmed")
	return op
}

func (c *CreateBookmarkOperation) Init() tea.Cmd { return nil }

func (c *CreateBookmarkOperation) IsFocused() bool { return true }

func (c *CreateBookmarkOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeBookmarkTarget,
			Leak:    common.LeakAll,
			Handler: c,
		},
	}
}

func (c *CreateBookmarkOperation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	c.target = commit
	return nil
}

func (c *CreateBookmarkOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Apply:
		if c.target == nil {
			return nil, true
		}
		return intents.Invoke(intents.OpenSetBookmark{Revision: c.target.GetChangeId()}), true
	case intents.Cancel:
		return common.Close, true
	}
	return nil, false
}

func (c *CreateBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		selected, ok := msg.Item.(common.SelectedRevision)
		if !ok {
			return nil
		}
		return c.SetSelectedRevision(&jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId})
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

func (c *CreateBookmarkOperation) Name() string {
	return "create bookmark"
}
