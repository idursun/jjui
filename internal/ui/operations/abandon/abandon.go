package abandon

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	model   *confirmation.Model
	context *context.MainContext
}

func (a *Operation) Mount(v *view.ViewNode) {
	a.ViewNode = v
}

func (a *Operation) GetId() view.ViewId {
	return "abandon"
}

func (a *Operation) Init() tea.Cmd {
	return nil
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

func (a *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.model, cmd = a.model.Update(msg)
	return a, cmd
}

func (a *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	current := a.context.Revisions.Current()

	isSelected := commit != nil && current != nil && commit.GetChangeId() == current.Commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return a.View()
}

func (a *Operation) close() tea.Msg {
	a.ViewManager.UnregisterView(a.Id)
	return nil
}

func NewOperation(context *context.MainContext, selectedRevisions jj.SelectedRevisions) *Operation {
	op := &Operation{
		context: context,
	}

	var ids []string
	var conflictingWarning string
	for _, rev := range selectedRevisions {
		ids = append(ids, rev.Commit.GetChangeId())
		if rev.Commit.IsConflicting() {
			conflictingWarning = "conflicting "
		}
	}
	message := fmt.Sprintf("Are you sure you want to abandon this %srevision?", conflictingWarning)
	if len(selectedRevisions) > 1 {
		message = fmt.Sprintf("Are you sure you want to abandon %d %srevisions?", len(selectedRevisions), conflictingWarning)
	}
	cmd := func(ignoreImmutable bool) tea.Cmd {
		return context.RunCommand(jj.Args(jj.AbandonArgs{
			Revisions:       selectedRevisions,
			RetainBookmarks: true,
			GlobalArguments: jj.GlobalArguments{IgnoreImmutable: ignoreImmutable},
		}), common.Refresh, op.close)
	}
	op.model = confirmation.New(
		[]string{message},
		confirmation.WithAltOption("Yes", cmd(false), cmd(true), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", op.close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		confirmation.WithStylePrefix("abandon"),
	)

	return op
}
