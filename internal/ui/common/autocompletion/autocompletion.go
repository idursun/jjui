package autocompletion

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
)

var _ common.Model = (*AutoCompletionInput)(nil)

type AutoCompletionInput struct {
	TextInput          textinput.Model
	CompletionProvider CompletionProvider
	Suggestions        []string
	SignatureHelp      string
	previousValue      string
	currentCompletions []Completion

	tabCompletionActive    bool
	lastCompletedValue     string
	currentSuggestionIndex int
	firstTabPressed        bool
	Styles                 *AutoCompleteStyles
}

type AutoCompleteStyles struct {
	Selected lipgloss.Style
	Matched  lipgloss.Style
	Text     lipgloss.Style
	Dimmed   lipgloss.Style
}

type Completion struct {
	FullText    string
	MatchedPart string
	RestPart    string
}

type Option func(m *AutoCompletionInput)

func WithStylePrefix(prefix string) Option {
	return func(m *AutoCompletionInput) {
		if prefix != "" {
			prefix += " "
		}
		styles := AutoCompleteStyles{
			Selected: common.DefaultPalette.Get(prefix + "selected"),
			Matched:  common.DefaultPalette.Get(prefix + "matched"),
			Text:     common.DefaultPalette.Get(prefix + "text"),
			Dimmed:   common.DefaultPalette.Get(prefix + "dimmed"),
		}
		m.Styles = &styles
	}
}

func New(provider CompletionProvider, options ...Option) *AutoCompletionInput {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = ""
	ti.ShowSuggestions = true

	m := &AutoCompletionInput{
		TextInput:          ti,
		CompletionProvider: provider,
	}

	for _, option := range options {
		option(m)
	}

	if m.Styles == nil {
		// applies default style
		WithStylePrefix("")(m)
	}
	m.TextInput.TextStyle = m.Styles.Text
	m.TextInput.PlaceholderStyle = m.Styles.Dimmed
	return m
}

func (ac *AutoCompletionInput) Init() tea.Cmd {
	return textinput.Blink
}

func (ac *AutoCompletionInput) SetValue(value string) {
	ac.TextInput.SetValue(value)
	ac.updateCompletions()
}

func (ac *AutoCompletionInput) Value() string {
	return ac.TextInput.Value()
}

func (ac *AutoCompletionInput) Focus() {
	ac.TextInput.Focus()
}

func (ac *AutoCompletionInput) Blur() {
	ac.TextInput.Blur()
}

func (ac *AutoCompletionInput) CursorEnd() {
	ac.TextInput.CursorEnd()
}

func (ac *AutoCompletionInput) Update(msg tea.Msg) tea.Cmd {
	prevValue := ac.TextInput.Value()

	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyTab:
			ac.tabCompletionActive = true
			ac.cycleCompletion(1)
			return cmd
		case tea.KeyShiftTab:
			ac.tabCompletionActive = true
			ac.cycleCompletion(-1)
			return cmd
		default:
			ac.tabCompletionActive = false
		}
	}

	ac.TextInput, cmd = ac.TextInput.Update(msg)

	if ac.TextInput.Value() != prevValue && !ac.tabCompletionActive {
		ac.updateCompletions()
	}

	return cmd
}

func (ac *AutoCompletionInput) cycleCompletion(direction int) {
	if len(ac.currentCompletions) == 0 {
		return
	}

	var newIndex int
	if !ac.firstTabPressed && direction > 0 {
		newIndex = 0
		ac.firstTabPressed = true
	} else {
		currentIndex := ac.currentSuggestionIndex
		newIndex = (currentIndex + direction + len(ac.currentCompletions)) % len(ac.currentCompletions)
	}

	currentValue := ac.TextInput.Value()
	completion := ac.currentCompletions[newIndex].FullText

	lastTokenIndex, _ := ac.CompletionProvider.GetLastToken(currentValue)

	var newValue string
	if lastTokenIndex > 0 {
		newValue = currentValue[:lastTokenIndex] + completion
	} else {
		newValue = completion
	}

	ac.previousValue = newValue
	ac.TextInput.SetValue(newValue)
	ac.TextInput.CursorEnd()
	ac.currentSuggestionIndex = newIndex
}

func (ac *AutoCompletionInput) updateCompletions() {
	value := ac.TextInput.Value()
	ac.previousValue = value

	suggestions := ac.CompletionProvider.GetCompletions(value)
	ac.Suggestions = suggestions
	ac.currentSuggestionIndex = 0
	ac.firstTabPressed = false
	ac.SignatureHelp = ac.CompletionProvider.GetSignatureHelp(value)
	ac.currentCompletions = make([]Completion, 0, len(suggestions))
	var inputSuggestions []string

	for _, suggestion := range suggestions {
		matchedPart := findMatchedPrefix(value, suggestion)
		restPart := strings.TrimPrefix(suggestion, matchedPart)

		ac.currentCompletions = append(ac.currentCompletions, Completion{
			FullText:    suggestion,
			MatchedPart: matchedPart,
			RestPart:    restPart,
		})

		inputSuggestions = append(inputSuggestions, value+restPart)
	}

	ac.TextInput.SetSuggestions(inputSuggestions)
}

func findMatchedPrefix(input, suggestion string) string {
	if strings.HasPrefix(suggestion, input) {
		return input
	}

	lastIndex := strings.LastIndexAny(input, " ,|&~(.:")
	if lastIndex != -1 {
		partialInput := input[lastIndex+1:]
		if strings.HasPrefix(suggestion, partialInput) {
			return partialInput
		}
	}

	return ""
}

func (ac *AutoCompletionInput) View() string {
	var builder strings.Builder

	builder.WriteString(ac.TextInput.View())

	// Show suggestions if available, otherwise show signature help
	if len(ac.Suggestions) > 0 {
		builder.WriteString("\n")

		visibleCount := len(ac.currentCompletions)
		for i := 0; i < visibleCount; i++ {
			completion := ac.currentCompletions[i]

			if i == ac.currentSuggestionIndex {
				builder.WriteString(ac.Styles.Selected.Render(completion.FullText))
			} else {
				matchedPart := ac.Styles.Matched.Render(completion.MatchedPart)
				restPart := ac.Styles.Text.Render(completion.RestPart)
				builder.WriteString(matchedPart)
				builder.WriteString(restPart)
			}

			if i < visibleCount-1 {
				builder.WriteString(ac.Styles.Text.Render(" "))
			}
		}

		if len(ac.currentCompletions) > visibleCount {
			builder.WriteString(" +" +
				ac.Styles.Text.Render(string(rune('0'+len(ac.currentCompletions)-visibleCount))+" more"))
		}
	} else if ac.SignatureHelp != "" {
		builder.WriteString("\n")
		builder.WriteString(ac.SignatureHelp)
	} else if ac.TextInput.Value() != "" {
		builder.WriteString(ac.Styles.Dimmed.Render("\nNo suggestions"))
	}

	return builder.String()
}
