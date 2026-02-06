package common

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
)

type (
	CloseViewMsg struct {
		Applied bool
	}
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
	SelectionChangedMsg struct {
		Item SelectedItem
	}
	QuickSearchMsg  string
	UpdateRevSetMsg string
	ExecMsg         struct {
		Line string
		Mode ExecMode
	}
	ShowChooseMsg struct {
		Options []string
		Title   string
	}
	ShowInputMsg struct {
		Title  string
		Prompt string
	}
	ExecProcessCompletedMsg struct {
		Err error
		Msg ExecMsg
	}
	FileSearchMsg struct {
		Revset       string
		PreviewShown bool
		Commit       *jj.Commit
		RawFileOut   []byte // raw output from `jj file list`
	}
	ShowPreview     bool
	RunLuaScriptMsg struct {
		Script string
	}
	TogglePasswordMsg struct {
		Prompt   string
		Password chan []byte
	}
	RestoreOperationMsg struct {
		Operation any
	}
	StartAceJumpMsg struct{}
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

func CloseApplied() tea.Msg {
	return CloseViewMsg{Applied: true}
}

func RestoreOperation(op interface{}) tea.Cmd {
	return func() tea.Msg {
		return RestoreOperationMsg{Operation: op}
	}
}

func StartAceJump() tea.Cmd {
	return func() tea.Msg {
		return StartAceJumpMsg{}
	}
}

func SelectionChanged(item SelectedItem) tea.Cmd {
	return func() tea.Msg {
		return SelectionChangedMsg{Item: item}
	}
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

func FileSearch(revset string, preview bool, commit *jj.Commit, rawFileOut []byte) tea.Cmd {
	return func() tea.Msg {
		return FileSearchMsg{
			Commit:       commit,
			RawFileOut:   rawFileOut,
			Revset:       revset,
			PreviewShown: preview,
		}
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

func IsInputMessage(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		return true
	default:
		return false
	}
}
