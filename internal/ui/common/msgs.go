package common

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	CloseViewMsg   struct{}
	ToggleHelpMsg  struct{}
	AutoRefreshMsg struct{}
	RefreshMsg     struct {
		SelectedRevision string
		KeepSelections   bool
	}
	ShowDiffMsg              string
	UpdateRevisionsFailedMsg struct {
		Output string
		Err    error
	}
	UpdateRevisionsSuccessMsg struct{}
	UpdateBookmarksMsg        struct {
		Bookmarks []string
		Revision  string
	}
	CommandRunningMsg   string
	CommandCompletedMsg struct {
		Output string
		Err    error
	}
	SelectionChangedMsg struct{}
	QuickSearchMsg      string
	UpdateRevSetMsg     string
	ExecMsg             struct {
		Line string
		Mode ExecMode
	}
)

type State int

const (
	Loading State = iota
	Ready
	Error
)

func Close() tea.Msg {
	return CloseViewMsg{}
}

func SelectionChanged() tea.Msg {
	return SelectionChangedMsg{}
}

func RefreshAndSelect(selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		return RefreshMsg{SelectedRevision: selectedRevision}
	}
}

func RefreshAndKeepSelections() tea.Msg {
	return RefreshMsg{KeepSelections: true}
}

func Refresh() tea.Msg {
	return RefreshMsg{}
}

func ToggleHelp() tea.Msg {
	return ToggleHelpMsg{}
}

func CommandRunning(args []string) tea.Cmd {
	return func() tea.Msg {
		command := "jj " + strings.Join(args, " ")
		return CommandRunningMsg(command)
	}
}

func UpdateRevSet(revset string) tea.Cmd {
	return func() tea.Msg {
		return UpdateRevSetMsg(revset)
	}
}

type ExecMode struct {
	Mode   string
	Prompt string
}

var ExecJJ ExecMode = ExecMode{
	Mode:   "jj",
	Prompt: ": ",
}

var ExecShell ExecMode = ExecMode{
	Mode:   "sh",
	Prompt: "$ ",
}
