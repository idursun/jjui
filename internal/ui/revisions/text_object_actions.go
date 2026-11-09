package revisions

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/workspace"
)

// focusedLocalBookmark returns the focused bookmark when it is a local one,
// otherwise a message explaining why the action does not apply.
func (m *Model) focusedLocalBookmark(action string) (*textObject, tea.Cmd) {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil, nil
	}
	if obj.Remote != "" {
		return nil, intents.Invoke(intents.AddMessage{Text: "Only local bookmarks can be " + action})
	}
	return obj, nil
}

func (m *Model) renameFocusedBookmark() tea.Cmd {
	obj, cmd := m.focusedLocalBookmark("renamed")
	if obj == nil {
		return cmd
	}
	return m.setBaseOperation(bookmark.NewRenameBookmarkOperation(m.context, m.SelectedRevision().GetChangeId(), obj.Value))
}

// localBookmarkNames lists the local bookmarks among targets, or returns a
// message when there are none.
func localBookmarkNames(targets []textObject, action string) ([]string, tea.Cmd) {
	var names []string
	for _, target := range targets {
		if target.Remote == "" && !slices.Contains(names, target.Value) {
			names = append(names, target.Value)
		}
	}
	if len(names) == 0 {
		return nil, intents.Invoke(intents.AddMessage{Text: "Only local bookmarks can be " + action})
	}
	return names, nil
}

func (m *Model) moveBookmarks() tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	names, cmd := localBookmarkNames(targets, "moved")
	if cmd != nil {
		return cmd
	}
	return m.setBaseOperation(bookmark.NewMoveBookmarkOperation(m.context, names, m.SelectedRevision()))
}

// advanceBookmarks moves the local bookmarks among the targets forward to jj's
// advance target, @ by default.
func (m *Model) advanceBookmarks() tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	names, cmd := localBookmarkNames(targets, "advanced")
	if cmd != nil {
		return cmd
	}
	return m.runBookmarkCommands([]jj.CommandArgs{jj.BookmarkAdvance(names...)})
}

type loadedBookmark struct {
	textObject
	commit *jj.Commit
}

// checkedBookmarkRows finds the checked bookmarks on loaded rows. A conflicted
// bookmark has one entry per revision.
func (m *Model) checkedBookmarkRows() map[checkedObject][]loadedBookmark {
	found := make(map[checkedObject][]loadedBookmark)
	if !slices.ContainsFunc(m.checked, func(c checkedObject) bool { return c.kind == textObjectBookmark }) {
		return found
	}
	for _, row := range m.rows {
		if row.Commit == nil {
			continue
		}
		for _, o := range textObjectsForRow(row) {
			if o.Kind != textObjectBookmark {
				continue
			}
			if key := checkedBookmark(o.Value, o.Remote); slices.Contains(m.checked, key) {
				found[key] = append(found[key], loadedBookmark{o, row.Commit})
			}
		}
	}
	return found
}

// bookmarkTargets returns what a bookmark action applies to: the checked
// bookmarks, or the focused one when none are checked.
func (m *Model) bookmarkTargets() []textObject {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil
	}
	var targets []textObject
	found := m.checkedBookmarkRows()
	for _, object := range m.checked {
		if rows := found[object]; len(rows) > 0 {
			targets = append(targets, rows[0].textObject)
		}
	}
	if len(targets) == 0 {
		return []textObject{*obj}
	}
	return targets
}

func (m *Model) runBookmarkCommands(commands []jj.CommandArgs) tea.Cmd {
	next := tea.Cmd(common.Refresh)
	for _, command := range slices.Backward(commands) {
		next = m.context.RunCommand(command, next)
	}
	return next
}

func (m *Model) listAllBookmarks() ([]jj.Bookmark, tea.Cmd) {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return nil, intents.Invoke(intents.AddMessage{Text: "Could not list bookmarks", Err: err})
	}
	return jj.ParseBookmarkListOutput(string(output)), nil
}

func hasRemoteBookmark(targets []textObject) bool {
	return slices.ContainsFunc(targets, func(o textObject) bool { return o.Remote != "" })
}

// deleteBookmarks deletes local bookmarks. Remote bookmarks are marked deleted
// so that pushing them deletes them on the remote; an untracked one has no
// local bookmark until it is tracked.
func (m *Model) deleteBookmarks() tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	var listing []jj.Bookmark
	if hasRemoteBookmark(targets) {
		var cmd tea.Cmd
		if listing, cmd = m.listAllBookmarks(); cmd != nil {
			return cmd
		}
	}
	var commands []jj.CommandArgs
	var names []string
	for _, target := range targets {
		if target.Remote != "" && !isRemoteBookmarkTracked(listing, target.Value, target.Remote) {
			commands = append(commands, jj.BookmarkTrack(target.Value, target.Remote))
		}
		if !slices.Contains(names, target.Value) {
			names = append(names, target.Value)
		}
	}
	return m.runBookmarkCommands(append(commands, jj.BookmarkDelete(names...)))
}

