package bookmarkpane

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/render"
)

func (m *Model) runBookmarkCommands(commands []jj.CommandArgs) tea.Cmd {
	if len(commands) == 0 {
		return nil
	}
	if len(commands) == 1 {
		return m.context.RunCommand(commands[0], common.Refresh)
	}
	cmds := make([]tea.Cmd, 0, len(commands))
	for i, command := range commands {
		continuations := []tea.Cmd(nil)
		if i == len(commands)-1 {
			continuations = append(continuations, common.Refresh)
		}
		cmds = append(cmds, m.context.RunCommand(command, continuations...))
	}
	return tea.Sequence(cmds...)
}

func (m *Model) toggleExpandSelected() {
	row, ok := m.selectedRow()
	if !ok || row.Depth > 0 {
		return
	}
	group := &m.tree.Items[row.BookmarkIndex]
	if len(group.Bookmark.Remotes) == 0 {
		return
	}
	target, _ := m.selectedTarget()
	group.Expanded = !group.Expanded
	m.expanded[group.Bookmark.Name] = group.Expanded
	m.applyFilters(false)
	m.selectTarget(target)
}

func (m *Model) revealSelected() tea.Cmd {
	commitID := m.selectedCommitID()
	target, _ := m.selectedTarget()
	if commitID == "" {
		return nil
	}
	if m.currentCommitID != commitID && !slices.Contains(m.visibleCommitIDs, commitID) {
		return intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("Bookmark %s is not visible in the current revisions list", target)})
	}
	return func() tea.Msg {
		return RevealRevisionMsg{CommitID: commitID}
	}
}

func (m *Model) showSelectedInRevisions() tea.Cmd {
	target, ok := m.selectedTarget()
	if !ok {
		return nil
	}
	commitID := m.selectedCommitID()
	return func() tea.Msg {
		return ShowBookmarkInRevisionsMsg{Target: target, CommitID: commitID}
	}
}

func (m *Model) editSelected() tea.Cmd {
	target, ok := m.selectedTarget()
	if !ok {
		return nil
	}
	return m.openConfirmation(
		[]string{fmt.Sprintf("Are you sure you want to edit %s?", target)},
		m.context.RunCommand(jj.Edit(target, false), common.Refresh),
	)
}

func (m *Model) newFromSelected() tea.Cmd {
	_, node, ok := m.selectedBookmarkAndNode()
	if !ok {
		return nil
	}
	commitID := node.CommitID
	if commitID == "" {
		return nil
	}
	target := node.Target()
	return m.openConfirmation(
		[]string{fmt.Sprintf("Are you sure you want to create a new change from %s?", target)},
		m.context.RunCommand(jj.New(jj.NewSelectedRevisions(&jj.Commit{ChangeId: commitID})), common.Refresh),
	)
}

func (m *Model) pushSelected() tea.Cmd {
	bookmark, node, ok := m.selectedBookmarkAndNode()
	if !ok {
		return nil
	}
	flags := []string{"--bookmark", bookmark.Name}
	if node.IsRemote() && node.Remote != "" {
		flags = append(flags, "--remote", node.Remote)
	}
	return m.openConfirmation(
		[]string{fmt.Sprintf("Are you sure you want to push %s?", node.Target())},
		m.context.RunCommand(jj.GitPush(flags...), common.Refresh),
	)
}

func (m *Model) fetchSelected() tea.Cmd {
	bookmark, node, ok := m.selectedBookmarkAndNode()
	if !ok {
		return nil
	}
	flags := []string{"--branch", bookmark.Name}
	if node.IsRemote() && node.Remote != "" {
		flags = append(flags, "--remote", node.Remote)
	}
	return m.openConfirmation(
		[]string{fmt.Sprintf("Are you sure you want to fetch %s?", node.Target())},
		m.context.RunCommand(jj.GitFetch(flags...), common.Refresh),
	)
}

func (m *Model) renameSelected() tea.Cmd {
	bookmark, _, ok := m.selectedLocalBookmark()
	if !ok {
		return nil
	}
	m.pendingInput = pendingInputRename
	return input.ShowWithTitle("Rename bookmark", "", bookmark.Name)
}

func (m *Model) createSelected() tea.Cmd {
	return func() tea.Msg {
		return BeginCreateBookmarkMsg{}
	}
}

func (m *Model) deleteSelected() tea.Cmd {
	return m.confirmBookmarkCommands("delete", m.deleteCommands(), m.distinctSelectedBookmarkCount())
}

func (m *Model) forgetSelected() tea.Cmd {
	return m.confirmBookmarkCommands("forget", m.forgetCommands(), m.distinctSelectedBookmarkCount())
}

func (m *Model) trackSelected() tea.Cmd {
	return m.confirmBookmarkCommands("track", m.trackCommands(), m.distinctSelectedBookmarkCount())
}

