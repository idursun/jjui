package ui

import (
	"jjui/internal/ui/common"
	"jjui/internal/ui/describe"
	"jjui/internal/ui/revisions"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	models []tea.Model
}

func New() Model {
	return Model{
		models: []tea.Model{revisions.New()},
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, model := range m.models {
		cmds = append(cmds, model.Init())
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ShowDescribeViewMsg:
		d := describe.New(msg.ChangeId, msg.Description)
		m.models = append(m.models, d)
		return m, d.Init()
	case common.CloseViewMsg:
		m.models = m.models[:len(m.models)-1]
		return m, nil
	}
	var cmd tea.Cmd
	top := m.Top()
	top, cmd = top.Update(msg)
	m.models[len(m.models)-1] = top
	return m, cmd
}

func (m Model) View() string {
	views := make([]string, 0)
	for _, model := range m.models {
		views = append(views, model.View())
	}
	return lipgloss.JoinVertical(0, views...)
}

func (m Model) Top() tea.Model {
	return m.models[len(m.models)-1]
}