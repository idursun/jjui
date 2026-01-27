package input

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type SelectedMsg struct {
	Value string
}

type CancelledMsg struct{}

var (
	_ common.ImmediateModel = (*Model)(nil)
	_ help.KeyMap           = (*Model)(nil)
)

type Model struct {
	input  textinput.Model
	title  string
	prompt string
	keymap config.KeyMappings[key.Binding]
	styles styles
}

type styles struct {
	border lipgloss.Style
	text   lipgloss.Style
	title  lipgloss.Style
}

func New() *Model {
	return NewWithTitle("", "")
}

func NewWithTitle(title string, prompt string) *Model {
	keymap := config.Current.GetKeyMap()
	styles := styles{
		border: common.DefaultPalette.GetBorder("input border", lipgloss.RoundedBorder()),
		text:   common.DefaultPalette.Get("input text"),
		title:  common.DefaultPalette.Get("input title"),
	}
	ti := textinput.New()
	ti.Width = 40
	ti.Focus()
	ti.Prompt = prompt
	ti.PromptStyle = styles.text
	if ti.Prompt == "" {
		ti.Prompt = "> "
	}

	return &Model{
		input:  ti,
		title:  title,
		prompt: prompt,
		keymap: keymap,
		styles: styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.input.Focus()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m.selectCurrent()
		case tea.KeyEsc:
			return newCmd(CancelledMsg{})
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}
	case common.CloseViewMsg:
		return newCmd(CancelledMsg{})
	}
	return nil
}

func (m *Model) selectCurrent() tea.Cmd {
	value := m.input.Value()
	return newCmd(SelectedMsg{Value: value})
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	var rows []string
	if m.title != "" {
		rows = append(rows, m.styles.title.Render(m.title))
	}
	m.input.Width = min(box.R.Dx()-2, 40)
	rows = append(rows, m.input.View())

	content := lipgloss.JoinVertical(0, rows...)
	content = m.styles.border.Padding(0, 1).Render(content)
	box = box.Center(lipgloss.Size(content))
	window := dl.Window(box.R, render.ZDialogs)
	window.AddDraw(box.R, content, render.ZDialogs)
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Apply,
		m.keymap.Cancel,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func newCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func ShowWithTitle(title string, prompt string) tea.Cmd {
	return func() tea.Msg {
		return common.ShowInputMsg{Title: title, Prompt: prompt}
	}
}
