package revset

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
)

type EditRevSetMsg struct {
	Clear bool
}

type Model struct {
	Editing       bool
	Value         string
	defaultRevSet string
	autoComplete  *common.AutoCompletionInput
	help          help.Model
	keymap        keymap

	History         []string
	historyIndex    int
	currentInput    string
	historyActive   bool
	MaxHistoryItems int
}

func (m *Model) IsFocused() bool {
	return m.Editing
}

type keymap struct{}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func New(defaultRevSet string) *Model {
	completionProvider := NewCompletionProvider()
	autoComplete := common.NewAutoCompletionInput(completionProvider)
	autoComplete.SetPrompt("revset: ")
	autoComplete.PromptStyle = common.DefaultPalette.ChangeId
	autoComplete.TextInput.Cursor.Style = cursorStyle
	autoComplete.SelectedStyle = common.DefaultPalette.Modified.Background(common.BrightBlack).Inline(true)
	autoComplete.UnselectedStyle = common.DefaultPalette.Dimmed
	autoComplete.SetValue(defaultRevSet)
	autoComplete.Focus()

	h := help.New()
	h.Styles.ShortKey = common.DefaultPalette.Shortcut
	h.Styles.ShortDesc = common.DefaultPalette.Dimmed

	return &Model{
		Editing:         false,
		Value:           defaultRevSet,
		defaultRevSet:   defaultRevSet,
		help:            h,
		keymap:          keymap{},
		autoComplete:    autoComplete,
		History:         []string{},
		historyIndex:    -1,
		MaxHistoryItems: 50,
	}
}

func (m *Model) Init() tea.Cmd {
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

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Editing {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Editing = false
			m.autoComplete.Blur()
			return m, nil
		case tea.KeyEnter:
			m.Editing = false
			m.autoComplete.Blur()
			m.Value = m.autoComplete.Value()
			m.AddToHistory(m.Value)
			if m.Value == "" {
				m.Value = m.defaultRevSet
			}
			return m, tea.Batch(common.Close, UpdateRevSet(m.Value))
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
				return m, nil
			}
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
				return m, nil
			}
		}
	case UpdateRevSetMsg:
		m.Editing = false
		m.Value = string(msg)
	case EditRevSetMsg:
		m.Editing = true
		m.autoComplete.Focus()
		if msg.Clear {
			m.autoComplete.SetValue("")
		}
		m.historyActive = false
		m.historyIndex = -1
		return m, m.autoComplete.Init()
	}

	var cmd tea.Cmd
	m.autoComplete, cmd = m.autoComplete.Update(msg)
	return m, cmd
}

var (
	promptStyle = common.DefaultPalette.ChangeId.SetString("revset:")
	cursorStyle = common.DefaultPalette.EmptyPlaceholder
)

func (m *Model) View() string {
	if m.Editing {
		return m.autoComplete.View()
	}

	revset := "(default)"
	if m.Value != "" {
		revset = m.Value
	}

	return promptStyle.Render(cursorStyle.Render(revset))
}

type UpdateRevSetMsg string

func UpdateRevSet(revset string) tea.Cmd {
	return func() tea.Msg {
		return UpdateRevSetMsg(revset)
	}
}
