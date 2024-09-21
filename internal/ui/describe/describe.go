package describe

import (
	"jjui/internal/ui/common"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	revision    string
	description textarea.Model
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, common.Close
		case "enter":
			return m, tea.Batch(common.Close, common.UpdateDescription(m.revision, m.description.Value()))
		}
	}
	var cmd tea.Cmd
	m.description, cmd = m.description.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.description.View()
}

func New(revision string, description string) tea.Model {
	t := textarea.New()
	t.SetValue(description)
	t.Focus()
	t.CharLimit = 80
	return Model{
		description: t,
		revision:    revision,
	}
}