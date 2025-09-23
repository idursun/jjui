package common

import tea "github.com/charmbracelet/bubbletea"

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type InvokeActionMsg struct {
	Action any
}

type InlineDescribeAction struct {
	ChangeId string
}

func InvokeAction(action any) tea.Cmd {
	return func() tea.Msg {
		return InvokeActionMsg{Action: action}
	}
}
