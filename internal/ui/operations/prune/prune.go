package prune

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Editable = (*Operation)(nil)

type Operation struct {
	model   *confirmation.Model
	current *jj.Commit
	context *context.MainContext
}

func (a *Operation) IsEditing() bool {
	return true
}

func (a *Operation) Init() tea.Cmd {
	return nil
}

func (a *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	return "prune"
}

func NewOperation(context *context.MainContext, selectedRevision *jj.Commit) *Operation {
	message := fmt.Sprintf("Are you sure you want to abandon this revision and all its descendants?")
	cmd := func(ignoreImmutable bool) tea.Cmd {
		return context.RunCommand(jj.Prune(selectedRevision.GetChangeId(), ignoreImmutable), common.Refresh, common.Close)
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
