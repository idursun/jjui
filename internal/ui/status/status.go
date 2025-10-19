package status

import (
	"slices"
	"strings"
	"time"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/view"

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

var _ tea.Model = (*Model)(nil)
var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	context          *context.MainContext
	spinner          spinner.Model
	command          string
	status           commandStatus
	showExtendedHelp bool
	running          bool
	width            int
	mode             string
	styles           styles
}

func (m *Model) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings("status")
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "status.help":
			m.showExtendedHelp = !m.showExtendedHelp
			return m, nil
		}
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
	default:
		var cmd tea.Cmd
		if m.status == commandRunning {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() string {
	commandStatusMark := m.styles.text.Render(" ")
	commandHelp := ""
	if m.status == commandRunning {
		commandStatusMark = m.styles.text.Render(m.spinner.View())
	} else if m.status == commandFailed {
		commandStatusMark = m.styles.error.Render("✗ ")
	} else if m.status == commandCompleted {
		commandStatusMark = m.styles.success.Render("✓ ")
	} else if m.showExtendedHelp {
		if actionMap := config.Current.GetBindings(m.mode); len(actionMap.Bindings) > 0 {
			commandHelp = m.actionMapView(actionMap)
		}
	}
	if v, ok := m.context.Router.Views[m.context.Router.Scope]; ok {
		boldStyle := m.styles.shortcut.Bold(true)
		if commandPreview, ok := v.(view.ICommandBuilder); ok {
			args := commandPreview.GetCommand()
			if len(args) > 0 {
				commandStatusMark = ""
				for _, arg := range args {
					if strings.HasPrefix(arg, "-") {
						commandStatusMark += boldStyle.Render(arg) + " "
					} else {
						commandStatusMark += m.styles.text.Render(arg) + " "
					}
				}
			}
		}
	}
	commandStatusMark = lipgloss.PlaceHorizontal(m.width, 0, commandStatusMark, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
	modeWith := max(10, len(m.mode)+2)
	ret := m.styles.text.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	mode := m.styles.title.Width(modeWith).Render("", m.mode)
	ret = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), commandStatusMark, ret)
	if commandHelp != "" {
		ret = lipgloss.JoinVertical(lipgloss.Left, ret, commandHelp)
	}
	height := lipgloss.Height(ret)
	return lipgloss.Place(m.width, height, 0, 0, ret, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

func (m *Model) SetHelp(status view.IStatus) {
	m.mode = status.Name()
}

func (m *Model) SetMode(mode string) {
	m.mode = mode
}

func (m *Model) actionMapView(actionMap actions.ActionMap) string {
	type entry struct {
		shortcut string
		desc     string
	}
	var entries []entry
	for _, binding := range actionMap.Bindings {
		k := config.JoinKeys(binding.On)
		entries = append(entries, entry{
			shortcut: k,
			desc:     binding.Do.GetDesc(),
		})
	}
	rendered := make([]string, len(entries))
	maxShortcutLen := 0
	maxDescLen := 0
	for _, e := range entries {
		if len(e.shortcut) > maxShortcutLen {
			maxShortcutLen = len(e.shortcut)
		}
		if len(e.desc) > maxDescLen {
			maxDescLen = len(e.desc)
		}
	}
	totalWidth := m.width
	shortcutStyle := m.styles.shortcut
	descStyle := m.styles.dimmed.PaddingLeft(1)
	for i, e := range entries {
		totalWidth -= maxShortcutLen
		totalWidth -= len(e.desc)
		rendered[i] = lipgloss.JoinHorizontal(0, shortcutStyle.Render(e.shortcut), descStyle.Render(e.desc+" "))
	}
	if totalWidth > 0 {
		return lipgloss.JoinHorizontal(0, rendered...)
	}

	var columns []string
	perColumn := ((maxShortcutLen + maxDescLen) * len(entries)) / m.width
	for chunk := range slices.Chunk(rendered, perColumn) {
		columns = append(columns, lipgloss.JoinVertical(0, chunk...))
	}
	return lipgloss.JoinHorizontal(0, columns...)
}

func New(context *context.MainContext) *Model {
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

	return &Model{
		context: context,
		spinner: s,
		command: "",
		status:  none,
		styles:  styles,
	}
}
