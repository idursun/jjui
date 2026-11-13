package exec_prompt

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/fuzzy_input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

var (
	_ common.StackedModel = (*Model)(nil)
	_ common.ScopeHandler = (*Model)(nil)
	_ common.Editable     = (*Model)(nil)
)

type Model struct {
	context *context.MainContext
	mode    common.ExecMode
	input   textinput.Model
	fuzzy   *fuzzy_input.Model
}

// New opens the exec prompt with line prefilled, e.g. to fix a failed command.
func New(ctx *context.MainContext, mode common.ExecMode, line string) *Model {
	ti := textinput.New()
	ti.Prompt = mode.Prompt
	ti.SetVirtualCursor(false)
	ti.Focus()

	m := &Model{
		context: ctx,
		mode:    mode,
		input:   ti,
	}
	m.fuzzy = fuzzy_input.New(&m.input, m.history().Entries())
	if line != "" {
		m.input.SetValue(line)
		m.input.CursorEnd()
		m.fuzzy.Search(line)
	}
	return m
}

func (m *Model) label() string {
	return "exec " + m.mode.Mode
}

func (m *Model) history() *config.History {
	return m.context.Histories.GetHistory(config.HistoryKey(m.label()), true)
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) IsEditing() bool {
	return true
}

func (m *Model) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeStatusInput,
			Leak:    common.LeakNone,
			Handler: m,
		},
	}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		return m.apply(), true
	}
	return m.fuzzy.Update(intent), true
}

func (m *Model) apply() tea.Cmd {
	if selected := m.fuzzy.Selected(); selected != "" {
		m.input.SetValue(selected)
	}
	input := m.input.Value()
	if strings.TrimSpace(input) != "" {
		m.history().Append(input)
	}
	prompt := m.mode.Prompt
	// close first so a quickly failing command can reopen the prompt
	return tea.Sequence(common.CloseApplied, func() tea.Msg {
		return exec_process.ExecMsgFromLine(prompt, input)
	})
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case fuzzy_input.ItemClickedMsg:
		m.fuzzy.Update(msg)
		return m.apply()
	case tea.KeyMsg, tea.PasteMsg:
		previous := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != previous {
			m.fuzzy.Search(m.input.Value())
		}
		return cmd
	default:
		return m.fuzzy.Update(msg)
	}
}

// ViewRect draws the input on the last row above the status line with the
// suggestions above it.
func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area, _ := box.CutBottom(1)
	if area.R.Dx() <= 0 || area.R.Dy() < 1 {
		return
	}
	suggestionsBox, inputBox := area.CutBottom(1)

	dimmedStyle := theme.DefaultPalette.Get("status", "", "dimmed", false)
	textStyle := theme.DefaultPalette.Get("status", "", "text", false)
	titleStyle := theme.DefaultPalette.Get("status", "", "title", false).PaddingLeft(1).PaddingRight(1)
	inputStyles := m.input.Styles()
	inputStyles.Focused.Text = textStyle
	inputStyles.Focused.Suggestion = dimmedStyle
	inputStyles.Focused.Placeholder = dimmedStyle
	inputStyles.Blurred.Text = textStyle
	inputStyles.Blurred.Suggestion = dimmedStyle
	inputStyles.Blurred.Placeholder = dimmedStyle
	m.input.SetStyles(inputStyles)

	label := m.label()
	labelWidth := len(label) + 2
	m.input.SetWidth(max(0, inputBox.R.Dx()-labelWidth-1-render.StringWidth(m.input.Prompt)))
	dl.AddFill(inputBox.R, ' ', textStyle, render.ZFuzzyOverlay-1)
	dl.AddDraw(inputBox.R, titleStyle.Width(labelWidth).Render(label)+textStyle.Render(" ")+m.input.View(), render.ZFuzzyOverlay)
	dl.SetCursorInRect(m.input.Cursor(), inputBox.R, labelWidth+1, 0)

	m.fuzzy.ViewRect(dl, suggestionsBox)
}
