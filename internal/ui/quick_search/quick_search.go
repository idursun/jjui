package quick_search

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
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

const label = "search"

type Model struct {
	context *context.MainContext
	input   textinput.Model
}

func New(ctx *context.MainContext) *Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.SetVirtualCursor(false)
	ti.Focus()

	m := &Model{
		context: ctx,
		input:   ti,
	}
	m.input.ShowSuggestions = true
	m.input.SetSuggestions(m.history().Entries())
	return m
}

func (m *Model) history() *config.History {
	return m.context.Histories.GetHistory(config.HistoryKey(label), true)
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
			Name:    actions.ScopeQuickSearchInput,
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
		input := m.input.Value()
		if strings.TrimSpace(input) != "" {
			m.history().Append(input)
		}
		return tea.Batch(common.CloseApplied, func() tea.Msg { return common.QuickSearchMsg(input) }), true
	}
	return nil, false
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case tea.KeyMsg, tea.PasteMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}
	return nil
}

// ViewRect draws the input on the last row above the status line.
func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area, _ := box.CutBottom(1)
	if area.R.Dx() <= 0 || area.R.Dy() < 1 {
		return
	}
	_, inputBox := area.CutBottom(1)

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

	labelWidth := len(label) + 2
	m.input.SetWidth(max(0, inputBox.R.Dx()-labelWidth-1-render.StringWidth(m.input.Prompt)))
	dl.AddFill(inputBox.R, ' ', textStyle, render.ZFuzzyOverlay-1)
	dl.AddDraw(inputBox.R, titleStyle.Width(labelWidth).Render(label)+textStyle.Render(" ")+m.input.View(), render.ZFuzzyOverlay)
	dl.SetCursorInRect(m.input.Cursor(), inputBox.R, labelWidth+1, 0)
}
