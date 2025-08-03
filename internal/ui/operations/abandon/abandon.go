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
	current *jj.Commit
	context *context.MainContext
}

func (a *Operation) SetSelectedRevision(commit *jj.Commit) {
	a.current = commit
}

func (a *Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	var cmd tea.Cmd
	a.model, cmd = a.model.Update(msg)
	return a, cmd
}

func (a *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	isSelected := commit != nil && commit.GetChangeId() == a.current.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return a.model.View()
}

func (a *Operation) Name() string {
	return "abandon"
}

func NewOperation(context *context.MainContext, selectedRevisions jj.SelectedRevisions) operations.Operation {
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
	cmd := func(ignoreImmutable bool) tea.Cmd {
		return context.RunCommand(jj.Abandon(selectedRevisions, ignoreImmutable), common.Refresh, common.Close)
	}
	model := confirmation.New(
		[]string{message},
		confirmation.WithAltOption("Yes", cmd(false), cmd(true), key.NewBinding(key.WithKeys("y"))),
		confirmation.WithOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc"))),
		confirmation.WithStylePrefix("abandon"),
	)

	op := &Operation{
		model: &model,
	}
	return op
}
