package test

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/operations"
)

type OperationHost struct {
	closed    bool
	Operation operations.Operation
}

func (o OperationHost) Init() tea.Cmd {
	return nil
}

func (o OperationHost) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.CloseViewMsg, confirmation.CloseMsg:
		o.closed = true
		return o, tea.Quit
	case tea.KeyMsg:
		if op, ok := o.Operation.(operations.HandleKey); ok {
			cmd := op.HandleKey(msg)
			return o, cmd
		}
	}
	if op, ok := o.Operation.(operations.OperationWithOverlay); ok {
		var cmd tea.Cmd
		o.Operation, cmd = op.Update(msg)
		return o, cmd
	}
	return o, nil
}

func (o OperationHost) View() string {
	if o.closed {
		return "closed"
	}
	return o.Operation.Render()
}
