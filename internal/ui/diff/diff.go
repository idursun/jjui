package diff

import (
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	view viewport.Model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, common.Close
		}
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.view.View()
}

func New(output string, width int, height int) tea.Model {
	view := viewport.New(width, height)
	content := output
	if content == "" {
		content = "(empty)"
	}
	view.SetContent(content)
	return Model{
		view: view,
	}
}
