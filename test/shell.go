package test

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

type model struct {
	embeddedModel tea.Model
}

func (m model) Init() tea.Cmd {
	return m.embeddedModel.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var shellCmd tea.Cmd
	if _, ok := msg.(common.CloseViewMsg); ok {
		shellCmd = tea.Quit
	}
	var cmd tea.Cmd
	m.embeddedModel, cmd = m.embeddedModel.Update(msg)
	return m, tea.Sequence(cmd, shellCmd)
}

func (m model) View() string {
	return m.embeddedModel.View()
}

func NewShell(details tea.Model) tea.Model {
	return model{
		embeddedModel: details,
	}
}
