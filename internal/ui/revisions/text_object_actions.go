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

func (m *Model) renameFocusedBookmark() tea.Cmd {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil
	}
	if obj.Remote != "" {
		return intents.Invoke(intents.AddMessage{Text: "Only local bookmarks can be renamed"})
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
	// Leaving the bookmarks view brings back the revset to pick a target from.
	return tea.Batch(
		m.setBaseOperation(bookmark.NewMoveBookmarkOperation(m.context, names, m.SelectedRevision())),
		intents.Invoke(intents.RevsetBookmarksExit{}),
	)
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
		// Bindings query this on every render, so skip parsing rows without bookmarks.
		if row.Commit == nil || !row.Commit.HasBookmarks {
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

// trackBookmarks tracks or untracks remote bookmarks, or local bookmarks
// against the default remote. Bookmarks already in that state are skipped.
func (m *Model) trackBookmarks(track bool) tea.Cmd {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return nil
	}
	defaultRemote := ""
	if slices.ContainsFunc(targets, func(o textObject) bool { return o.Remote == "" }) {
		// The remote list puts git.default_remote first.
		if output, err := m.context.RunCommandImmediate(jj.GitRemoteList()); err == nil {
			if remotes := jj.ParseRemoteListOutput(string(output)); len(remotes) > 0 {
				defaultRemote = remotes[0]
			}
		}
		if defaultRemote == "" {
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

type bookmarkTrackingState struct {
	listing       []jj.Bookmark
	defaultRemote string
}

type bookmarkTrackingStateMsg struct {
	state *bookmarkTrackingState
	tag   uint64
}

// loadTrackingState lists bookmarks alongside the log because the log output
// does not show the tracking state. A failed listing keeps the previous state.
func (m *Model) loadTrackingState(tag uint64) tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
		if err != nil {
			return nil
		}
		state := &bookmarkTrackingState{listing: jj.ParseBookmarkListOutput(string(output))}
		// The remote list puts git.default_remote first.
		if output, err := m.context.RunCommandImmediate(jj.GitRemoteList()); err == nil {
			if remotes := jj.ParseRemoteListOutput(string(output)); len(remotes) > 0 {
				state.defaultRemote = remotes[0]
			}
		}
		return bookmarkTrackingStateMsg{state: state, tag: tag}
	}
}

// canChangeBookmarkTracking reports whether tracking (or untracking) applies
// to any bookmark target. Both apply until the state is known; neither applies
// to a local bookmark when there is no remote.
func (m *Model) canChangeBookmarkTracking(track bool) bool {
	targets := m.bookmarkTargets()
	if len(targets) == 0 {
		return false
	}
	state := m.trackingState
	if state == nil {
		return true
	}
	return slices.ContainsFunc(targets, func(target textObject) bool {
		remote := target.Remote
		if remote == "" {
			remote = state.defaultRemote
		}
		return remote != "" && isRemoteBookmarkTracked(state.listing, target.Value, remote) != track
	})
}

// bookmarkRemotes returns the remotes of the named bookmark, or nil while the
// tracking state is unknown.
func (m *Model) bookmarkRemotes(name string) []jj.BookmarkRemote {
	if m.trackingState == nil {
		return nil
	}
	for _, b := range m.trackingState.listing {
		if b.Name == name {
			return append([]jj.BookmarkRemote{}, b.Remotes...)
		}
	}
	return []jj.BookmarkRemote{}
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

// widenedRevset unions the current revset with extra, or explains why it
// cannot.
func (m *Model) widenedRevset(extra func(current string) string) tea.Cmd {
	current := strings.TrimSpace(m.context.CurrentRevset)
	if current == "" {
		return intents.Invoke(intents.AddMessage{Text: "The revset cannot be widened without an explicit revset"})
	}
	current = "(" + current + ")"
	return common.UpdateRevSet(current + " | " + extra(current))
}

// showElidedBelow adds the revisions hidden between revision and its nearest
// shown ancestors.
func (m *Model) showElidedBelow(revision *jj.Commit) tea.Cmd {
	return m.widenedRevset(func(current string) string {
		return "heads(" + current + " & ::" + revision.CommitId + "-)::" + revision.CommitId
	})
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

func (m *Model) renameFocusedWorkspace() tea.Cmd {
	obj, root, cmd := m.focusedWorkspaceRoot()
	if obj == nil {
		return cmd
	}
	if _, err := os.Stat(root); err != nil {
		return intents.Invoke(intents.AddMessage{Text: "Workspace " + obj.Value + " is missing on disk", Err: err})
	}
	return m.setBaseOperation(workspace.NewRenameWorkspaceOperation(m.context, m.SelectedRevision().GetChangeId(), root, obj.Value))
}

func (m *Model) removeFocusedWorkspace() tea.Cmd {
	obj, root, cmd := m.focusedWorkspaceRoot()
	if obj == nil {
		return cmd
	}
	current, err := m.context.RunCommandImmediate(jj.CurrentWorkspaceRoot())
	if err != nil {
		return intents.Invoke(intents.AddMessage{Text: "Could not find the current workspace", Err: err})
	}
	if root == filepath.Clean(strings.TrimSpace(string(current))) {
		return intents.Invoke(intents.AddMessage{Text: "Switch to another workspace before removing " + obj.Value})
	}
	return m.setBaseOperation(workspace.NewRemoveWorkspaceOperation(m.context, m.SelectedRevision().GetChangeId(), root, obj.Value))
}
