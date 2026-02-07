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

const (
	maxCompletionItems = 10
	pillWidth          = 10
)

type completionScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m completionScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

type completionClickMsg struct {
	index int
}

type Model struct {
	Editing            bool
	autoComplete       *autocompletion.AutoCompletionInput
	completionProvider *CompletionProvider
	History            []string
	MaxHistoryItems    int
	context            *appContext.MainContext
	styles             styles
	listRenderer       *render.ListRenderer
	completionItems    []CompletionItem
	selectedIndex      int
	userInput          string // tracks what the user actually typed (separate from preview)
}

type styles struct {
	title lipgloss.Style
	text  lipgloss.Style

	// Completion overlay styles
	completionText       lipgloss.Style
	completionMatched    lipgloss.Style
	completionSelected   lipgloss.Style
	completionDimmed     lipgloss.Style
	completionBackground lipgloss.Style
}

func (m *Model) IsFocused() bool {
	return m.Editing
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle completions")),
		key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "cycle back")),
		key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "move selection")),
		key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "move selection")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func New(context *appContext.MainContext) *Model {
	palette := common.DefaultPalette
	styles := styles{
		title:                palette.Get("revset title"),
		text:                 palette.Get("revset text"),
		completionText:       palette.Get("revset completion text"),
		completionMatched:    palette.Get("revset completion matched"),
		completionSelected:   palette.Get("revset completion selected"),
		completionDimmed:     palette.Get("revset completion dimmed"),
		completionBackground: palette.Get("revset completion"),
	}

	revsetAliases := context.JJConfig.RevsetAliases
	completionProvider := NewCompletionProvider(revsetAliases)
	autoComplete := autocompletion.New(
		completionProvider,
		autocompletion.WithStylePrefix("revset"),
		autocompletion.WithCompletionsDisabled(),
	)

	autoComplete.SetValue(context.DefaultRevset)
	autoComplete.Focus()

	return &Model{
		context:            context,
		Editing:            false,
		autoComplete:       autoComplete,
		completionProvider: completionProvider,
		History:            []string{},
		MaxHistoryItems:    50,
		styles:             styles,
		listRenderer:       render.NewListRenderer(completionScrollMsg{}),
		selectedIndex:      -1, // no selection initially
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

	m.selectedIndex = -1
}

func (m *Model) SetHistory(history []string) {
	m.History = history
	m.selectedIndex = -1
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(revsetMsg); ok {
		msg = k.msg
	}

	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case completionScrollMsg:
		if msg.Horizontal {
			return nil
		}
		m.listRenderer.StartLine += msg.Delta
		if m.listRenderer.StartLine < 0 {
			m.listRenderer.StartLine = 0
		}
		return nil
	case completionClickMsg:
		if msg.index >= 0 && msg.index < len(m.completionItems) {
			m.selectedIndex = msg.index
			item := m.completionItems[msg.index]
			m.selectCompletionItem(item)
		}
		return nil
	case tea.KeyMsg:
		if !m.Editing {
			return nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m.handleIntent(intents.Cancel{})
		case tea.KeyEnter:
			return m.handleIntent(intents.Apply{Value: m.autoComplete.Value()})
		case tea.KeyTab:
			// Fish-style tab completion: cycle through completions
			if len(m.completionItems) > 0 {
				m.selectedIndex++
				if m.selectedIndex >= len(m.completionItems) {
					m.selectedIndex = 0
				}
				m.updatePreview()
			}
			return nil
		case tea.KeyShiftTab:
			// Shift+Tab cycles backwards
			if len(m.completionItems) > 0 {
				if m.selectedIndex <= 0 {
					m.selectedIndex = len(m.completionItems) - 1
				} else {
					m.selectedIndex--
				}
				m.updatePreview()
			}
			return nil
		case tea.KeyUp:
			if len(m.completionItems) > 0 {
				if m.selectedIndex < 0 {
					m.selectedIndex = len(m.completionItems) - 1
				} else {
					m.selectedIndex--
					if m.selectedIndex < 0 {
						m.selectedIndex = len(m.completionItems) - 1
					}
				}
				m.updatePreview()
			}
			return nil
		case tea.KeyDown:
			if len(m.completionItems) > 0 {
				if m.selectedIndex < 0 {
					m.selectedIndex = 0
				} else {
					m.selectedIndex++
					if m.selectedIndex >= len(m.completionItems) {
						m.selectedIndex = 0
					}
				}
				m.updatePreview()
			}
			return nil
		}
	case common.UpdateRevSetMsg:
		if m.Editing {
			m.Editing = false
		}
	case EditRevSetMsg:
		return m.handleIntent(intents.Edit{Clear: msg.Clear})
	}

	prevValue := m.autoComplete.Value()
	cmd := m.autoComplete.Update(msg)

	// If the value changed due to user typing (not tab cycling),
	// update userInput, reset selection, and re-filter completions
	newValue := m.autoComplete.Value()
	if newValue != prevValue {
		m.userInput = newValue
		m.selectedIndex = -1 // reset to no selection
		m.updateCompletionItems()
	}

	return cmd
}

func (m *Model) selectCompletionItem(item CompletionItem) {
	newValue := m.applyCompletion(m.userInput, item)

	m.autoComplete.SetValue(newValue)
	m.autoComplete.CursorEnd()
	// Commit the completion: update userInput and re-filter
	m.userInput = newValue
	m.selectedIndex = -1
	m.updateCompletionItems()
}

