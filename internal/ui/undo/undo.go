package undo

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	confirmation *confirmation.Model
	context      *context.MainContext
}

func (m Model) GetActionMap() map[string]common.Action {
	return map[string]common.Action{
		"y": {Id: "undo.accept", Args: nil, Switch: common.ScopeRevisions},
		"n": {Id: "close undo", Next: []common.Action{
			{Id: "switch revisions"},
		}},
		"esc": {Id: "close undo", Next: []common.Action{
			{Id: "switch revisions"},
		}},
	}
}

func (m Model) ShortHelp() []key.Binding {
	return m.confirmation.ShortHelp()
}

func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m Model) Init() tea.Cmd {
	return m.confirmation.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(common.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "undo.accept":
			return m, tea.Batch(common.InvokeAction(common.Action{Id: "close undo", Switch: common.ScopeRevisions}), m.context.RunCommand(jj.Undo(), common.Refresh, common.Close))
		}
	}
	var cmd tea.Cmd
	m.confirmation, cmd = m.confirmation.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.confirmation.View()
}

func NewModel(context *context.MainContext) Model {
	output, _ := context.RunCommandImmediate(jj.OpLog(1))
	lastOperation := lipgloss.NewStyle().PaddingBottom(1).Render(string(output))
	model := confirmation.New(
		[]string{lastOperation, "Are you sure you want to undo last change?"},
		confirmation.WithStylePrefix("undo"),
		confirmation.WithOption("Yes", context.RunCommand(jj.Undo(), common.Refresh, common.Close), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)
	model.Styles.Border = common.DefaultPalette.GetBorder("undo border", lipgloss.NormalBorder()).Padding(1)
	return Model{
		context:      context,
		confirmation: model,
	}
}
