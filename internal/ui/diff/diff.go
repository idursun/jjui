package diff

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	view   viewport.Model
	keymap config.KeyMappings[key.Binding]
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.DiffView.ScrollUp,
		m.keymap.DiffView.ScrollDown,
		m.keymap.DiffView.HalfPageDown,
		m.keymap.DiffView.HalfPageUp,
		m.keymap.DiffView.PageDown,
		m.keymap.DiffView.PageUp,
		m.keymap.Quit,
		m.keymap.DiffView.Close,
	}
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
		case key.Matches(msg, m.keymap.DiffView.Close):
			return common.Close
		case key.Matches(msg, m.keymap.Quit):
			return tea.Quit
		case key.Matches(msg, m.keymap.ExpandStatus):
			return func() tea.Msg { return intents.ExpandStatusToggle{} }
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
	km := config.Current.GetKeyMap()
	view.KeyMap = viewport.KeyMap{
		Up:           km.DiffView.ScrollUp,
		Down:         km.DiffView.ScrollDown,
		PageUp:       km.DiffView.PageUp,
		PageDown:     km.DiffView.PageDown,
		HalfPageUp:   km.DiffView.HalfPageUp,
		HalfPageDown: km.DiffView.HalfPageDown,
		Left:         km.DiffView.Left,
		Right:        km.DiffView.Right,
	}
	return &Model{
		view:   view,
		keymap: km,
	}
}
