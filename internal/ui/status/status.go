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
	"github.com/idursun/jjui/internal/ui/fuzzy_files"
	"github.com/idursun/jjui/internal/ui/fuzzy_input"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
)

var accept = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept"))

type commandStatus int

const (
	none commandStatus = iota
	commandRunning
	commandCompleted
	commandFailed
)

var _ common.Model = (*Model)(nil)

type Model struct {
	context    *context.MainContext
	spinner    spinner.Model
	input      textinput.Model
	keyMap     help.KeyMap
	command    string
	status     commandStatus
	running    bool
	width      int
	mode       string
	editStatus editStatus
	history    map[string][]string
	fuzzy      fuzzy_search.Model
	styles     styles
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
}

// a function that will be used to show
// dynamic help when editing is focused.
type editStatus = func() (help.KeyMap, string)

func emptyEditStatus() (help.KeyMap, string) {
	return nil, ""
}

func (m *Model) IsFocused() bool {
	return m.editStatus != nil
}

func (m *Model) FuzzyView() string {
	if m.fuzzy == nil {
		return ""
	}
	return m.fuzzy.View()
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

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	km := config.Current.GetKeyMap()
	switch msg := msg.(type) {
	case clearMsg:
		if m.command == string(msg) {
			m.command = ""
			m.status = none
		}
		return nil
	case common.CommandRunningMsg:
		m.command = string(msg)
		m.status = commandRunning
		return m.spinner.Tick
	case common.CommandCompletedMsg:
		if msg.Err != nil {
			m.status = commandFailed
		} else {
			m.status = commandCompleted
		}
		commandToBeCleared := m.command
		return tea.Tick(CommandClearDuration, func(time.Time) tea.Msg {
			return clearMsg(commandToBeCleared)
		})
	case common.FileSearchMsg:
		m.mode = "rev file"
		m.input.Prompt = "> "
		m.loadEditingSuggestions()
		m.fuzzy, m.editStatus = fuzzy_files.NewModel(msg)
		return tea.Batch(m.fuzzy.Init(), m.input.Focus())
	case common.ExecProcessCompletedMsg:
		if msg.Err != nil {
			m.mode = "exec " + msg.Msg.Mode.Mode
			m.input.Prompt = msg.Msg.Mode.Prompt
			m.loadEditingSuggestions()
			m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())

			// Avoid to change the current behavior when coming back from exec process
			focusCmd := m.input.Focus()
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(msg.Msg.Line)}
			updateCmd := m.Update(keyMsg)
			return tea.Batch(m.fuzzy.Init(), focusCmd, updateCmd)
		}
		return nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, km.Cancel) && m.IsFocused():
			var cmd tea.Cmd
			if m.fuzzy != nil {
				_, cmd = m.fuzzy.Update(msg)
			}

			m.fuzzy = nil
			m.editStatus = nil
			m.input.Reset()
			return cmd
		case key.Matches(msg, accept) && m.IsFocused():
			editMode := m.mode
			input := m.input.Value()
			prompt := m.input.Prompt
			fuzzy := m.fuzzy
			m.saveEditingSuggestions()

			m.fuzzy = nil
			m.command = ""
			m.editStatus = nil
			m.mode = ""
			m.input.Reset()

			switch {
			case strings.HasSuffix(editMode, "file"):
				_, cmd := fuzzy.Update(msg)
				return cmd
			case strings.HasPrefix(editMode, "exec"):
				return func() tea.Msg { return exec_process.ExecMsgFromLine(prompt, input) }
			}
			return func() tea.Msg { return common.QuickSearchMsg(input) }
		case key.Matches(msg, km.ExecJJ, km.ExecShell) && !m.IsFocused():
			mode := common.ExecJJ
			if key.Matches(msg, km.ExecShell) {
				mode = common.ExecShell
			}
			m.mode = "exec " + mode.Mode
			m.input.Prompt = mode.Prompt
			m.loadEditingSuggestions()

			m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())
			return tea.Batch(m.fuzzy.Init(), m.input.Focus())
		case key.Matches(msg, km.QuickSearch) && !m.IsFocused():
			m.editStatus = emptyEditStatus
			m.mode = "search"
			m.input.Prompt = "> "
			m.loadEditingSuggestions()
			return m.input.Focus()
		default:
			if m.IsFocused() {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.fuzzy != nil {
					cmd = tea.Batch(cmd, fuzzy_search.Search(m.input.Value(), msg))
				}
				return cmd
			}
		}
		return nil
	default:
		var cmd tea.Cmd
		if m.status == commandRunning {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		if m.fuzzy != nil {
			m.fuzzy, cmd = fuzzy_search.Update(m.fuzzy, msg)
		}
		return cmd
	}
}

