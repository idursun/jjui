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

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	confirmation *confirmation.Model
}

func (m *Model) GetId() view.ViewId {
	return "undo"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	m.Width = 100
	m.Height = 3
	v.Id = "undo"
}

func (m *Model) ShortHelp() []key.Binding {
	return m.confirmation.ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return m.confirmation.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirmation, cmd = m.confirmation.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	return m.confirmation.View()
}

func (m *Model) close() tea.Msg {
	m.ViewManager.UnregisterView(m.Id)
	return nil
}

func NewModel(context *context.MainContext) view.IViewModel {
	output, _ := context.RunCommandImmediate(jj.Args(jj.OpLogArgs{
		NoGraph:         false,
		Limit:           1,
		GlobalArguments: jj.GlobalArguments{IgnoreWorkingCopy: true, Color: "always"},
	}))
	lastOperation := lipgloss.NewStyle().PaddingBottom(1).Render(string(output))
	m := &Model{}
	model := confirmation.New(
		[]string{lastOperation, "Are you sure you want to undo last change?"},
		confirmation.WithStylePrefix("undo"),
		confirmation.WithOption("Yes", context.RunCommand(jj.Args(jj.UndoArgs{}), common.Refresh, m.close), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", m.close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)
	model.Styles.Border = common.DefaultPalette.GetBorder("undo border", lipgloss.NormalBorder()).Padding(1)
	m.confirmation = model
	return m
}
