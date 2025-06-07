package quick_describe

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type QuickDescribeOperation struct {
	context  context.AppContext
	revision string
	message  textarea.Model
}

func (s QuickDescribeOperation) Init() tea.Cmd {
	return textarea.Blink
}

func (s QuickDescribeOperation) View() string {
	return s.message.View()
}

func (s QuickDescribeOperation) IsFocused() bool {
	return true
}

func (s QuickDescribeOperation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return s, common.Close
		case "enter":
			return s, s.context.RunCommand(jj.QuickDescribe(s.revision, s.message.Value()), common.Close, common.Refresh)
		}
	}
	var cmd tea.Cmd
	s.message, cmd = s.message.Update(msg)
	return s, cmd
}

func (s QuickDescribeOperation) Render() string {
	return s.message.View()
}

func (s QuickDescribeOperation) RenderPosition() operations.RenderPosition {
	return operations.RenderPositionAfter
}

func (s QuickDescribeOperation) Name() string {
	return "describe"
}

func NewQuickDescribeOperation(context context.AppContext, changeId string) (operations.Operation, tea.Cmd) {

	description, _ := context.RunCommandImmediate(jj.Args(
		"show", "-r", changeId, "--ignore-working-copy", "--no-patch", "--template", "description",
	))

	t := textarea.New()
	t.CharLimit = 120
	t.ShowLineNumbers = false
	t.SetValue(string(description))
	t.SetWidth(60)
	t.SetHeight(1)
	t.Focus()

	op := QuickDescribeOperation{
		message:  t,
		revision: changeId,
		context:  context,
	}
	return op, op.Init()
}
