package diff

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	view   viewport.Model
	keymap config.KeyMappings[key.Binding]
}

func (m *Model) GetActionMap() map[string]actions.Action {
	return config.Current.GetBindings("diff")
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "diff.up":
			m.view.ScrollUp(1)
		case "diff.down":
			m.view.ScrollDown(1)
		case "diff.halfpageup":
			m.view.HalfPageDown()
		case "diff.halfpagedown":
			m.view.HalfPageDown()
		case "diff.pageup":
			m.view.PageUp()
		case "diff.pagedown":
			m.view.PageDown()
		}
	case common.ShowDiffMsg:
		content := strings.ReplaceAll(string(msg), "\r", "")
		if content == "" {
			content = "(empty)"
		}
		m.view.SetContent(content)
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	return m.view.View()
}

func New(output string, width int, height int) tea.Model {
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
