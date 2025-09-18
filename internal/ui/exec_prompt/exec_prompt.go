package exec_prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Model)(nil)
var _ help.KeyMap = (*Model)(nil)

type suggestMode int

const (
	suggestOff suggestMode = iota
	suggestFuzzy
	suggestRegex
)

var tabKey = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "complete"),
)

var ctrlR = key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "suggest: off"))

type Model struct {
	*view.ViewNode
	context     *context.MainContext
	fuzzyView   *fuzzy_search.Model
	keymap      config.KeyMappings[key.Binding]
	input       textinput.Model
	styles      styles
	suggestMode suggestMode
	execMode    common.ExecMode
	historyKey  config.HistoryKey
}

func (m *Model) ShortHelp() []key.Binding {
	shortHelp := []key.Binding{
		m.keymap.Apply,
		m.keymap.Cancel,
		m.keymap.Up,
		m.keymap.Down,
		tabKey,
		ctrlR,
	}

	switch m.suggestMode {
	case suggestOff:
		ctrlR.SetHelp("ctrl+r", "suggest: off")
	case suggestFuzzy:
		ctrlR.SetHelp("ctrl+r", "suggest: fuzzy")
	case suggestRegex:
		ctrlR.SetHelp("ctrl+r", "suggest: regex")
	}
	return shortHelp
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	m.input.Focus()
	m.loadEditingSuggestions(m.historyKey)
	return m.fuzzyView.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			m.ViewManager.UnregisterView(m.Id)
			m.ViewManager.StopEditing()
			return m, nil
		case key.Matches(msg, m.keymap.Apply):
			input := m.input.Value()
			m.ViewManager.UnregisterView(m.Id)
			m.ViewManager.StopEditing()
			m.saveEditingSuggestions()
			return m, ExecLine(m.context, m.execMode, input)
		case key.Matches(msg, m.keymap.Up):
			m.fuzzyView.MoveCursor(1)
		case key.Matches(msg, m.keymap.Down):
			m.fuzzyView.MoveCursor(-1)
		case key.Matches(msg, tabKey):
			m.input.SetValue(m.fuzzyView.SelectedMatch())
			m.input.CursorEnd()
		case key.Matches(msg, ctrlR):
			switch m.suggestMode {
			case suggestOff:
				m.suggestMode = suggestFuzzy
				m.fuzzyView.Search(m.input.Value())
			case suggestFuzzy:
				m.suggestMode = suggestRegex
				m.fuzzyView.SearchRegex(m.input.Value())
			case suggestRegex:
				m.suggestMode = suggestOff
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		switch m.suggestMode {
		case suggestOff:
			// no suggestions
		case suggestFuzzy:
			m.fuzzyView.Search(m.input.Value())
		case suggestRegex:
			m.fuzzyView.SearchRegex(m.input.Value())
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() string {
	content := lipgloss.PlaceHorizontal(m.Width-2, lipgloss.Left, m.input.View())
	content = m.styles.borderStyle.Render(content)

	if m.suggestMode != suggestOff {
		title := fmt.Sprintf(
			"  %d of %d elements in history ",
			m.fuzzyView.Matches.Len(),
			m.fuzzyView.Source.Len(),
		)
		title = m.styles.selectedMatch.Render(title)
		completionView := lipgloss.JoinVertical(0, title, m.fuzzyView.View())
		return lipgloss.JoinVertical(lipgloss.Left, completionView, content)
	}
	return content
}

func (m *Model) GetId() view.ViewId {
	switch m.execMode {
	case common.ExecShell:
		return "exec shell"
	case common.ExecJJ:
		return "exec jj"
	default:
		return "exec"
	}
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
	if v.Parent != nil {
		m.SetWidth(v.Parent.Width - 3)
	}
}

func (m *Model) saveEditingSuggestions() {
	input := m.input.Value()
	if len(strings.TrimSpace(input)) == 0 {
		return
	}
	h := m.context.Histories.GetHistory(m.historyKey, true)
	h.Append(input)
}

func (m *Model) loadEditingSuggestions(key config.HistoryKey) tea.Msg {
	h := m.context.Histories.GetHistory(key, true)
	history := h.Entries()
	m.fuzzyView.Source = source{suggestions: history}
	m.input.SetSuggestions(history)
	return nil
}

type source struct {
	suggestions []string
}

func (s source) String(i int) string {
	return s.suggestions[i]
}

func (s source) Len() int {
	return len(s.suggestions)
}

type styles struct {
	Dimmed        lipgloss.Style
	DimmedMatch   lipgloss.Style
	Selected      lipgloss.Style
	selectedMatch lipgloss.Style
	borderStyle   lipgloss.Style
	textStyle     lipgloss.Style
}

func NewExecPrompt(ctx *context.MainContext, mode common.ExecMode) *Model {
	s := styles{
		Dimmed:        common.DefaultPalette.Get("exec dimmed"),
		DimmedMatch:   common.DefaultPalette.Get("exec shortcut"),
		Selected:      common.DefaultPalette.Get("exec selected"),
		selectedMatch: common.DefaultPalette.Get("exec title"),
		borderStyle:   common.DefaultPalette.GetBorder("exec border", lipgloss.NormalBorder()),
		textStyle:     common.DefaultPalette.Get("exec text"),
	}

	i := textinput.New()
	i.ShowSuggestions = false
	i.SetSuggestions([]string{})
	i.Prompt = "$ "
	i.Cursor.SetMode(cursor.CursorStatic)
	i.TextStyle = s.textStyle

	return &Model{
		context:     ctx,
		keymap:      config.Current.GetKeyMap(),
		fuzzyView:   fuzzy_search.NewModel(source{suggestions: make([]string, 0)}, 30),
		input:       i,
		styles:      s,
		suggestMode: suggestOff,
		execMode:    mode,
		historyKey:  config.HistoryKey(fmt.Sprintf("exec_%s", mode)),
	}
}
