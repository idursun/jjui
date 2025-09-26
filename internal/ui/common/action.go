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
	ScopePreview   Scope = "preview"
	ScopeUndo      Scope = "undo"
	ScopeBookmarks Scope = "bookmarks"
)

type SetScopeMsg struct {
	Scope Scope
}

type Action struct {
	Id     string
	Args   map[string]any
	Switch Scope
	Next   []Action
}

func (a Action) GetNext() tea.Cmd {
	if len(a.Next) == 0 {
		return nil
	}
	nextAction := a.Next[0]
	nextAction.Next = a.Next[1:]
	return InvokeAction(nextAction)
}

func (a Action) Wait() (WaitChannel, tea.Cmd) {
	ch := make(WaitChannel, 1)
	return ch, func() tea.Msg {
		select {
		case <-ch:
			nextAction := a.Next[0]
			nextAction.Next = a.Next[1:]
			return InvokeActionMsg{Action: nextAction}
		}
	}
}

func (a Action) Get(name string, defaultValue any) any {
	if a.Args == nil {
		return defaultValue
	}
	if v, ok := a.Args[name]; ok {
		return v
	}
	return defaultValue
}

func InvokeAction(action Action) tea.Cmd {
	return func() tea.Msg {
		return InvokeActionMsg{Action: action}
	}
}

type InvokeActionMsg struct {
	Action Action
}
