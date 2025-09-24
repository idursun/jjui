package common

import tea "github.com/charmbracelet/bubbletea"

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type Scope string

const (
	ScopeNone      Scope = ""
	ScopeUI        Scope = "ui"
	ScopeRevisions Scope = "revisions"
	ScopeOplog     Scope = "oplog"
	ScopeDiff      Scope = "diff"
	ScopeRevset    Scope = "revset"
)

type SetScopeMsg struct {
	Scope Scope
}

func SetScope(scope Scope) tea.Cmd {
	return func() tea.Msg {
		return SetScopeMsg{Scope: scope}
	}
}

type Action struct {
	Id     string
	Args   map[string]any
	Switch Scope
}

func (a Action) Get(name string, defaultValue any) any {
	if v, ok := a.Args[name]; ok {
		return v
	}
	return defaultValue
}

func NewAction(id string, args map[string]any) tea.Cmd {
	return func() tea.Msg {
		return InvokeActionMsg{
			Action: Action{Id: id, Args: args},
		}
	}
}

func InvokeAction(action Action) tea.Cmd {
	return func() tea.Msg {
		return InvokeActionMsg{Action: action}
	}
}

type InvokeActionMsg struct {
	Action Action
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

type SquashAction struct {
	ChangeId string
	Files    []string
}
