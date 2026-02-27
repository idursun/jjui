package input

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type SelectedMsg struct {
	Value string
}

type CancelledMsg struct{}

var (
	_ common.ImmediateModel = (*Model)(nil)
	_ common.Focusable      = (*Model)(nil)
)

type Model struct {
	input  textinput.Model
	title  string
	prompt string
	styles styles
}

func (m *Model) IsFocused() bool {
	return true
}

func (m *Model) StackedActionOwner() string {
	return actions.OwnerInput
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
	styles := styles{
		border: common.DefaultPalette.GetBorder("input border", lipgloss.RoundedBorder()),
		text:   common.DefaultPalette.Get("input text"),
		title:  common.DefaultPalette.Get("input title"),
	}
	ti := textinput.New()
	ti.SetWidth(40)
	ti.Focus()
	ti.Prompt = prompt
	is := ti.Styles()
	is.Focused.Prompt = styles.text
	is.Blurred.Prompt = styles.text
	ti.SetStyles(is)
	if ti.Prompt == "" {
		ti.Prompt = "> "
	}

	return &Model{
		input:  ti,
		title:  title,
		prompt: prompt,
		styles: styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.input.Focus()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		switch msg.(type) {
		case intents.Apply:
			return m.selectCurrent()
		case intents.Cancel:
			return newCmd(CancelledMsg{})
		}
		return nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case common.CloseViewMsg:
		return newCmd(CancelledMsg{})
	}
	return nil
}

func (m *Model) selectCurrent() tea.Cmd {
	value := m.input.Value()
	return newCmd(SelectedMsg{Value: value})
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	var rows []string
	if m.title != "" {
		rows = append(rows, m.styles.title.Render(m.title))
	}
	m.input.SetWidth(min(box.R.Dx()-2, 40))
	rows = append(rows, m.input.View())

	content := lipgloss.JoinVertical(0, rows...)
	content = m.styles.border.Padding(0, 1).Render(content)
	box = box.Center(lipgloss.Size(content))
	window := dl.Window(box.R, render.ZDialogs)
	window.AddDraw(box.R, content, render.ZDialogs)
}

func newCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func ShowWithTitle(title string, prompt string) tea.Cmd {
	return func() tea.Msg {
		return common.ShowInputMsg{Title: title, Prompt: prompt}
	}
}