// pushBookmarks pushes the bookmarks; remote bookmarks go to their remote,
// which also pushes a deletion marked with d. A push takes a single remote, so
// there is one command per remote.
func (m *Model) pushBookmarks() tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	var remotes []string
	byRemote := make(map[string][]string)
	for _, target := range targets {
		if _, ok := byRemote[target.Remote]; !ok {
			remotes = append(remotes, target.Remote)
		}
		byRemote[target.Remote] = append(byRemote[target.Remote], "--bookmark", jj.ExactStringPattern(target.Value))
	}
	commands := make([]jj.CommandArgs, 0, len(remotes))
	for _, remote := range remotes {
		args := byRemote[remote]
		if remote != "" {
			args = append(args, "--remote", remote)
		}
		commands = append(commands, jj.GitPush(args...))
	}
	return m.runBookmarkCommands(commands)
}

// trackBookmarks tracks or untracks remote bookmarks, or local bookmarks
// against the default remote. Bookmarks already in that state are skipped.
func (m *Model) trackBookmarks(track bool) tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	defaultRemote := ""
	if slices.ContainsFunc(targets, func(o textObject) bool { return o.Remote == "" }) {
		if defaultRemote = m.defaultTrackRemote(); defaultRemote == "" {
			return intents.Invoke(intents.AddMessage{Text: "There is no git remote to track " + targets[0].Value + " on"})
		}
	}
	listing, cmd := m.listAllBookmarks()
	if cmd != nil {
		return cmd
	}
	var commands []jj.CommandArgs
	skipped := ""
	for _, target := range targets {
		remote := target.Remote
		if remote == "" {
			remote = defaultRemote
		}
		switch tracked := isRemoteBookmarkTracked(listing, target.Value, remote); {
		case track && !tracked:
			commands = append(commands, jj.BookmarkTrack(target.Value, remote))
		case !track && tracked:
			commands = append(commands, jj.BookmarkUntrack(target.Value, remote))
		case track:
			skipped = target.Value + "@" + remote + " is already tracked"
		default:
			skipped = target.Value + "@" + remote + " is not tracked"
		}
	}
	if len(commands) == 0 {
		return intents.Invoke(intents.AddMessage{Text: skipped})
	}
	return m.runBookmarkCommands(commands)
}

// isRemoteBookmarkTracked reads a `jj bookmark list` listing because the log
// output does not show the tracking state.
func isRemoteBookmarkTracked(listing []jj.Bookmark, name string, remote string) bool {
	for _, b := range listing {
		if b.Name != name {
			continue
		}
		for _, r := range b.Remotes {
			if r.Remote == remote {
				return r.Tracked
			}
		}
	}
	return false
}

// defaultTrackRemote picks the remote local bookmarks are tracked on, the
// same way the bookmark pane does: git.default_remote when it exists,
// otherwise the first remote.
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

func (m *Model) deleteFocusedTag() tea.Cmd {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectTag {
		return nil
	}
	if obj.Remote != "" {
		return intents.Invoke(intents.AddMessage{Text: "Only local tags can be deleted"})
	}
	return m.context.RunCommand(jj.TagDelete(obj.Value), common.Refresh)
}

// revisionPartTargets returns the revisions owning the checked objects of the
// focused kind, or the focused revision when none are checked.
func (m *Model) revisionPartTargets(kind string) []*jj.Commit {
	obj := m.focusedTextObject()
	revision := m.SelectedRevision()
	if obj == nil || obj.Kind != kind || revision == nil {
		return nil
	}
	loaded := make(map[string]*jj.Commit, len(m.rows))
	for _, row := range m.rows {
		if row.Commit != nil {
			loaded[row.Commit.CommitId] = row.Commit
		}
	}
	var commits []*jj.Commit
	for _, object := range m.checked {
		if commit, ok := loaded[object.commitId]; ok && object.kind == kind {
			commits = append(commits, commit)
		}
	}
	if len(commits) == 0 {
		return []*jj.Commit{revision}
	}
	return commits
}

// describeRevisionParts opens the editor for the revisions of the checked
// descriptions; checked revisions alone do not count.
func (m *Model) describeRevisionParts() tea.Cmd {
	commits := m.revisionPartTargets(textObjectDescription)
	if len(commits) == 0 {
		return nil
	}
	return m.startDescribe(intents.Describe{Selected: jj.NewSelectedRevisions(commits...)})
}