func (m *Model) untrackSelected() tea.Cmd {
	return m.confirmBookmarkCommands("untrack", m.untrackCommands(), m.distinctSelectedBookmarkCount())
}

func (m *Model) distinctSelectedBookmarkCount() int {
	seen := make(map[string]bool)
	for _, selection := range m.selectionsForBookmarkOperation() {
		seen[selection.bookmark.Name] = true
	}
	return len(seen)
}

func (m *Model) moveSelected() tea.Cmd {
	bookmark, selected, ok := m.selectedBookmarkAndNode()
	if !ok {
		return nil
	}
	if bookmark.Local == nil || !bookmark.Local.Present {
		return intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("No local bookmark for %s", bookmark.Name)})
	}
	if selected.IsRemote() {
		return nil
	}
	m.pendingSelectionHint = bookmark.Name
	return func() tea.Msg {
		return BeginMoveBookmarkMsg{Name: bookmark.Name}
	}
}

func (m *Model) deleteCommands() []jj.CommandArgs {
	var commands []jj.CommandArgs
	seen := make(map[string]bool)
	for _, selection := range m.selectionsForBookmarkOperation() {
		if selection.node.IsRemote() || seen[selection.bookmark.Name] {
			continue
		}
		seen[selection.bookmark.Name] = true
		if command, ok := jj.BookmarkDeleteCommand(selection.bookmark); ok {
			commands = append(commands, command)
		}
	}
	return commands
}

func (m *Model) forgetCommands() []jj.CommandArgs {
	var commands []jj.CommandArgs
	seen := make(map[string]bool)
	for _, selection := range m.selectionsForBookmarkOperation() {
		if seen[selection.bookmark.Name] {
			continue
		}
		seen[selection.bookmark.Name] = true
		if command, ok := jj.BookmarkForgetCommand(selection.bookmark); ok {
			commands = append(commands, command)
		}
	}
	return commands
}

func (m *Model) trackCommands() []jj.CommandArgs {
	defaultRemote := ""
	var commands []jj.CommandArgs
	seen := make(map[string]bool)
	for _, selection := range m.selectionsForBookmarkOperation() {
		key := selection.node.Target()
		if seen[key] {
			continue
		}
		seen[key] = true
		if selection.node.IsRemote() {
			if remote, ok := selection.node.bookmarkRemote(selection.bookmark); ok {
				if command, ok := jj.BookmarkTrackRemoteCommand(selection.bookmark, remote); ok {
					commands = append(commands, command)
				}
			}
			continue
		}
		if defaultRemote == "" {
			defaultRemote = m.defaultTrackRemote()
		}
		if command, ok := jj.BookmarkTrackLocalCommand(selection.bookmark, defaultRemote); ok {
			commands = append(commands, command)
		}
	}
	return commands
}

func (m *Model) untrackCommands() []jj.CommandArgs {
	var commands []jj.CommandArgs
	seen := make(map[string]bool)
	for _, selection := range m.selectionsForBookmarkOperation() {
		if selection.node.IsRemote() {
			key := selection.bookmark.Name + "@" + selection.node.Remote
			if seen[key] {
				continue
			}
			seen[key] = true
			if remote, ok := selection.node.bookmarkRemote(selection.bookmark); ok {
				if command, ok := jj.BookmarkUntrackRemoteCommand(selection.bookmark, remote); ok {
					commands = append(commands, command)
				}
			}
			continue
		}
		for _, remote := range selection.bookmark.Remotes {
			key := selection.bookmark.Name + "@" + remote.Remote
			if seen[key] {
				continue
			}
			seen[key] = true
			if command, ok := jj.BookmarkUntrackRemoteCommand(selection.bookmark, remote); ok {
				commands = append(commands, command)
			}
		}
	}
	return commands
}

func (m *Model) defaultTrackRemote() string {
	output, err := m.context.RunCommandImmediate(jj.GitRemoteList())
	if err != nil {
		return ""
	}
	remotes := jj.ParseRemoteListOutput(string(output))
	if len(remotes) == 0 {
		return ""
	}
	return remotes[0]
}

func (m *Model) confirmBookmarkCommands(verb string, commands []jj.CommandArgs, bookmarkCount int) tea.Cmd {
	if len(commands) == 0 {
		return nil
	}
	target := "the selected bookmark"
	if bookmarkCount > 1 {
		target = "the selected bookmarks"
	}
	return m.openConfirmation(
		[]string{fmt.Sprintf("Are you sure you want to %s %s?", verb, target)},
		m.runBookmarkCommands(commands),
	)
}

func (m *Model) openConfirmation(messages []string, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	m.confirmation = confirmation.New(
		messages,
		confirmation.WithZIndex(render.ZDialogs),
		confirmation.WithOption("Yes",
			tea.Batch(cmd, confirmation.Close),
			key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No",
			confirmation.Close,
			key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)
	return m.confirmation.Init()
}
