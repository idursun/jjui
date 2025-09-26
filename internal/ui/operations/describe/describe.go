package describe

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ operations.Operation = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)

type Operation struct {
	context  *context.MainContext
	keyMap   config.KeyMappings[key.Binding]
	input    textarea.Model
	revision string
}

func (o Operation) GetActionMap() map[string]actions.Action {
	return config.Current.GetBindings("inline_describe")
}

func (o Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Cancel,
		o.keyMap.InlineDescribe.Editor,
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

func (o Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "inline_describe.accept":
			return o, o.context.RunCommand(jj.SetDescription(o.revision, o.input.Value()), common.Refresh)
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

func NewOperation(context *context.MainContext, revision string, width int) Operation {
	descOutput, _ := context.RunCommandImmediate(jj.GetDescription(revision))
	desc := string(descOutput)
	h := lipgloss.Height(desc)

	selectedStyle := common.DefaultPalette.Get("revisions selected")

	input := textarea.New()
	input.CharLimit = 0
	input.MaxHeight = 10
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.FocusedStyle.Base = selectedStyle.Underline(false).Strikethrough(false).Reverse(false).Blink(false)
	input.FocusedStyle.CursorLine = input.FocusedStyle.Base
	input.SetValue(desc)
	input.SetHeight(h + 1)
	input.SetWidth(width)
	input.Focus()

	return Operation{
		context:  context,
		keyMap:   config.Current.GetKeyMap(),
		input:    input,
		revision: revision,
	}
}
