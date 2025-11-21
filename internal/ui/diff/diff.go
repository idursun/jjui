package diff

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
)

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
	m.view.SetHeight(h)
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		}
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

func (m *Model) View() *lipgloss.Layer {
	return lipgloss.NewLayer(m.view.View())
}

func New(output string, width int, height int) *Model {
	view := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
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
