package revset

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/autocompletion"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type EditRevSetMsg struct{}

var _ common.ImmediateModel = (*Model)(nil)

type revsetMsg struct {
	msg tea.Msg
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
	listRenderer       *render.ListRenderer
	completionItems    []CompletionItem
	selectedIndex      int
	userInput          string // tracks what the user actually typed (separate from preview)
}

func (m *Model) Scopes() []dispatch.Scope {
	if !m.Editing {
		return nil
	}
	return []dispatch.Scope{
		{
			Name:    actions.ScopeRevset,
			Leak:    dispatch.LeakNone,
			Handler: m,
		},
	}
}

func (m *Model) IsFocused() bool {
	return m.Editing
}

func (m *Model) GetValue() string {
	return m.autoComplete.Value()
}

func New(context *appContext.MainContext) *Model {
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
		cmd, _ := m.HandleIntent(msg)
		return cmd
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
	case common.UpdateRevSetMsg:
		if m.Editing {
			m.Editing = false
		}
		m.autoComplete.SetValue(string(msg))
		m.userInput = string(msg)
		return nil
	case EditRevSetMsg:
		cmd, _ := m.HandleIntent(intents.Edit{})
		return cmd
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

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Set:
		m.Editing = false
		m.autoComplete.Blur()
		value := intent.Value
		if strings.TrimSpace(value) == "" {
			value = m.context.DefaultRevset
		}
		return tea.Batch(common.Close, common.UpdateRevSet(value)), true
	case intents.Reset:
		m.Editing = false
		m.autoComplete.Blur()
		return tea.Batch(common.Close, common.UpdateRevSet(m.context.DefaultRevset)), true
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
		return m.autoComplete.Init(), true
	case intents.Cancel:
		m.Editing = false
		m.autoComplete.Blur()
		return nil, true
	case intents.Apply:
		value := intent.Value
		if value == "" {
			value = m.autoComplete.Value()
		}
		if strings.TrimSpace(value) == "" {
			value = m.context.DefaultRevset
		}

		// Validate the revset before applying
		_, err := m.context.RunCommandImmediate(jj.RevsetValidate(value))
		if err != nil {
			return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err}), true
		}

		m.Editing = false
		m.autoComplete.Blur()
		return tea.Batch(common.Close, common.UpdateRevSet(value)), true
	case intents.CompletionCycle:
		if len(m.completionItems) == 0 {
			return nil, true
		}
		if intent.Reverse {
			if m.selectedIndex <= 0 {
				m.selectedIndex = len(m.completionItems) - 1
			} else {
				m.selectedIndex--
			}
		} else {
			m.selectedIndex++
			if m.selectedIndex >= len(m.completionItems) {
				m.selectedIndex = 0
			}
		}
		m.updatePreview()
		return nil, true
	case intents.CompletionMove:
		if len(m.completionItems) == 0 {
			return nil, true
		}
		if intent.Delta < 0 {
			if m.selectedIndex < 0 {
				m.selectedIndex = len(m.completionItems) - 1
			} else {
				m.selectedIndex--
				if m.selectedIndex < 0 {
					m.selectedIndex = len(m.completionItems) - 1
				}
			}
			m.updatePreview()
			return nil, true
		}
		if intent.Delta > 0 {
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
		return nil, true
	}
	return nil, false
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {

	titleStyle := common.DefaultPalette.Get("revset title")
	textStyle := common.DefaultPalette.Get("revset text")
	completionDimmed := common.DefaultPalette.Get("revset completion dimmed")

	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZFuzzyInput)
	tb.Styled("revset: ", titleStyle)
	if m.Editing {
		// Only render the text input part, not the completions from autoComplete.View()
		tb.Write(m.autoComplete.TextInput.View())
	} else {
		tb.Styled(m.context.CurrentRevset, textStyle)
	}
	tb.Done()

	if !m.Editing {
		return
	}

	// Check if we have completions to show or signature help
	items := m.completionItems
	signatureHelp := m.autoComplete.SignatureHelp

	if len(items) == 0 && signatureHelp == "" {
		// Show "No suggestions" when there's input but no matches
		if m.autoComplete.Value() != "" {
			noSuggestionsRect := layout.Rect(box.R.Min.X, box.R.Max.Y, box.R.Dx(), 1)
			noSuggestionsText := completionDimmed.Render("No suggestions")
			dl.AddDraw(noSuggestionsRect, noSuggestionsText, render.ZRevsetOverlay)
		}
		return
	}

	// If no items but we have signature help, show it
	if len(items) == 0 && signatureHelp != "" {
		sigRect := layout.Rect(box.R.Min.X, box.R.Max.Y, box.R.Dx(), 1)
		sigText := completionDimmed.Render(signatureHelp)
		dl.AddDraw(sigRect, sigText, render.ZRevsetOverlay)
		return
	}

	// Render the completion list as a multi-line overlay
	overlayHeight := min(len(items), maxCompletionItems)
	overlayWidth := box.R.Dx()
	outerBox := layout.NewBox(layout.Rect(box.R.Min.X, box.R.Max.Y, overlayWidth, overlayHeight))
	// Fill the background to prevent underlying content from showing through
	dl.AddFill(outerBox.R, ' ', common.DefaultPalette.Get("revset completion"), render.ZRevsetOverlay-1)
	completionText := common.DefaultPalette.Get("revset completion text")
	completionMatched := common.DefaultPalette.Get("revset completion matched")

	m.listRenderer.Render(
		dl,
		outerBox,
		len(items),
		m.selectedIndex,
		true, // ensureCursorVisible
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			if index < 0 || index >= len(items) {
				return
			}
			isSelected := index == m.selectedIndex

			item := items[index]

			ts := completionText
			ms := completionMatched
			ds := completionDimmed

			if isSelected {
				ts = common.DefaultPalette.Get("revset completion selected text")
				ms = common.DefaultPalette.Get("revset completion selected matched")
				ds = common.DefaultPalette.Get("revset completion selected dimmed")
				dl.AddFill(rect, ' ', common.DefaultPalette.Get("revset completion selected"), render.ZRevsetOverlay-1)
			}

			tb := dl.Text(rect.Min.X, rect.Min.Y, render.ZRevsetOverlay)
			pillStyle := ds.Width(pillWidth).Align(lipgloss.Right)
			tb.Styled(pillLabel(item.Kind), pillStyle)
			tb.Styled(" ", ts)
			tb.Styled(item.MatchedPart, ms)
			tb.Styled(item.RestPart, ts)
			tb.Styled(" ", ts)

			if item.SignatureHelp != "" && item.Kind != KindHistory {
				sigDisplay := m.formatSignature(item)
				tb.Styled(sigDisplay, ds)
			}
			tb.Done()
		},
		func(index int, _ tea.Mouse) tea.Msg { return completionClickMsg{index: index} },
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
	if _, after, ok := strings.Cut(sig, "):"); ok {
		return strings.TrimSpace(after)
	}
	if _, after, ok := strings.Cut(sig, ":"); ok {
		return strings.TrimSpace(after)
	}
	return sig
}

func (m *Model) applyCompletion(input string, item CompletionItem) string {
	if item.Kind == KindHistory {
		return item.Name
	}

	paren := ""
	if item.Kind == KindFunction {
		if !item.HasParameters {
			paren = "()"
		} else {
			paren = "("
		}
	}
	completionText := item.Name + paren

	lastTokenIndex, _ := m.completionProvider.GetLastToken(input)
	if lastTokenIndex > 0 {
		return input[:lastTokenIndex] + completionText
	}
	return completionText
}
