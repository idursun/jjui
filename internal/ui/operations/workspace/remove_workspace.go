package workspace

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*RemoveWorkspaceOperation)(nil)
var _ operations.EmbeddedOperation = (*RemoveWorkspaceOperation)(nil)
var _ common.Editable = (*RemoveWorkspaceOperation)(nil)
var _ common.ScopeProvider = (*RemoveWorkspaceOperation)(nil)

// RemoveWorkspaceOperation asks before running jj workspace remove, which
// deletes the workspace directory; jj undo does not bring the files back.
type RemoveWorkspaceOperation struct {
	revision     string
	confirmation *confirmation.Model
}

func NewRemoveWorkspaceOperation(context *context.MainContext, changeId string, root string, name string) *RemoveWorkspaceOperation {
	model := confirmation.New(
		[]string{
			root + " will be deleted from disk; jj undo cannot restore it.",
			"Remove workspace " + name + "?",
		},
		confirmation.WithStyleScope("revisions"),
		confirmation.WithOption("Yes",
			context.RunCommand(jj.WorkspaceRemove(name), common.CloseApplied, common.Refresh),
			key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No",
			confirmation.Close,
			key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		// Start on No so that enter alone does not delete anything.
		confirmation.WithSelected(1),
	)
	return &RemoveWorkspaceOperation{
		revision:     changeId,
		confirmation: model,
	}
}

func (r *RemoveWorkspaceOperation) IsEditing() bool {
	return true
}

func (r *RemoveWorkspaceOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeRemoveWorkspace,
			Leak:    common.LeakNone,
			Handler: r,
		},
	}
}

func (r *RemoveWorkspaceOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Apply, intents.Cancel, intents.OptionSelect:
		return r.confirmation.Update(intent), true
	}
	return nil, false
}

func (r *RemoveWorkspaceOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case confirmation.CloseMsg:
		return common.Close
	case confirmation.SelectOptionMsg, tea.KeyMsg:
		return r.confirmation.Update(msg)
	case intents.Intent:
		cmd, _ := r.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (r *RemoveWorkspaceOperation) Init() tea.Cmd {
	return nil
}

func (r *RemoveWorkspaceOperation) CanEmbed(commit *jj.Commit, pos operations.RenderPosition) bool {
	return pos == operations.RenderPositionAfter && commit.GetChangeId() == r.revision
}

func (r *RemoveWorkspaceOperation) EmbeddedHeight(commit *jj.Commit, pos operations.RenderPosition, _ int) int {
	if !r.CanEmbed(commit, pos) {
		return 0
	}
	return lipgloss.Height(r.confirmation.View())
}

func (r *RemoveWorkspaceOperation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	r.confirmation.ViewRect(dl, box)
}

func (r *RemoveWorkspaceOperation) Render(*jj.Commit, operations.RenderPosition) string {
	return ""
}

func (r *RemoveWorkspaceOperation) Name() string {
	return "remove workspace"
}
