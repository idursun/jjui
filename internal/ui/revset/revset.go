package revset

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/autocompletion"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type EditRevSetMsg struct {
	Clear bool
}

var _ common.ImmediateModel = (*Model)(nil)

type revsetMsg struct {
	msg tea.Msg
}

// Allow a message to be targeted to this component.
func RevsetCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return revsetMsg{msg: msg}
	}
}

type Model struct {
	Editing         bool
	autoComplete    *autocompletion.AutoCompletionInput
	History         []string
	historyIndex    int
	currentInput    string
	historyActive   bool
	MaxHistoryItems int
	context         *appContext.MainContext
	styles          styles
}

type styles struct {
	promptStyle lipgloss.Style
	textStyle   lipgloss.Style
}

func (m *Model) IsFocused() bool {
	return m.Editing
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
		key.NewBinding(key.WithKeys("up/down"), key.WithHelp("↑/↓", "history")),
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func New(context *appContext.MainContext) *Model {
	styles := styles{
		promptStyle: common.DefaultPalette.Get("revset title"),
		textStyle:   common.DefaultPalette.Get("revset text"),
	}

	revsetAliases := context.JJConfig.RevsetAliases
	completionProvider := NewCompletionProvider(revsetAliases)
	autoComplete := autocompletion.New(completionProvider, autocompletion.WithStylePrefix("revset"))

	autoComplete.SetValue(context.DefaultRevset)
	autoComplete.Focus()

	return &Model{
		context:         context,
		Editing:         false,
		autoComplete:    autoComplete,
		History:         []string{},
		historyIndex:    -1,
		MaxHistoryItems: 50,
		styles:          styles,
	}
}

func (m *Model) Init() tea.Cmd {
	// Ensure CurrentRevset is initialized
	if m.context.CurrentRevset == "" {
		m.context.CurrentRevset = m.context.DefaultRevset
	}
	return nil
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

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(revsetMsg); ok {
		msg = k.msg
	}

	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case tea.KeyMsg:
		if !m.Editing {
			return nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m.handleIntent(intents.Cancel{})
		case tea.KeyEnter:
			return m.handleIntent(intents.Apply{Value: m.autoComplete.Value()})
		case tea.KeyUp:
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
			} else {
				m.autoComplete.SetValue(m.context.CurrentRevset)
			}

			return nil
		case tea.KeyDown:
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
				return nil
			}
		}
	case common.UpdateRevSetMsg:
		if m.Editing {
			m.Editing = false
		}
	case EditRevSetMsg:
		return m.handleIntent(intents.Edit{Clear: msg.Clear})
	}

	return m.autoComplete.Update(msg)
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.Set:
		m.Editing = false
		m.autoComplete.Blur()
		value := intent.Value
		if strings.TrimSpace(value) == "" {
			value = m.context.DefaultRevset
		}
		return tea.Batch(common.Close, common.UpdateRevSet(value))
	case intents.Reset:
		m.Editing = false
		m.autoComplete.Blur()
		return tea.Batch(common.Close, common.UpdateRevSet(m.context.DefaultRevset))
	case intents.Edit:
		m.Editing = true
		m.autoComplete.Focus()
		if intent.Clear {
			m.autoComplete.SetValue("")
		}
		m.historyActive = false
		m.historyIndex = -1
		return m.autoComplete.Init()
	case intents.Cancel:
		m.Editing = false
		m.autoComplete.Blur()
		return nil
	case intents.Apply:
		m.Editing = false
		m.autoComplete.Blur()
		value := intent.Value
		if value == "" {
			value = m.autoComplete.Value()
		}
		if strings.TrimSpace(value) == "" {
			value = m.context.DefaultRevset
		}
		return tea.Batch(common.Close, common.UpdateRevSet(value))
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	var w strings.Builder
	w.WriteString(m.styles.promptStyle.PaddingRight(1).Render("revset:"))
	if m.Editing {
		w.WriteString(m.autoComplete.View())
	} else {
		w.WriteString(m.styles.textStyle.Render(m.context.CurrentRevset))
	}
	content := w.String()
	parts := strings.SplitN(content, "\n", 2)
	line := parts[0]
	dl.AddDraw(box.R, line, render.ZFuzzyInput)

	if !m.Editing || len(parts) < 2 {
		return
	}

	overlay := parts[1]
	overlayHeight := 1 + strings.Count(overlay, "\n")
	if overlayHeight <= 0 {
		return
	}

	overlayRect := cellbuf.Rect(box.R.Min.X, box.R.Max.Y, box.R.Dx(), overlayHeight)
	dl.AddDraw(overlayRect, overlay, render.ZRevsetOverlay)
}
