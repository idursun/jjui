package choose

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type SelectedMsg struct {
	Value string
}

type CancelledMsg struct{}

type itemClickMsg struct {
	Index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m itemScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

var (
	_ common.ImmediateModel = (*Model)(nil)
	_ help.KeyMap           = (*Model)(nil)
)

type Model struct {
	options             []string
	selected            int
	title               string
	keymap              config.KeyMappings[key.Binding]
	styles              styles
	listRenderer        *render.ListRenderer
	ensureCursorVisible bool
}

type styles struct {
	border   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	selected lipgloss.Style
}

const maxVisibleItems = 20

func New(options []string) *Model {
	return NewWithTitle(options, "")
}

func NewWithTitle(options []string, title string) *Model {
	keymap := config.Current.GetKeyMap()
	return &Model{
		options: options,
		title:   title,
		keymap:  keymap,
		styles: styles{
			border:   common.DefaultPalette.GetBorder("choose border", lipgloss.RoundedBorder()),
			text:     common.DefaultPalette.Get("choose text"),
			title:    common.DefaultPalette.Get("choose title"),
			selected: common.DefaultPalette.Get("choose selected"),
		},
		listRenderer: render.NewListRenderer(itemScrollMsg{}),
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
	case itemScrollMsg:
		if msg.Horizontal {
			return nil
		}
		if m.listRenderer == nil {
			m.listRenderer = render.NewListRenderer(itemScrollMsg{})
		}
		m.listRenderer.StartLine += msg.Delta
		if m.listRenderer.StartLine < 0 {
			m.listRenderer.StartLine = 0
		}
	case itemClickMsg:
		if msg.Index < 0 || msg.Index >= len(m.options) {
			return nil
		}
		m.selected = msg.Index
		return m.selectCurrent()
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
	if next == m.selected {
		return
	}
	m.selected = next
	m.ensureCursorVisible = true
}

func (m *Model) selectCurrent() tea.Cmd {
	if len(m.options) == 0 {
		return newCmd(CancelledMsg{})
	}
	value := m.options[m.selected]
	return newCmd(SelectedMsg{Value: value})
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if m.listRenderer == nil {
		m.listRenderer = render.NewListRenderer(itemScrollMsg{})
	}

	maxContentWidth := max(box.R.Dx()-2, 0)
	maxContentHeight := max(box.R.Dy()-2, 0)
	if maxContentWidth <= 0 || maxContentHeight <= 0 {
		return
	}

	titleHeight := 0
	if m.title != "" {
		titleHeight = 1
	}

	itemWidth := 0
	for _, opt := range m.options {
		itemWidth = max(itemWidth, lipgloss.Width(opt)+2)
	}
	if m.title != "" {
		itemWidth = max(itemWidth, lipgloss.Width(m.title))
	}
	contentWidth := min(itemWidth, maxContentWidth)
	listHeightLimit := maxContentHeight - titleHeight
	if listHeightLimit < 0 {
		listHeightLimit = 0
	}
	listHeight := min(min(len(m.options), listHeightLimit), maxVisibleItems)
	contentHeight := titleHeight + listHeight
	if contentWidth <= 0 || contentHeight <= 0 {
		return
	}

	frame := box.Center(contentWidth+2, contentHeight+2)
	if frame.R.Dx() <= 0 || frame.R.Dy() <= 0 {
		return
	}

	window := dl.Window(frame.R, render.ZMenuContent)
	contentBox := frame.Inset(1)
	if contentBox.R.Dx() <= 0 || contentBox.R.Dy() <= 0 {
		return
	}

	borderBase := lipgloss.NewStyle().Width(contentBox.R.Dx()).Height(contentBox.R.Dy()).Render("")
	window.AddDraw(frame.R, m.styles.border.Render(borderBase), render.ZMenuBorder)

	listBox := contentBox
	if titleHeight > 0 {
		var titleBox layout.Box
		titleBox, listBox = contentBox.CutTop(1)
		window.AddDraw(titleBox.R, m.styles.title.Render(m.title), render.ZMenuContent)
	}

	if listBox.R.Dx() <= 0 || listBox.R.Dy() <= 0 {
		return
	}

	itemCount := len(m.options)
	itemHeight := 1
	m.listRenderer.StartLine = render.ClampStartLine(m.listRenderer.StartLine, listBox.R.Dy(), itemCount, itemHeight)
	m.listRenderer.Render(
		window,
		listBox,
		itemCount,
		m.selected,
		m.ensureCursorVisible,
		func(_ int) int { return itemHeight },
		func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= itemCount || rect.Dx() <= 0 || rect.Dy() <= 0 {
				return
			}
			style := m.styles.text
			if index == m.selected {
				style = m.styles.selected
			}
			line := style.Padding(0, 1).Width(rect.Dx()).Render(m.options[index])
			dl.AddDraw(rect, line, render.ZMenuContent)
		},
		func(index int) tea.Msg { return itemClickMsg{Index: index} },
	)
	m.listRenderer.RegisterScroll(window, listBox)
	m.ensureCursorVisible = false
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
