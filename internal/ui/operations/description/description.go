package description

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	context  *context.MainContext
	keyMap   config.KeyMappings[key.Binding]
	input    textarea.Model
	revision string
}

func (o Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Cancel,
		o.keyMap.InlineDescribe.Accept,
	}
}

func (o Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o Operation) Width() int {
	return o.input.Width()
}

func (o Operation) Height() int {
	return o.input.Height()
}

func (o Operation) SetWidth(w int) {
	o.input.SetWidth(w)
}

func (o Operation) SetHeight(h int) {
	o.input.SetHeight(h)
}

func (o Operation) IsFocused() bool {
	return true
}

func (o Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderOverDescription {
		return ""
	}
	return o.View()
}

func (o Operation) Name() string {
	return "desc"
}

func (o Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, o.keyMap.Cancel):
			return o, common.Close
		case key.Matches(keyMsg, o.keyMap.InlineDescribe.Accept):
			if o.input.Value() != "" {
				return o, o.context.RunCommand(jj.SetDescription(o.revision, o.input.Value()), common.Close, common.Refresh)
			}
			return o, common.Close
		}
	}
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)

	newValue := o.input.Value()
	h := lipgloss.Height(newValue)
	if h >= o.input.Height() {
		o.input.SetHeight(h + 1)
	}

	return o, cmd
}

func (o Operation) Init() tea.Cmd {
	return nil
}

func (o Operation) View() string {
	return o.input.View()
}

func NewOperation(context *context.MainContext, revision string) (operations.Operation, tea.Cmd) {
	descOutput, _ := context.RunCommandImmediate(jj.GetDescription(revision))
	desc := string(descOutput)
	h := lipgloss.Height(desc)

	input := textarea.New()
	input.CharLimit = 0
	input.MaxHeight = 10
	input.Prompt = ""
	input.ShowLineNumbers = false
	highlightBackground := lipgloss.AdaptiveColor{
		Light: config.Current.UI.HighlightLight,
		Dark:  config.Current.UI.HighlightDark,
	}
	input.FocusedStyle.Base = lipgloss.NewStyle().Background(highlightBackground)
	input.SetValue(desc)
	input.SetHeight(h + 1)
	input.SetWidth(150)
	input.Focus()

	return Operation{
		context:  context,
		keyMap:   config.Current.GetKeyMap(),
		input:    input,
		revision: revision,
	}, input.Focus()
}
