package abandon

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ operations.Operation = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)
var _ view.ICommandBuilder = (*Operation)(nil)

type Operation struct {
	confirmation      *confirmation.Model
	ignoreImmutable   bool
	current           *jj.Commit
	context           *context.MainContext
	selectedRevisions jj.SelectedRevisions
}

func (a *Operation) GetCommand() jj.CommandArgs {
	return jj.Abandon(a.selectedRevisions, a.ignoreImmutable)
}

func (a *Operation) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings("abandon")
}

func (a *Operation) Init() tea.Cmd {
	return nil
}

func (a *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "--ignore-immutable":
			a.ignoreImmutable = !a.ignoreImmutable
			return a, nil
		case "abandon.apply", "abandon.force_apply":
			ignoreImmutable := msg.Action.Id == "abandon.force_apply"
			return a, a.context.RunCommand(jj.Abandon(a.selectedRevisions, ignoreImmutable), common.Refresh)
		}
	}
	var cmd tea.Cmd
	a.confirmation, cmd = a.confirmation.Update(msg)
	return a, cmd
}

func (a *Operation) View() string {
	return a.confirmation.View()
}

func (a *Operation) ShortHelp() []key.Binding {
	return a.confirmation.ShortHelp()
}

func (a *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{a.ShortHelp()}
}

func (a *Operation) SetSelectedRevision(commit *jj.Commit) {
	a.current = commit
}

func (a *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	isSelected := commit != nil && commit.GetChangeId() == a.current.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return a.View()
}

func (a *Operation) Name() string {
	return "abandon"
}

func NewOperation(context *context.MainContext, selectedRevisions jj.SelectedRevisions) *Operation {
	var ids []string
	var conflictingWarning string
	for _, rev := range selectedRevisions.Revisions {
		ids = append(ids, rev.GetChangeId())
		if rev.IsConflicting() {
			conflictingWarning = "conflicting "
		}
	}
	message := fmt.Sprintf("Are you sure you want to abandon this %srevision?", conflictingWarning)
	if len(selectedRevisions.Revisions) > 1 {
		message = fmt.Sprintf("Are you sure you want to abandon %d %srevisions?", len(selectedRevisions.Revisions), conflictingWarning)
	}
	model := confirmation.New(
		[]string{message},
		confirmation.WithAltOption("Yes", actions.InvokeAction(actions.Action{Id: "abandon.apply"}), actions.InvokeAction(actions.Action{Id: "abandon.force_apply"}), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", actions.InvokeAction(actions.Action{Id: "close abandon"}), key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		confirmation.WithStylePrefix("abandon"),
	)

	op := &Operation{
		context:           context,
		confirmation:      model,
		selectedRevisions: selectedRevisions,
	}
	return op
}
