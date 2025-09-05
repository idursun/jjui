package status

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type commandStatus int

const (
	none commandStatus = iota
	commandRunning
	commandCompleted
	commandFailed
)

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	context *context.MainContext
	spinner spinner.Model
	input   textinput.Model
	command string
	status  commandStatus
	running bool
	width   int
	mode    string
	history map[string][]string
	styles  styles
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
	v.NeedsRefresh = true
}

func (m *Model) GetId() view.ViewId {
	return view.StatusViewId
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
}

const CommandClearDuration = 3 * time.Second

type clearMsg string

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearMsg:
		if m.command == string(msg) {
			m.command = ""
			m.status = none
		}
		return m, nil
	case common.CommandRunningMsg:
		m.command = string(msg)
		m.status = commandRunning
		return m, m.spinner.Tick
	case common.CommandCompletedMsg:
		if msg.Err != nil {
			m.status = commandFailed
		} else {
			m.status = commandCompleted
		}
		commandToBeCleared := m.command
		return m, tea.Tick(CommandClearDuration, func(time.Time) tea.Msg {
			return clearMsg(commandToBeCleared)
		})
	//case common.FileSearchMsg:
	//	m.mode = "rev file"
	//	m.input.Prompt = "> "
	//	m.loadEditingSuggestions()
	//	m.fuzzy, m.editStatus = fuzzy_files.NewModel(msg)
	//	return m, tea.Batch(m.fuzzy.Init(), m.input.Focus())
	case tea.KeyMsg:
		switch {
		//case key.Matches(msg, km.Cancel) && m.IsFocused():
		//	var cmd tea.Cmd
		//	if m.fuzzy != nil {
		//		_, cmd = m.fuzzy.Update(msg)
		//	}
		//
		//	m.fuzzy = nil
		//	m.editStatus = nil
		//	m.input.Reset()
		//	return m, cmd
		//case key.Matches(msg, accept) && m.IsFocused():
		//	editMode := m.mode
		//	input := m.input.Value()
		//	prompt := m.input.Prompt
		//	fuzzy := m.fuzzy
		//	m.saveEditingSuggestions()
		//
		//	m.fuzzy = nil
		//	m.command = ""
		//	m.editStatus = nil
		//	m.mode = ""
		//	m.input.Reset()
		//
		//	switch {
		//	case strings.HasSuffix(editMode, "file"):
		//		_, cmd := fuzzy.Update(msg)
		//		return m, cmd
		//	case strings.HasPrefix(editMode, "exec"):
		//		return m, func() tea.Msg { return exec_process.ExecMsgFromLine(prompt, input) }
		//	}
		//	return m, func() tea.Msg { return common.QuickSearchMsg(input) }
		//case key.Matches(msg, km.ExecJJ, km.ExecShell) && !m.IsFocused():
		//	mode := common.ExecJJ
		//	if key.Matches(msg, km.ExecShell) {
		//		mode = common.ExecShell
		//	}
		//	m.mode = "exec " + mode.Mode
		//	m.input.Prompt = mode.Prompt
		//	m.loadEditingSuggestions()
		//
		//	m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())
		//	return m, tea.Batch(m.fuzzy.Init(), m.input.Focus())
		//default:
		//	if m.IsFocused() {
		//		var cmd tea.Cmd
		//		m.input, cmd = m.input.Update(msg)
		//		if m.fuzzy != nil {
		//			cmd = tea.Batch(cmd, fuzzy_search.Search(m.input.Value(), msg))
		//		}
		//		return m, cmd
		//	}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		if m.status == commandRunning {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		//if m.fuzzy != nil {
		//	m.fuzzy, cmd = fuzzy_search.Update(m.fuzzy, msg)
		//}
		return m, cmd
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
	m.input.SetSuggestions(history)
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

		// Get the ID of the currently focused view
		if focusedView := m.ViewManager.GetFocusedView(); focusedView != nil {
			if keymap, ok := focusedView.Model.(help.KeyMap); ok {
				commandStatusMark = m.helpView(keymap)
			}
		}
		commandStatusMark = lipgloss.PlaceHorizontal(m.width, 0, commandStatusMark, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
	}

	// Get the ID of the currently focused view
	if focusedView := m.ViewManager.GetFocusedView(); focusedView != nil {
		m.mode = string(focusedView.Id)
	}

	modeWidth := max(10, len(m.mode)+2)
	ret := m.styles.text.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	//if m.IsFocused() {
	//	commandStatusMark = ""
	//	editKeys, editHelp := m.editStatus()
	//	if editKeys != nil {
	//		editHelp = lipgloss.JoinHorizontal(0, m.helpView(editKeys), editHelp)
	//	}
	//	promptWidth := len(m.input.Prompt) + 2
	//	m.input.Width = m.width - modeWidth - promptWidth - lipgloss.Width(editHelp)
	//	ret = lipgloss.JoinHorizontal(0, m.input.View(), editHelp)
	//}

	mode := m.styles.title.Width(modeWidth).Render("", m.mode)
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), commandStatusMark, ret)
	height := lipgloss.Height(ret)
	return lipgloss.Place(m.width, height, 0, 0, ret, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

func (m *Model) helpView(keyMap help.KeyMap) string {
	if keyMap == nil {
		return ""
	}
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

func New(ctx *context.MainContext) view.IViewModel {
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

	m := Model{
		context: ctx,
		spinner: s,
		command: "",
		status:  none,
		input:   t,
		styles:  styles,
	}
	return &m
}