// forgetBookmarks forgets the bookmarks without marking a deletion to push.
// Remote and tracked bookmarks are forgotten with --include-remotes, otherwise
// their remote bookmarks stay behind; they come back on the next fetch if they
// still exist on the remote.
func (m *Model) forgetBookmarks() tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	var listing []jj.Bookmark
	if slices.ContainsFunc(targets, func(o textObject) bool { return o.Remote == "" }) {
		var cmd tea.Cmd
		if listing, cmd = m.listAllBookmarks(); cmd != nil {
			return cmd
		}
	}
	var plain, withRemotes []string
	for _, target := range targets {
		if target.Remote != "" || isBookmarkTracked(listing, target.Value) {
			if !slices.Contains(withRemotes, target.Value) {
				withRemotes = append(withRemotes, target.Value)
			}
		} else {
			plain = append(plain, target.Value)
		}
	}
	plain = slices.DeleteFunc(plain, func(name string) bool { return slices.Contains(withRemotes, name) })
	var commands []jj.CommandArgs
	if len(plain) > 0 {
		commands = append(commands, jj.BookmarkForget(plain...))
	}
	if len(withRemotes) > 0 {
		commands = append(commands, jj.BookmarkForgetIncludingRemotes(withRemotes...))
	}
	return m.runBookmarkCommands(commands)
}

// isBookmarkTracked reports whether the local bookmark tracks any remote.
func isBookmarkTracked(listing []jj.Bookmark, name string) bool {
	for _, b := range listing {
		if b.Name != name {
			continue
		}
		for _, r := range b.Remotes {
			if r.Tracked {
				return true
			}
		}
	}
	return false
}

// runOnFocusedCommit targets the commit id because the change id of a
// divergent revision is ambiguous.
func (m *Model) runOnFocusedCommit(kind string, command func(string) jj.CommandArgs) tea.Cmd {
	obj := m.focusedTextObject()
	revision := m.SelectedRevision()
	if obj == nil || obj.Kind != kind || revision == nil {
		return nil
	}
	return m.context.RunCommand(command(revision.CommitId), common.Refresh)
}

// showAllDivergentCopies limits the log to every copy of the change.
func (m *Model) showAllDivergentCopies() tea.Cmd {
	obj := m.focusedTextObject()
	revision := m.SelectedRevision()
	if obj == nil || obj.Kind != textObjectDivergent || revision == nil {
		return nil
	}
	changeId, _, _ := strings.Cut(revision.ChangeId, "/")
	return common.UpdateRevSet("change_id(" + changeId + ")")
}

// focusedWorkspaceRoot returns the focused workspace and its root directory.
func (m *Model) focusedWorkspaceRoot() (*textObject, string, tea.Cmd) {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectWorkspace {
		return nil, "", nil
	}
	output, err := m.context.RunCommandImmediate(jj.WorkspaceRoot(obj.Value))
	if err != nil {
		return nil, "", intents.Invoke(intents.AddMessage{Text: "Could not find workspace " + obj.Value, Err: err})
	}
	return obj, filepath.Clean(strings.TrimSpace(string(output))), nil
}

func (m *Model) isCurrentWorkspace(root string) bool {
	return root == filepath.Clean(m.context.Location)
}

func (m *Model) switchToFocusedWorkspace() tea.Cmd {
	obj, root, cmd := m.focusedWorkspaceRoot()
	if obj == nil {
		return cmd
	}
	if m.isCurrentWorkspace(root) {
		return intents.Invoke(intents.AddMessage{Text: "Already in workspace " + obj.Value})
	}
	if _, err := os.Stat(root); err != nil {
		return intents.Invoke(intents.AddMessage{Text: "Workspace " + obj.Value + " is missing on disk", Err: err})
	}
	m.context.ChangeWorkspace(root)
	return common.Refresh
}

func (m *Model) forgetFocusedWorkspace() tea.Cmd {
	obj, root, cmd := m.focusedWorkspaceRoot()
	if obj == nil {
		return cmd
	}
	if m.isCurrentWorkspace(root) {
		return intents.Invoke(intents.AddMessage{Text: "Switch to another workspace before forgetting " + obj.Value})
	}
	return m.context.RunCommand(jj.WorkspaceForget(obj.Value), common.Refresh)
}

func (m *Model) renameFocusedWorkspace() tea.Cmd {
	obj, root, cmd := m.focusedWorkspaceRoot()
	if obj == nil {
		return cmd
	}
	if !m.isCurrentWorkspace(root) {
		return intents.Invoke(intents.AddMessage{Text: "Only the current workspace can be renamed"})
	}
	return m.setBaseOperation(workspace.NewRenameWorkspaceOperation(m.context, m.SelectedRevision().GetChangeId(), obj.Value))
}
