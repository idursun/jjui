package abandon

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Editable = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)

type Operation struct {
	model   *confirmation.Model
	current *jj.Commit
	context *context.MainContext
}

func (a *Operation) GetActionMap() map[string]common.Action {
	return map[string]common.Action{
		"y":         {Id: "abandon.accept", Args: nil},
		"alt+enter": {Id: "abandon.force_apply", Args: nil},
		"n":         {Id: "close abandon", Args: nil},
		"esc":       {Id: "close abandon", Args: nil},
	}
}

func (a *Operation) IsEditing() bool {
	return true
}

func (a *Operation) Init() tea.Cmd {
	return nil
}

func (a *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(common.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "abandon.accept", "abandon.force_apply":
			ignoreImmutable := msg.Action.Id == "abandon.force_apply"
			return a, a.context.RunCommand(jj.Abandon(jj.SelectedRevisions{Revisions: []*jj.Commit{a.current}}, ignoreImmutable), common.Refresh)
		}
	}
	var cmd tea.Cmd
	a.model, cmd = a.model.Update(msg)
	return a, cmd
}

func (a *Operation) View() string {
	return a.model.View()
}

func (a *Operation) ShortHelp() []key.Binding {
	return a.model.ShortHelp()
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
	cmd := func(ignoreImmutable bool) tea.Cmd {
		return context.RunCommand(jj.Abandon(selectedRevisions, ignoreImmutable), common.Refresh, common.Close)
	}
	model := confirmation.New(
		[]string{message},
		confirmation.WithAltOption("Yes", cmd(false), cmd(true), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		confirmation.WithStylePrefix("abandon"),
	)

	op := &Operation{
		model: model,
	}
	return op
}