func (m *Model) saveEditingSuggestions() {
	input := m.input.Value()
	if len(strings.TrimSpace(input)) == 0 {
		return
	}
	h := m.context.Histories.GetHistory(config.HistoryKey(m.mode), true)
	h.Append(input)
}

func (m *Model) loadEditingSuggestions() {
	h := m.context.Histories.GetHistory(config.HistoryKey(m.mode), true)
	history := h.Entries()
	m.input.ShowSuggestions = true
	m.input.SetSuggestions([]string(history))
}

func (m *Model) View() string {
	commandStatusMark := m.styles.text.Render(" ")
	if m.status == commandRunning {
		commandStatusMark = m.styles.text.Render(m.spinner.View())
	} else if m.status == commandFailed {
		commandStatusMark = m.styles.error.Render("✗ ")
	} else if m.status == commandCompleted {
		commandStatusMark = m.styles.success.Render("✓ ")
	} else {
		commandStatusMark = m.helpView(m.keyMap)
		commandStatusMark = lipgloss.PlaceHorizontal(m.width, 0, commandStatusMark, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
	}
	modeWith := max(10, len(m.mode)+2)
	ret := m.styles.text.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	if m.IsFocused() {
		commandStatusMark = ""
		editKeys, editHelp := m.editStatus()
		if editKeys != nil {
			editHelp = lipgloss.JoinHorizontal(0, m.helpView(editKeys), editHelp)
		}
		promptWidth := len(m.input.Prompt) + 2
		m.input.Width = m.width - modeWith - promptWidth - lipgloss.Width(editHelp)
		ret = lipgloss.JoinHorizontal(0, m.input.View(), editHelp)
	}
	mode := m.styles.title.Width(modeWith).Render("", m.mode)
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), commandStatusMark, ret)
	height := lipgloss.Height(ret)
	return lipgloss.Place(m.width, height, 0, 0, ret, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

func (m *Model) SetHelp(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m *Model) SetMode(mode string) {
	if !m.IsFocused() {
		m.mode = mode
	}
}

func (m *Model) helpView(keyMap help.KeyMap) string {
	shortHelp := keyMap.ShortHelp()
	var entries []string
	for _, binding := range shortHelp {
		if !binding.Enabled() {
			continue
		}
		h := binding.Help()
		entries = append(entries, m.styles.shortcut.Render(h.Key)+m.styles.dimmed.PaddingLeft(1).Render(h.Desc))
	}
	help := strings.Join(entries, m.styles.dimmed.Render(" • "))
	return help
}

func New(context *context.MainContext) Model {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		text:     common.DefaultPalette.Get("status text"),
		title:    common.DefaultPalette.Get("status title"),
		success:  common.DefaultPalette.Get("status success"),
		error:    common.DefaultPalette.Get("status error"),
	}
	s := spinner.New()
	s.Spinner = spinner.Dot

	t := textinput.New()
	t.Width = 50
	t.TextStyle = styles.text
	t.CompletionStyle = styles.dimmed
	t.PlaceholderStyle = styles.dimmed

	return Model{
		context: context,
		spinner: s,
		command: "",
		status:  none,
		input:   t,
		keyMap:  nil,
		styles:  styles,
	}
}
