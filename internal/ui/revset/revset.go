package revset

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/autocompletion"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Model)(nil)
var _ help.KeyMap = (*Model)(nil)
var _ view.Editable = (*Model)(nil)

type Model struct {
	*view.ViewNode
	context         *appContext.MainContext
	autoComplete    *autocompletion.AutoCompletionInput
	keymap          config.KeyMappings[key.Binding]
	History         []string
	historyIndex    int
	currentInput    string
	historyActive   bool
	MaxHistoryItems int
	styles          styles
}

func (m *Model) OnEdit() {
	m.autoComplete.SetValue("")
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
}

func (m *Model) GetId() view.ViewId {
	return view.RevsetViewId
}

type styles struct {
	promptStyle lipgloss.Style
	textStyle   lipgloss.Style
}

func (m *Model) Init() tea.Cmd {
	return nil
}

var upKey = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "previous"))
var downKey = key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next"))
var applyKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply"))
var cancelKey = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel"))
var tabKey = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle"))

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		upKey,
		downKey,
		tabKey,
		applyKey,
		cancelKey,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) AddToHistory(input string) {
	if input == "" {
		return
	}

	for i, item := range m.History {
		if item == input {
			m.History = append(m.History[:i], m.History[i+1:]...)
			break
		}
	}

	m.History = append([]string{input}, m.History...)

	if len(m.History) > m.MaxHistoryItems && m.MaxHistoryItems > 0 {
		m.History = m.History[:m.MaxHistoryItems]
	}

	m.historyIndex = -1
	m.historyActive = false
}

func (m *Model) SetHistory(history []string) {
	m.History = history
	m.historyIndex = -1
	m.historyActive = false
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.ViewManager.IsThisEditing(m.Id) {
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			m.ViewManager.StopEditing()
			m.ViewManager.RestorePreviousFocus()
			return m, nil
		case key.Matches(msg, m.keymap.Apply):
			value := m.autoComplete.Value()
			m.AddToHistory(value)
			m.context.CurrentRevset = value
			m.ViewManager.StopEditing()
			m.ViewManager.RestorePreviousFocus()
			return m, common.Refresh
		case key.Matches(msg, upKey):
			if len(m.History) > 0 {
				if !m.historyActive {
					m.currentInput = m.autoComplete.Value()
					m.historyActive = true
				}

				if m.historyIndex < len(m.History)-1 {
					m.historyIndex++
					m.autoComplete.SetValue(m.History[m.historyIndex])
					m.autoComplete.CursorEnd()
				}
				return m, nil
			}
		case key.Matches(msg, downKey):
			if m.historyActive {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.autoComplete.SetValue(m.History[m.historyIndex])
				} else {
					m.historyIndex = -1
					m.historyActive = false
					m.autoComplete.SetValue(m.currentInput)
				}
				m.autoComplete.CursorEnd()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.autoComplete, cmd = m.autoComplete.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	var w strings.Builder
	w.WriteString(m.styles.promptStyle.PaddingRight(1).Render("revset:"))
	if m.ViewManager.IsThisEditing(m.Id) {
		w.WriteString(m.autoComplete.View())
	} else {
		revset := m.context.DefaultRevset
		if m.context.CurrentRevset != "" {
			revset = m.context.CurrentRevset
		}
		w.WriteString(m.styles.textStyle.Render(revset))
	}
	content := w.String()
	width, height := lipgloss.Size(content)
	m.SetHeight(height)
	return lipgloss.Place(width, height, 0, 0, w.String(), lipgloss.WithWhitespaceBackground(m.styles.textStyle.GetBackground()))
}

func New(ctx *appContext.MainContext) *Model {
	styles := styles{
		promptStyle: common.DefaultPalette.Get("revset title"),
		textStyle:   common.DefaultPalette.Get("revset text"),
	}

	revsetAliases := ctx.JJConfig.RevsetAliases
	completionProvider := NewCompletionProvider(revsetAliases)
	autoComplete := autocompletion.New(completionProvider, autocompletion.WithStylePrefix("revset"))

	autoComplete.SetValue(ctx.DefaultRevset)
	autoComplete.Focus()

	m := &Model{
		context:         ctx,
		keymap:          config.Current.GetKeyMap(),
		autoComplete:    autoComplete,
		History:         []string{},
		historyIndex:    -1,
		MaxHistoryItems: 50,
		styles:          styles,
	}
	return m
}
