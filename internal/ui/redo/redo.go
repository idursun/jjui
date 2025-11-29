package redo

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
)

var _ common.Model = (*Model)(nil)

type Model struct {
	*common.ViewNode
	confirmation *confirmation.Model
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

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	return m.confirmation.Update(msg)
}

func (m *Model) View() string {
	v := m.confirmation.View()
	w, h := lipgloss.Size(v)
	pw, ph := m.Parent.Width, m.Parent.Height
	sx := (pw - w) / 2
	sy := (ph - h) / 2
	m.SetFrame(cellbuf.Rect(sx, sy, w, h))
	return v
}

func NewModel(context *context.MainContext) *Model {
	output, _ := context.RunCommandImmediate(jj.OpLog(1))
	lastOperation := lipgloss.NewStyle().PaddingBottom(1).Render(string(output))
	model := confirmation.New(
		[]string{lastOperation, "Are you sure you want to redo last change?"},
		confirmation.WithStylePrefix("redo"),
		confirmation.WithOption("Yes", context.RunCommand(jj.Redo(), common.Refresh, common.Close), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)
	model.Styles.Border = common.DefaultPalette.GetBorder("redo border", lipgloss.NormalBorder()).Padding(1)
	return &Model{
		ViewNode:     common.NewViewNode(0, 0),
		confirmation: model,
	}
}
