package test

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"time"
)

type model struct {
	closed        bool
	embeddedModel tea.Model
}

func (m model) Init() tea.Cmd {
	return m.embeddedModel.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.CloseViewMsg, confirmation.CloseMsg:
		m.closed = true
		// give enough time to clear pending messages before quitting
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tea.QuitMsg{}
		})
	default:
		var cmd tea.Cmd
		m.embeddedModel, cmd = m.embeddedModel.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.closed {
		return "closed"
	}
	return m.embeddedModel.View()
}

func NewShell(embeddedModel tea.Model) tea.Model {
	return model{
		embeddedModel: embeddedModel,
	}
}
