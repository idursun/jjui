package confirmation

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type CloseMsg struct{}

type SelectOptionMsg struct {
	Index int
}

type MoveSelectionMsg struct {
	Delta int
}

type ApplySelectionMsg struct {
	Alt bool
}

type CancelMsg struct{}

type option struct {
	label      string
	cmd        tea.Cmd
	keyBinding key.Binding
	altCmd     tea.Cmd
}

type Styles struct {
	Border   lipgloss.Style
	Selected lipgloss.Style
	Dimmed   lipgloss.Style
	Text     lipgloss.Style
}

type Model struct {
	options     []option
	selected    int
	Styles      Styles
	messages    []string
	stylePrefix string
	zIndex      int
}

// Option is a function that configures a Model
type Option func(*Model)

// WithStylePrefix returns an Option that sets the style prefix for palette lookups
func WithStylePrefix(prefix string) Option {
	return func(m *Model) {
		m.stylePrefix = prefix
	}
}

// WithOption adds an option to the confirmation dialog
func WithOption(label string, cmd tea.Cmd, keyBinding key.Binding) Option {
	return func(m *Model) {
		m.options = append(m.options, option{label, cmd, keyBinding, cmd})
	}
}

func WithAltOption(label string, cmd tea.Cmd, altCmd tea.Cmd, keyBinding key.Binding) Option {
	return func(m *Model) {
		m.options = append(m.options, option{label, cmd, keyBinding, altCmd})
	}
}

// WithZIndex sets the z-index for rendering
func WithZIndex(z int) Option {
	return func(m *Model) {
		m.zIndex = z
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SelectOptionMsg:
		if msg.Index >= 0 && msg.Index < len(m.options) {
			m.selected = msg.Index
			selectedOption := m.options[m.selected]
			return selectedOption.cmd
		}
		return nil
	case MoveSelectionMsg:
		if len(m.options) == 0 {
			return nil
		}
		next := m.selected + msg.Delta
		if next < 0 {
			next = 0
		}
		if next >= len(m.options) {
			next = len(m.options) - 1
		}
		m.selected = next
		return nil
	case ApplySelectionMsg:
		if len(m.options) == 0 {
			return nil
		}
		selectedOption := m.options[m.selected]
		if msg.Alt {
			return selectedOption.altCmd
		}
		return selectedOption.cmd
	case CancelMsg:
		return m.runOptionForKey("esc", false)
	case intents.Intent:
		switch msg := msg.(type) {
		case intents.Apply:
			return m.Update(ApplySelectionMsg{Alt: msg.Force})
		case intents.Cancel:
			return m.Update(CancelMsg{})
		case intents.OptionSelect:
			if msg.Delta < 0 {
				return m.Update(MoveSelectionMsg{Delta: -1})
			}
			if msg.Delta > 0 {
				return m.Update(MoveSelectionMsg{Delta: 1})
			}
		}
		return nil
	case tea.KeyMsg:
		for _, option := range m.options {
			if key.Matches(msg, option.keyBinding) {
				if msg.Alt {
					return option.altCmd
				}
				return option.cmd
			}
		}
	}
	return nil
}

func (m *Model) runOptionForKey(bindingKey string, alt bool) tea.Cmd {
	for _, option := range m.options {
		for _, keyName := range option.keyBinding.Keys() {
			if keyName == bindingKey {
				if alt {
					return option.altCmd
				}
				return option.cmd
			}
		}
	}
	return Close
}

func (m *Model) View() string {
	w := strings.Builder{}
	for i, message := range m.messages {
		w.WriteString(m.Styles.Text.PaddingLeft(1).Render(message))
		if i < len(m.messages)-1 {
			w.WriteString(m.Styles.Text.Render("\n"))
		}
	}
	for i, option := range m.options {
		if i == m.selected {
			w.WriteString(m.Styles.Selected.Render(option.label))
		} else {
			w.WriteString(m.Styles.Dimmed.Render(option.label))
		}
	}
	content := w.String()
	width, height := lipgloss.Size(content)
	content = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content, lipgloss.WithWhitespaceBackground(m.Styles.Text.GetBackground()))
	return m.Styles.Border.Render(content)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	z := m.zIndex

	measure := dl.Text(0, 0, 0)
	m.buildContent(measure)
	contentWidth, contentHeight := measure.Measure()
	if contentWidth <= 0 {
		contentWidth = 1
	}
	if contentHeight <= 0 {
		contentHeight = 1
	}

	base := lipgloss.NewStyle().Width(contentWidth).Height(contentHeight).Render("")
	bordered := m.Styles.Border.Render(base)
	bw, bh := lipgloss.Size(bordered)

	sx := box.R.Min.X
	sy := box.R.Min.Y

	frame := cellbuf.Rect(sx, sy, bw, bh)
	window := dl.Window(frame, z)
	window.AddDraw(frame, bordered, z)

	mt, mr, mb, ml := m.Styles.Border.GetMargin()
	pt, pr, pb, pl := m.Styles.Border.GetPadding()
	bl := m.Styles.Border.GetBorderLeftSize()
	br := m.Styles.Border.GetBorderRightSize()
	bt := m.Styles.Border.GetBorderTopSize()
	bb := m.Styles.Border.GetBorderBottomSize()

	contentRect := cellbuf.Rect(
		frame.Min.X+ml+bl+pl,
		frame.Min.Y+mt+bt+pt,
		max(frame.Dx()-ml-mr-bl-br-pl-pr, 0),
		max(frame.Dy()-mt-mb-bt-bb-pt-pb, 0),
	)
	if contentRect.Dx() <= 0 || contentRect.Dy() <= 0 {
		return
	}

	background := lipgloss.NewStyle().Background(m.Styles.Text.GetBackground())
	window.AddFill(contentRect, ' ', background, z+1)

	tb := window.Text(contentRect.Min.X, contentRect.Min.Y, z+2)
	m.buildContent(tb)
	tb.Done()
}

// getStyleKey prefixes the key with the style prefix if one is set
func (m *Model) getStyleKey(key string) string {
	if m.stylePrefix == "" {
		return key
	}
	return m.stylePrefix + " " + key
}

func New(messages []string, opts ...Option) *Model {
	m := Model{
		messages: messages,
		options:  []option{},
		selected: 0,
	}

	// Apply options if provided
	for _, opt := range opts {
		opt(&m)
	}

	// Set styles after options are applied so stylePrefix is considered
	m.Styles = Styles{
		Border:   common.DefaultPalette.GetBorder(m.getStyleKey("confirmation border"), lipgloss.RoundedBorder()),
		Text:     common.DefaultPalette.Get(m.getStyleKey("confirmation text")).PaddingRight(1),
		Selected: common.DefaultPalette.Get(m.getStyleKey("confirmation selected")).PaddingLeft(2).PaddingRight(2),
		Dimmed:   common.DefaultPalette.Get(m.getStyleKey("confirmation dimmed")).PaddingLeft(2).PaddingRight(2),
	}

	return &m
}

func Close() tea.Msg {
	return CloseMsg{}
}

func (m *Model) buildContent(tb *render.TextBuilder) {
	for i, message := range m.messages {
		tb.Styled(message, m.Styles.Text.PaddingLeft(1))
		if i < len(m.messages)-1 {
			tb.NewLine()
		}
	}

	for idx, option := range m.options {
		style := m.Styles.Dimmed
		if idx == m.selected {
			style = m.Styles.Selected
		}
		tb.Clickable(option.label, style, SelectOptionMsg{Index: idx})
	}
}
