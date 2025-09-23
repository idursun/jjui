package script

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

type ResumeScriptExecutionMsg struct {
	Execution *ScriptExecution
}

type CancelSkipScriptExecutionMsg struct{}

type ScriptExecution struct {
	Script    *Script
	StepIndex int
}

func StartScript(script *Script) tea.Cmd {
	se := &ScriptExecution{
		Script:    script,
		StepIndex: 0,
	}
	return func() tea.Msg {
		return ResumeScriptExecutionMsg{Execution: se}
	}
}

func (se *ScriptExecution) Resume() tea.Msg {
	if se.StepIndex < len(se.Script.Steps) {
		se.StepIndex++
		return ResumeScriptExecutionMsg{Execution: se}
	}
	return nil
}

func (se *ScriptExecution) Current() IScriptStep {
	if se.StepIndex < len(se.Script.Steps) {
		return se.Script.Steps[se.StepIndex]
	}
	return nil
}

func (se *ScriptExecution) Wait(seconds int) (chan tea.Msg, tea.Cmd) {
	done := make(chan tea.Msg, 1)
	return done, func() tea.Msg {
		select {
		case result := <-done:
			switch result {
			case common.WaitResultContinue:
				return se.Resume()
			case common.WaitResultCancel:
				break
			}
		case <-time.After(time.Duration(seconds) * time.Second):
			break
		}
		return CancelSkipScriptExecutionMsg{}
	}
}
