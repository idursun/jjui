package redo

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
)

type Model struct {
	confirmation *confirmation.Model
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

func (m Model) Update(msg tea.Msg) (common.Stackable, tea.Cmd) {
	var cmd tea.Cmd
	m.confirmation, cmd = m.confirmation.Update(msg)
	return m, cmd
}

func (m Model) View() *lipgloss.Layer {
	return lipgloss.NewLayer(m.confirmation.View())
}

func NewModel(context *context.MainContext) Model {
	output, _ := context.RunCommandImmediate(jj.OpLog(1))
	lastOperation := lipgloss.NewStyle().PaddingBottom(1).Render(string(output))
	model := confirmation.New(
		[]string{lastOperation, "Are you sure you want to redo last change?"},
		confirmation.WithStylePrefix("redo"),
		confirmation.WithOption("Yes", context.RunCommand(jj.Redo(), common.Refresh, common.Close), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)
	model.Styles.Border = common.DefaultPalette.GetBorder("redo border", lipgloss.NormalBorder()).Padding(1)
	return Model{
		confirmation: model,
	}
}
