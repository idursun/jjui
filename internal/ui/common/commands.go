package common

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
)

type UICommands struct {
	jj jj.Commands
}

func (c UICommands) GitFetch(revision string) tea.Cmd {
	return RunCommand(c.jj.GitFetch(), Refresh(revision))
}

func (c UICommands) GitPush(revision string) tea.Cmd {
	return RunCommand(c.jj.GitPush(), Refresh(revision))
}

func (c UICommands) Rebase(from, to string, operation Operation) tea.Cmd {
	rebase := c.jj.RebaseCommand
	if operation == RebaseBranchOperation {
		rebase = c.jj.RebaseBranchCommand
	}
	cmd := rebase(from, to)
	return RunCommand(cmd, Refresh(to))
}

func (c UICommands) Squash(from, destination string) tea.Cmd {
	cmd := c.jj.Squash(from, destination)
	return tea.Sequence(
		CommandRunning(strings.Join(cmd.Args(), " ")),
		tea.ExecProcess(cmd.GetCommand(), func(err error) tea.Msg {
			return CommandCompletedMsg{Output: "", Err: err}
		}),
		Refresh(destination),
	)
}

func (c UICommands) SetDescription(revision string, description string) tea.Cmd {
	return RunCommand(c.jj.SetDescription(revision, description), Refresh(revision), Close)
}

func (c UICommands) MoveBookmark(revision string, bookmark string) tea.Cmd {
	return RunCommand(c.jj.MoveBookmark(revision, bookmark), Refresh(revision), Close)
}

func (c UICommands) DeleteBookmark(revision, bookmark string) tea.Cmd {
	return RunCommand(c.jj.DeleteBookmark(bookmark), Refresh(revision), Close)
}

func (c UICommands) FetchRevisions(revset string) tea.Cmd {
	return func() tea.Msg {
		graphLines, err := c.jj.GetCommits(revset)
		if err != nil {
			return UpdateRevisionsFailedMsg(err)
		}
		return UpdateRevisionsMsg(graphLines)
	}
}

func (c UICommands) FetchBookmarks(revision string) tea.Cmd {
	return func() tea.Msg {
		cmd := c.jj.ListBookmark(revision)
		// TODO: handle error
		output, _ := cmd.CombinedOutput()
		var bookmarks []string
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			bookmarks = append(bookmarks, line)
		}
		return UpdateBookmarksMsg{
			Bookmarks: bookmarks,
			Revision:  revision,
		}
	}
}

func (c UICommands) SetBookmark(revision string, name string) tea.Cmd {
	return RunCommand(c.jj.SetBookmark(revision, name), Refresh(revision), Close)
}

func (c UICommands) GetDiff(revision string, fileName string) tea.Cmd {
	return func() tea.Msg {
		output, _ := c.jj.Diff(revision, fileName).CombinedOutput()
		return ShowDiffMsg(output)
	}
}

func (c UICommands) Restore(revision string, files []string) tea.Cmd {
	return RunCommand(
		c.jj.Restore(revision, files),
		Refresh(revision),
	)
}

func (c UICommands) Edit(revision string) tea.Cmd {
	return RunCommand(c.jj.Edit(revision), Refresh("@"))
}

func (c UICommands) DiffEdit(revision string) tea.Cmd {
	return tea.ExecProcess(c.jj.DiffEdit(revision).GetCommand(), func(err error) tea.Msg {
		return RefreshMsg{SelectedRevision: revision}
	})
}

func (c UICommands) Split(revision string, files []string) tea.Cmd {
	return tea.ExecProcess(c.jj.Split(revision, files).GetCommand(), func(err error) tea.Msg {
		return RefreshMsg{SelectedRevision: revision}
	})
}

func (c UICommands) Abandon(revision string) tea.Cmd {
	return RunCommand(c.jj.Abandon(revision), Refresh("@"), Close)
}

func (c UICommands) NewRevision(from string) tea.Cmd {
	return RunCommand(c.jj.New(from), Refresh("@"))
}

func (c UICommands) Undo() tea.Cmd {
	return RunCommand(c.jj.Undo(), Refresh("@"))
}

func (c UICommands) Status(revision string) tea.Cmd {
	cmd := c.jj.Status(revision)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return func() tea.Msg {
			summary := strings.Split(strings.TrimSpace(string(output)), "\n")
			return UpdateCommitStatusMsg(summary)
		}
	}
	return func() tea.Msg {
		return CommandCompletedMsg{
			Output: string(output),
			Err:    err,
		}
	}
}

func NewUICommands(jj jj.Commands) UICommands {
	return UICommands{jj}
}
