package describe

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
)

type Model struct {
	context     common.AppContext
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
			return m, m.context.RunCommand(jj.Describe(m.revision, m.description.Value()), common.Close, common.Refresh)
		}
	case tea.WindowSizeMsg:
		m.description.SetWidth(msg.Width)
	}
	var cmd tea.Cmd
	m.description, cmd = m.description.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.description.View()
}

func New(context common.AppContext, revision string, description string, width int) tea.Model {
	t := textarea.New()
	t.SetValue(description)
	t.Focus()
	t.SetWidth(width)
	t.SetHeight(1)
	t.CharLimit = 120
	t.ShowLineNumbers = false
	return Model{
		description: t,
		revision:    revision,
		context:     context,
	}
}
