package description

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	context  context.AppContext
	input    textarea.Model
	revision string
}

func (o Operation) IsFocused() bool {
	return true
}

func (o Operation) RenderPosition() operations.RenderPosition {
	return operations.RenderOverDescription
}

func (o Operation) Render() string {
	return o.View()
}

func (o Operation) Name() string {
	return "desc"
}

func (o Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEscape:
			return o, common.Close
		case "enter":
			if o.input.Value() != "" {
				return o, o.context.RunCommand(jj.SetDescription(o.revision, o.input.Value()), common.Close, common.Refresh)
			}
			return o, common.Close
		}
	}
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return o, cmd
}

func (o Operation) Init() tea.Cmd {
	return nil
}

func (o Operation) View() string {
	return o.input.View()
}

func NewOperation(context context.AppContext, revision string) (operations.Operation, tea.Cmd) {
	input := textarea.New()
	input.CharLimit = 100
	input.Prompt = "| "
	input.SetHeight(4)
	input.SetWidth(100)
	input.ShowLineNumbers = false
	input.Focus()

	desc, _ := context.RunCommandImmediate(jj.GetDescription(revision))
	input.SetValue(string(desc))

	return Operation{
		context:  context,
		input:    input,
		revision: revision,
	}, input.Focus()
}
