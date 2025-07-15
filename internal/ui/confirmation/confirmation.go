package confirmation

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"strings"
)

var (
	right = key.NewBinding(key.WithKeys("right", "l"))
	left  = key.NewBinding(key.WithKeys("left", "h"))
	enter = key.NewBinding(key.WithKeys("enter"))
)

type CloseMsg struct{}

type option struct {
	label      string
	cmd        tea.Cmd
	keyBinding key.Binding
}

type styles struct {
	borderStyle      lipgloss.Style
	selectedButton   lipgloss.Style
	unselectedButton lipgloss.Style
	text             lipgloss.Style
}

type Model struct {
	message  string
	options  []option
	selected int
	styles   styles
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, left):
			if m.selected > 0 {
				m.selected--
			}
		case key.Matches(msg, right):
			if m.selected < len(m.options) {
				m.selected++
			}
		case key.Matches(msg, enter):
			selectedOption := m.options[m.selected]
			return m, selectedOption.cmd
		default:
			for _, option := range m.options {
				if key.Matches(msg, option.keyBinding) {
					return m, option.cmd
				}
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	w := strings.Builder{}
	w.WriteString(m.styles.text.Render(m.message))
	for i, option := range m.options {
		w.WriteString(" ")
		if i == m.selected {
			w.WriteString(m.styles.selectedButton.Render(option.label))
		} else {
			w.WriteString(m.styles.unselectedButton.Render(option.label))
		}
	}
	return m.styles.borderStyle.Render(w.String())
}

func (m *Model) AddOption(label string, cmd tea.Cmd, keyBinding key.Binding) {
	m.options = append(m.options, option{label, cmd, keyBinding})
}

func (m *Model) SetBorderStyle(style lipgloss.Style) {
	m.styles.borderStyle = style
}

func New(message string) Model {
	styles := styles{
		borderStyle:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1, 0, 1),
		text:             common.DefaultPalette.Get("confirmation text"),
		selectedButton:   common.DefaultPalette.Get("confirmation selected").PaddingLeft(2).PaddingRight(2),
		unselectedButton: common.DefaultPalette.Get("confirmation dimmed").PaddingLeft(2).PaddingRight(2),
	}
	return Model{
		message:  message,
		options:  []option{},
		selected: 0,
		styles:   styles,
	}
}

func Close() tea.Msg {
	return CloseMsg{}
}
