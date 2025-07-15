package status

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/idursun/jjui/internal/config"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

var cancel = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "dismiss"))
var accept = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept"))

type Model struct {
	context *context.MainContext
	spinner spinner.Model
	input   textinput.Model
	help    help.Model
	keyMap  help.KeyMap
	command string
	running bool
	output  string
	error   error
	width   int
	mode    string
	editing bool
}

func (m *Model) IsFocused() bool {
	return m.editing
}

const CommandClearDuration = 3 * time.Second

type clearMsg string

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return 1
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m *Model) SetHeight(int) {}
func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	km := config.Current.GetKeyMap()
	switch msg := msg.(type) {
	case clearMsg:
		if m.command == string(msg) {
			m.command = ""
			m.error = nil
			m.output = ""
		}
		return m, nil
	case common.CommandRunningMsg:
		m.command = string(msg)
		m.running = true
		return m, m.spinner.Tick
	case common.CommandCompletedMsg:
		m.running = false
		m.output = msg.Output
		m.error = msg.Err
		if m.error == nil {
			commandToBeCleared := m.command
			return m, tea.Tick(CommandClearDuration, func(time.Time) tea.Msg {
				return clearMsg(commandToBeCleared)
			})
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, km.Cancel) && m.error != nil:
			m.error = nil
			m.output = ""
			m.command = ""
			m.editing = false
			m.mode = ""
		case key.Matches(msg, km.Cancel) && m.editing:
			m.editing = false
			m.input.Reset()
		case key.Matches(msg, accept) && m.editing:
			m.error = nil
			m.output = ""
			m.command = ""
			m.editing = false
			m.mode = ""
			query := m.input.Value()
			m.input.Reset()
			return m, func() tea.Msg {
				return common.QuickSearchMsg(query)
			}
		case key.Matches(msg, km.QuickSearch) && !m.editing:
			m.editing = true
			m.mode = "search"
			return m, m.input.Focus()
		default:
			if m.editing {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		if m.running {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}
}

func (m *Model) View() string {
	commandStatusMark := common.DefaultPalette.Normal.Render(" ")
	if m.running {
		commandStatusMark = common.DefaultPalette.Normal.Render(m.spinner.View())
	} else if m.error != nil {
		commandStatusMark = common.DefaultPalette.Error.Render("✗ ")
	} else if m.command != "" {
		commandStatusMark = common.DefaultPalette.Success.Render("✓ ")
	} else {
		commandStatusMark = m.help.View(m.keyMap)
	}
	ret := common.DefaultPalette.Normal.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	if m.editing {
		commandStatusMark = ""
		ret = m.input.View()
	}
	mode := common.DefaultPalette.StatusMode.Width(10).Render("", m.mode)
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, " ", commandStatusMark, ret)
	if m.error != nil {
		k := cancel.Help().Key
		return lipgloss.JoinVertical(0,
			ret,
			common.DefaultPalette.Error.Render(strings.Trim(m.output, "\n")),
			common.DefaultPalette.Shortcut.Render("press ", k, " to dismiss"))
	}
	return ret
}

func (m *Model) SetHelp(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m *Model) SetMode(mode string) {
	if m.editing {
		m.mode = "search"
	} else {
		m.mode = mode
	}
}

func New(context *context.MainContext) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	h := help.New()
	h.Styles.ShortKey = common.DefaultPalette.Shortcut
	h.Styles.ShortDesc = common.DefaultPalette.Dimmed
	h.Styles.ShortSeparator = common.DefaultPalette.Dimmed
	h.Styles.FullSeparator = common.DefaultPalette.Dimmed

	t := textinput.New()
	t.Width = 50

	return Model{
		context: context,
		spinner: s,
		help:    h,
		command: "",
		running: false,
		output:  "",
		input:   t,
		keyMap:  nil,
	}
}
