package abandon

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	model   tea.Model
	context context.AppContext
}

func (a Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	var cmd tea.Cmd
	a.model, cmd = a.model.Update(msg)
	return a, cmd
}

func (a Operation) RenderPosition() operations.RenderPosition {
	return operations.RenderPositionAfter
}

func (a Operation) Render() string {
	return a.model.View()
}

func (a Operation) Name() string {
	return "abandon"
}

func NewOperation(context context.AppContext, selectedRevisions jj.SelectedRevisions) operations.Operation {
	var ids []string
	var conflictingWarning string
	for _, rev := range selectedRevisions.Revisions {
		ids = append(ids, rev.GetChangeId())
		if rev.IsConflicting() {
			conflictingWarning = "conflicting "
		}
	}
	message := fmt.Sprintf("Are you sure you want to abandon this %srevision?", conflictingWarning)
	if len(selectedRevisions.Revisions) > 1 {
		message = fmt.Sprintf("Are you sure you want to abandon %d %srevisions?", len(selectedRevisions.Revisions), conflictingWarning)
	}
	model := confirmation.New(message)
	model.AddOption("Yes", context.RunCommand(jj.Abandon(selectedRevisions), common.Refresh, common.Close), key.NewBinding(key.WithKeys("y")))
	model.AddOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc")))

	op := Operation{
		model: &model,
	}
	return op
}
