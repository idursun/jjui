package common

import tea "github.com/charmbracelet/bubbletea"

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type Scope int

const (
	ScopeUI Scope = iota
	ScopeRevisions
	ScopeOplog
	ScopeRevset
)

type InvokeActionMsg struct {
	Scope  Scope
	Action any
}

type InlineDescribeAction struct {
	ChangeId string
}

type CursorUpAction struct {
	Amount int
}

type CursorDownAction struct {
	Amount int
}

type EditRevsetAction struct {
	Clear bool
}

type ShowDetailsAction struct{}
type SquashAction struct {
	ChangeId string
	Files    []string
}
type RebaseAction struct {
}

type SwitchToOplogAction struct{}

func InvokeAction(scope Scope, action any) tea.Cmd {
	return func() tea.Msg {
		return InvokeActionMsg{Scope: scope, Action: action}
	}
}
