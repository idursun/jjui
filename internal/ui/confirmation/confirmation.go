package confirmation

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/idursun/jjui/internal/ui/common"

	tea "github.com/charmbracelet/bubbletea"
)

type CloseMsg struct{}

type option struct {
	label string
	cmd   tea.Cmd
}

type Model struct {
	message  string
	options  []option
	selected int
}

var (
	textStyle   = lipgloss.NewStyle().Bold(true).Foreground(common.Magenta)
	normalStyle = lipgloss.NewStyle().
			Foreground(common.White).
			PaddingLeft(2).
			PaddingRight(2)
)

var selectedStyle = lipgloss.NewStyle().
	Foreground(common.DarkWhite).
	Background(common.Blue).
	PaddingLeft(2).
	PaddingRight(2)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyLeft:
			if m.selected > 0 {
				m.selected--
			}
		case tea.KeyRight:
			if m.selected < len(m.options) {
				m.selected++
			}
		case tea.KeyEnter:
			selectedOption := m.options[m.selected]
			return m, selectedOption.cmd
		case tea.KeyEscape:
			return m, common.Close
		default:
		}
	}
	return m, nil
}

func (m Model) View() string {
	w := strings.Builder{}
	w.WriteString(textStyle.Render(m.message))
	for i, option := range m.options {
		w.WriteString(" ")
		if i == m.selected {
			w.WriteString(selectedStyle.Render(option.label))
		} else {
			w.WriteString(normalStyle.Render(option.label))
		}
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1, 0, 1).Render(w.String())
}

func (m *Model) AddOption(label string, cmd tea.Cmd) {
	m.options = append(m.options, option{label, cmd})
}

func New(message string) Model {
	return Model{
		message:  message,
		options:  []option{},
		selected: 0,
	}
}

func Close() tea.Msg {
	return CloseMsg{}
}
