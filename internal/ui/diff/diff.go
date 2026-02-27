package diff

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type Model struct {
	view viewport.Model
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
	case intents.DiffScroll:
		switch msg.Kind {
		case intents.DiffScrollUp:
			m.view.ScrollUp(1)
		case intents.DiffScrollDown:
			m.view.ScrollDown(1)
		case intents.DiffPageUp:
			m.view.ScrollUp(m.view.Height)
		case intents.DiffPageDown:
			m.view.ScrollDown(m.view.Height)
		case intents.DiffHalfPageUp:
			m.view.ScrollUp(m.view.Height / 2)
		case intents.DiffHalfPageDown:
			m.view.ScrollDown(m.view.Height / 2)
		}
		return nil
	case intents.DiffScrollHorizontal:
		switch msg.Kind {
		case intents.DiffScrollLeft:
			m.view.ScrollLeft(1)
		case intents.DiffScrollRight:
			m.view.ScrollRight(1)
		}
		return nil
	case ScrollMsg:
		if msg.Horizontal {
			return nil
		}
		return m.Scroll(msg.Delta)
	}
	return nil
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
		view: view,
	}
}
