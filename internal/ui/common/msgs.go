package common

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
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
	UpdateBookmarksMsg struct {
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
