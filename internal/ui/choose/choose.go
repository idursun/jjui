package choose

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	options  []string
	selected int
	title    string
	keymap   config.KeyMappings[key.Binding]
	styles   styles
}

type styles struct {
	border lipgloss.Style
	text   lipgloss.Style
	title  lipgloss.Style
}

func New(options []string) *Model {
	return NewWithTitle(options, "")
}

func NewWithTitle(options []string, title string) *Model {
	keymap := config.Current.GetKeyMap()
	return &Model{
		ViewNode: common.NewViewNode(0, 0),
		options:  options,
		title:    title,
		keymap:   keymap,
		styles: styles{
			border: common.DefaultPalette.GetBorder("choose border", lipgloss.RoundedBorder()),
			text:   common.DefaultPalette.Get("choose text"),
			title:  common.DefaultPalette.Get("choose title"),
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			m.move(-1)
		case key.Matches(msg, m.keymap.Down):
			m.move(1)
		case key.Matches(msg, m.keymap.Apply):
			return m.selectCurrent()
		case key.Matches(msg, m.keymap.Cancel):
			return newCmd(CancelledMsg{})
		}
	case common.CloseViewMsg:
		return newCmd(CancelledMsg{})
	}
	return nil
}

func (m *Model) move(delta int) {
	if len(m.options) == 0 {
		return
	}
	next := m.selected + delta
	n := len(m.options)
	if next < 0 {
		next = 0
	}
	if next >= n {
		next = n - 1
	}
	m.selected = next
}

func (m *Model) selectCurrent() tea.Cmd {
	if len(m.options) == 0 {
		return newCmd(CancelledMsg{})
	}
	value := m.options[m.selected]
	return newCmd(SelectedMsg{Value: value})
}

func (m *Model) View() string {
	var rows []string
	if m.title != "" {
		rows = append(rows, m.styles.title.Render(m.title))
	}
	for i, opt := range m.options {
		style := m.styles.text
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}
		rows = append(rows, style.Render(prefix+opt))
	}
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
		m.keymap.Up,
		m.keymap.Down,
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

func ShowWithTitle(options []string, title string) tea.Cmd {
	return func() tea.Msg {
		return common.ShowChooseMsg{Options: options, Title: title}
	}
}

func Show(options []string) tea.Cmd {
	return ShowWithTitle(options, "")
}
