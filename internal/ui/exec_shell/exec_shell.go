package exec_shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	context   *context.MainContext
	fuzzyView *fuzzy_search.Model
	keymap    config.KeyMappings[key.Binding]
	input     textinput.Model
}

func (m *Model) Init() tea.Cmd {
	m.input.Focus()
	m.loadEditingSuggestions()
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
			return m, exec_process.ExecLine(m.context, common.ExecShell, input)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.fuzzyView.Search(m.input.Value())
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() string {
	content := lipgloss.JoinVertical(0, fmt.Sprintf("%d", m.fuzzyView.Source.Len()), m.fuzzyView.View(), m.input.View())
	return content
}

func (m *Model) GetId() view.ViewId {
	return "exec shell"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
	if v.Parent != nil {
		m.SetWidth(v.Parent.Width - 20)
	}
}

func (m *Model) saveEditingSuggestions() {
	input := m.input.Value()
	if len(strings.TrimSpace(input)) == 0 {
		return
	}
	h := m.context.Histories.GetHistory("exec_sh", true)
	h.Append(input)
}

func (m *Model) loadEditingSuggestions() tea.Msg {
	h := m.context.Histories.GetHistory("exec_sh", true)
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

func NewShellExecuteModel(ctx *context.MainContext) *Model {
	i := textinput.New()
	i.ShowSuggestions = false
	i.SetSuggestions([]string{})
	i.Prompt = "$ "
	i.Cursor.SetMode(cursor.CursorStatic)

	return &Model{
		context:   ctx,
		keymap:    config.Current.GetKeyMap(),
		fuzzyView: fuzzy_search.NewModel(source{suggestions: make([]string, 0)}, 30),
		input:     i,
	}
}
