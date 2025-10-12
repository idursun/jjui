package exec_process

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/fuzzy_input"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ tea.Model = (*Model)(nil)
var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	width   int
	height  int
	context *context.MainContext
	input   *fuzzy_input.Model
	styles  styles
	scope   view.Scope
}

func (m *Model) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings(string(m.scope))
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
}

func NewModel(ctx *context.MainContext, scope view.Scope) *Model {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		text:     common.DefaultPalette.Get("status text"),
		title:    common.DefaultPalette.Get("status title"),
		success:  common.DefaultPalette.Get("status success"),
		error:    common.DefaultPalette.Get("status error"),
	}
	t := textinput.New()
	t.Width = 50
	t.TextStyle = styles.text
	t.CompletionStyle = styles.dimmed
	t.PlaceholderStyle = styles.dimmed
	fi := fuzzy_input.NewModel(t, []string{"hello", "world", "word"})

	return &Model{
		context: ctx,
		scope:   scope,
		input:   fi,
		styles:  styles,
	}
}

func (m *Model) Init() tea.Cmd {
	m.loadEditingSuggestions()
	return m.input.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		//case "exec_jj.cycle_suggest_mode":
		//	m.input.CycleSuggestMode()
		//	return m, nil
		case "exec_jj.accept", "exec_sh.accept":
			input := m.input.Value()
			prompt := common.ExecJJ.Prompt
			if msg.Action.Id == "exec_sh.accept" {
				prompt = common.ExecShell.Prompt
			}
			return m, func() tea.Msg { return ExecMsgFromLine(prompt, input) }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	mode := string(m.scope)
	modeWidth := len(mode) + 2
	mode = m.styles.title.Width(modeWidth).Render("", mode)

	ret := lipgloss.JoinHorizontal(lipgloss.Left, mode, m.input.View())
	completionView := m.input.CompletionView()
	if completionView != "" {
		ret = lipgloss.JoinVertical(lipgloss.Left, completionView, ret)
	}
	height := lipgloss.Height(ret)
	return lipgloss.Place(m.width, height, 0, 0, ret, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

func (m *Model) loadEditingSuggestions() {
	mode := "exec_jj"
	if m.scope == view.ScopeExecSh {
		mode = "exec_sh"
	}
	h := m.context.Histories.GetHistory(config.HistoryKey(mode), true)
	history := h.Entries()
	m.input.SetSuggestions(history)
}
