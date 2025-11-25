package diff

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
)

var _ common.Model = (*Model)(nil)

type Model struct {
	view   viewport.Model
	keymap config.KeyMappings[key.Binding]
}

func (m *Model) ShortHelp() []key.Binding {
	vkm := m.view.KeyMap
	return []key.Binding{
		vkm.Up, vkm.Down, vkm.HalfPageDown, vkm.HalfPageUp, vkm.PageDown, vkm.PageUp,
		m.keymap.Cancel}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetHeight(h int) {
	m.view.Height = h
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return common.Close
		}
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return cmd
}

func (m *Model) View() string {
	return m.view.View()
}

func New(output string, width int, height int) *Model {
	view := viewport.New(width, height)
	content := strings.ReplaceAll(output, "\r", "")
	if content == "" {
		content = "(empty)"
	}
	view.SetContent(content)
	return &Model{
		view:   view,
		keymap: config.Current.GetKeyMap(),
	}
}
