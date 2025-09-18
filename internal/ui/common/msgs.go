package common

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
)

type (
	CloseViewMsg  struct{}
	ToggleHelpMsg struct{}
	RefreshMsg    struct {
		SelectedRevision string
		KeepSelections   bool
	}
	ShowDiffMsg              string
	UpdateRevisionsFailedMsg struct {
		Output string
		Err    error
	}
	UpdateRevisionsSuccessMsg struct{}
	CommandRunningMsg         string
	CommandCompletedMsg       struct {
		Output string
		Err    error
	}
	QuickSearchMsg  string
	UpdateRevSetMsg string
	ExecMsg         struct {
		Line string
		Mode ExecMode
	}
	FileSearchMsg struct {
		Revset       string
		PreviewShown bool
		Commit       *models.Commit
		RawFileOut   []byte // raw output from `jj file list`
	}
	LoadDiffLayoutMsg struct {
		Args jj.DiffCommandArgs
	}
	LoadOplogLayoutMsg struct{}
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
