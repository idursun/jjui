package common

import tea "github.com/charmbracelet/bubbletea"

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type ActionScope int

const (
	ActionScopeUI ActionScope = iota
	ActionScopeRevisions
	ActionScopeOplog
	ActionScopeRevset
)

type InvokeActionMsg struct {
	Scope  ActionScope
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

type SwitchToOplogAction struct{}

func InvokeAction(scope ActionScope, action any) tea.Cmd {
	return func() tea.Msg {
		return InvokeActionMsg{Scope: scope, Action: action}
	}
}
