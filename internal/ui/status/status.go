package status

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/idursun/jjui/internal/config"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_process"
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
	styles  styles
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
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
			editMode := m.mode
			input := m.input.Value()
			prompt := m.input.Prompt

			m.error = nil
			m.output = ""
			m.command = ""
			m.editing = false
			m.mode = ""
			m.input.Reset()
			return m, func() tea.Msg {
				if strings.HasPrefix(editMode, "exec") {
					return exec_process.ExecMsgFromLine(prompt, input)
				}
				return common.QuickSearchMsg(input)
			}
		case key.Matches(msg, km.ExecJJ, km.ExecShell) && !m.editing:
			mode := common.ExecJJ
			if key.Matches(msg, km.ExecShell) {
				mode = common.ExecShell
			}
			m.editing = true
			m.mode = "exec " + mode.Mode
			m.input.Prompt = mode.Prompt
			return m, m.input.Focus()
		case key.Matches(msg, km.QuickSearch) && !m.editing:
			m.editing = true
			m.mode = "search"
			m.input.Prompt = "> "
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
		commandStatusMark = m.styles.error.Render("✗ ")
	} else if m.command != "" {
		commandStatusMark = m.styles.success.Render("✓ ")
	} else {
		commandStatusMark = m.help.View(m.keyMap)
	}
	ret := common.DefaultPalette.Normal.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	if m.editing {
		commandStatusMark = ""
		ret = m.input.View()
	}
	mode := m.styles.title.Width(10).Render("", m.mode)
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, " ", commandStatusMark, ret)
	if m.error != nil {
		k := cancel.Help().Key
		return lipgloss.JoinVertical(0,
			ret,
			strings.Trim(m.output, "\n"),
			m.styles.shortcut.Render("press ", k, " to dismiss"))
	}
	return ret
}

func (m *Model) SetHelp(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m *Model) SetMode(mode string) {
	if !m.editing {
		m.mode = mode
	}
}

func New(context *context.MainContext) Model {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		title:    common.DefaultPalette.Get("status title"),
		success:  common.DefaultPalette.Get("status success"),
		error:    common.DefaultPalette.Get("status error"),
	}
	s := spinner.New()
	s.Spinner = spinner.Dot

	h := help.New()
	h.Styles.ShortKey = styles.shortcut
	h.Styles.ShortDesc = styles.dimmed
	h.Styles.ShortSeparator = styles.dimmed
	h.Styles.FullSeparator = styles.dimmed

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
		styles:  styles,
	}
}
