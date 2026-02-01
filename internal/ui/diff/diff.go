package diff

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	view   viewport.Model
	keymap config.KeyMappings[key.Binding]
}

func (m *Model) ShortHelp() []key.Binding {
	vkm := m.view.KeyMap
	return []key.Binding{
		vkm.Up,
		vkm.Down,
		vkm.HalfPageDown,
		vkm.HalfPageUp,
		vkm.PageDown,
		vkm.PageUp,
		m.keymap.Quit,
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

func (m *Model) Scroll(delta int) tea.Cmd {
	if delta > 0 {
		m.view.ScrollDown(delta)
	} else if delta < 0 {
		m.view.ScrollUp(-delta)
	}
	return nil
}

type ScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (s ScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	s.Delta = delta
	s.Horizontal = horizontal
	return s
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return common.Close
		case key.Matches(msg, m.keymap.Quit):
			return tea.Quit
		}
	case ScrollMsg:
		if msg.Horizontal {
			return nil
		}
		return m.Scroll(msg.Delta)
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return cmd
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	m.view.Height = box.R.Dy()
	m.view.Width = box.R.Dx()
	dl.AddDraw(box.R, m.view.View(), 0)
	dl.AddInteraction(box.R, ScrollMsg{}, render.InteractionScroll, 0)
}

func New(output string) *Model {
	view := viewport.New(0, 0)
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
