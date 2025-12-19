package input

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
)

type SelectedMsg struct {
	Value string
}

type CancelledMsg struct{}

var (
	_ common.Model     = (*Model)(nil)
	_ common.IViewNode = (*Model)(nil)
	_ help.KeyMap      = (*Model)(nil)
)

type Model struct {
	*common.ViewNode
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
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = prompt
	if ti.Prompt == "" {
		ti.Prompt = "> "
	}

	return &Model{
		ViewNode: common.NewViewNode(0, 0),
		input:    ti,
		title:    title,
		prompt:   prompt,
		keymap:   keymap,
		styles: styles{
			border: common.DefaultPalette.GetBorder("input border", lipgloss.RoundedBorder()),
			text:   common.DefaultPalette.Get("input text"),
			title:  common.DefaultPalette.Get("input title"),
		},
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

func (m *Model) View() string {
	var rows []string
	if m.title != "" {
		rows = append(rows, m.styles.title.Render(m.title))
	}

	rows = append(rows, m.input.View())

	content := lipgloss.JoinVertical(0, rows...)
	content = m.styles.border.Padding(0, 1).Render(content)
	w, h := lipgloss.Size(content)

	if m.Parent != nil {
		pw, ph := m.Parent.Width, m.Parent.Height
		sx := max((pw-w)/2, 0)
		sy := max((ph-h)/2, 0)
		m.SetFrame(cellbuf.Rect(sx, sy, w, h))
	}

	return content
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

func Show() tea.Cmd {
	return ShowWithTitle("", "")
}
