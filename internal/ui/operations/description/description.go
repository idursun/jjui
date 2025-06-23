package description

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"strconv"
)

type Operation struct {
	context  context.AppContext
	input    textarea.Model
	revision string
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

func (o Operation) RenderPosition() operations.RenderPosition {
	return operations.RenderOverDescription
}

func (o Operation) Render() string {
	return o.View() + strconv.Itoa(o.input.Height())
}

func (o Operation) Name() string {
	return "desc"
}

func (o Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEscape:
			return o, common.Close
		case tea.KeyCtrlD:
			if o.input.Value() != "" {
				return o, o.context.RunCommand(jj.SetDescription(o.revision, o.input.Value()), common.Close, common.Refresh)
			}
			return o, common.Close
		}
	}
	//newValue := o.input.Value()
	//h := lipgloss.Height(newValue)
	//o.input.SetHeight(h)

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
	descOutput, _ := context.RunCommandImmediate(jj.GetDescription(revision))
	desc := string(descOutput)
	h := lipgloss.Height(desc)

	input := textarea.New()
	input.CharLimit = 100
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.SetValue(desc)
	input.SetHeight(h)
	input.SetWidth(150)
	input.Focus()

	return Operation{
		context:  context,
		input:    input,
		revision: revision,
	}, input.Focus()
}