func (m *Model) updateCompletionItems() {
	// Use userInput for filtering, not the preview value in the text input
	m.completionItems = m.completionProvider.GetCompletionItems(m.userInput, m.History)
	// Reset scroll position when items change
	m.listRenderer.StartLine = 0
}

// updatePreview updates the text input to show a preview of the selected item
// without changing the userInput (what the user actually typed)
func (m *Model) updatePreview() {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.completionItems) {
		return
	}

	item := m.completionItems[m.selectedIndex]
	previewValue := m.applyCompletion(m.userInput, item)

	m.autoComplete.SetValue(previewValue)
	m.autoComplete.CursorEnd()
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
		m.completionProvider.Load(m.context.RunCommandImmediate)
		if intent.Clear {
			m.autoComplete.SetValue("")
			m.userInput = ""
		} else {
			m.userInput = m.autoComplete.Value()
		}
		m.selectedIndex = -1 // no selection initially
		m.updateCompletionItems()
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
	// Render the prompt and text input line
	var w strings.Builder
	w.WriteString(m.styles.title.PaddingRight(1).Render("revset:"))
	if m.Editing {
		// Only render the text input part, not the completions from autoComplete.View()
		w.WriteString(m.autoComplete.TextInput.View())
	} else {
		w.WriteString(m.styles.text.Render(m.context.CurrentRevset))
	}
	line := w.String()
	dl.AddDraw(box.R, line, render.ZFuzzyInput)

	if !m.Editing {
		return
	}

	// Check if we have completions to show or signature help
	items := m.completionItems
	signatureHelp := m.autoComplete.SignatureHelp

	if len(items) == 0 && signatureHelp == "" {
		// Show "No suggestions" when there's input but no matches
		if m.autoComplete.Value() != "" {
			noSuggestionsRect := cellbuf.Rect(box.R.Min.X, box.R.Max.Y, box.R.Dx(), 1)
			noSuggestionsText := m.styles.completionDimmed.Render("No suggestions")
			dl.AddDraw(noSuggestionsRect, noSuggestionsText, render.ZRevsetOverlay)
		}
		return
	}

	// If no items but we have signature help, show it
	if len(items) == 0 && signatureHelp != "" {
		sigRect := cellbuf.Rect(box.R.Min.X, box.R.Max.Y, box.R.Dx(), 1)
		sigText := m.styles.completionDimmed.Render(signatureHelp)
		dl.AddDraw(sigRect, sigText, render.ZRevsetOverlay)
		return
	}

	// Render the completion list as a multi-line overlay
	overlayHeight := min(len(items), maxCompletionItems)
	overlayWidth := box.R.Dx()
	outerBox := layout.NewBox(cellbuf.Rect(box.R.Min.X, box.R.Max.Y, overlayWidth, overlayHeight))
	// Fill the background to prevent underlying content from showing through
	dl.AddFill(outerBox.R, ' ', m.styles.completionBackground, render.ZRevsetOverlay-1)

	m.listRenderer.Render(
		dl,
		outerBox,
		len(items),
		m.selectedIndex,
		true, // ensureCursorVisible
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= len(items) {
				return
			}
			isSelected := index == m.selectedIndex

			item := items[index]
			tb := dl.Text(rect.Min.X, rect.Min.Y, render.ZRevsetOverlay)
			pillStyle := m.styles.completionDimmed.Width(pillWidth).Align(lipgloss.Right)
			tb.Styled(pillLabel(item.Kind), pillStyle)
			tb.Styled(" ", m.styles.completionText)
			tb.Styled(item.MatchedPart, m.styles.completionMatched)
			tb.Styled(item.RestPart, m.styles.completionText)
			tb.Styled(" ", m.styles.completionText)

			if item.SignatureHelp != "" && item.Kind != KindHistory {
				sigDisplay := m.formatSignature(item)
				tb.Styled(sigDisplay, m.styles.completionDimmed)
			}

			if isSelected {
				dl.AddPaint(rect, m.styles.completionSelected, render.ZRevsetOverlay+1)
			}
			tb.Done()
		},
		func(index int) tea.Msg { return completionClickMsg{index: index} },
	)
	m.listRenderer.RegisterScroll(dl, outerBox)
}

func pillLabel(kind CompletionKind) string {
	switch kind {
	case KindFunction:
		return "function"
	case KindAlias:
		return "alias"
	case KindHistory:
		return "history"
	case KindBookmark:
		return "bookmark"
	case KindTag:
		return "tag"
	default:
		return ""
	}
}

func (m *Model) formatSignature(item CompletionItem) string {
	sig := item.SignatureHelp
	// The signature format is "name(args): description"
	// We want to show just the description or "(args): desc" if different from name
	if colonIdx := strings.Index(sig, "):"); colonIdx != -1 {
		// Return description part after "): "
		return strings.TrimSpace(sig[colonIdx+2:])
	}
	if colonIdx := strings.Index(sig, ":"); colonIdx != -1 {
		// Return description part after ": "
		return strings.TrimSpace(sig[colonIdx+1:])
	}
	return sig
}

func (m *Model) applyCompletion(input string, item CompletionItem) string {
	if item.Kind == KindHistory {
		return item.Name
	}

	lastTokenIndex, _ := m.completionProvider.GetLastToken(input)
	if lastTokenIndex > 0 {
		return input[:lastTokenIndex] + item.Name
	}
	return item.Name
}

// getHistoryIndices returns the indices of history items in completionItems
