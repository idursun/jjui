package revisions

import (
	"os"
	"path/filepath"
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

func (m *Model) moveFocusedBookmark() tea.Cmd {
	obj, cmd := m.focusedLocalBookmark("moved")
	if obj == nil {
		return cmd
	}
	return m.setBaseOperation(bookmark.NewMoveBookmarkOperation(m.context, obj.Value, m.SelectedRevision()))
}

// advanceFocusedBookmark moves the bookmark forward to jj's advance target,
// @ by default.
func (m *Model) advanceFocusedBookmark() tea.Cmd {
	obj, cmd := m.focusedLocalBookmark("advanced")
	if obj == nil {
		return cmd
	}
	return m.context.RunCommand(jj.BookmarkAdvance(obj.Value), common.Refresh)
}

// pushFocusedBookmark pushes the bookmark; on a remote bookmark it pushes to
// that remote, which also pushes a deletion marked with d.
func (m *Model) pushFocusedBookmark() tea.Cmd {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil
	}
	args := []string{"--bookmark", jj.ExactStringPattern(obj.Value)}
	if obj.Remote != "" {
		args = append(args, "--remote", obj.Remote)
	}
	return m.context.RunCommand(jj.GitPush(args...), common.Refresh)
}

// trackFocusedBookmark tracks or untracks a remote bookmark, or a local
// bookmark against the default remote.
func (m *Model) trackFocusedBookmark(track bool) tea.Cmd {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil
	}
	remote := obj.Remote
	if remote == "" {
		if remote = m.defaultTrackRemote(); remote == "" {
			return intents.Invoke(intents.AddMessage{Text: "There is no git remote to track " + obj.Value + " on"})
		}
	}
	symbol := obj.Value + "@" + remote

	tracked, err := m.isRemoteBookmarkTracked(obj.Value, remote)
	if err != nil {
		return intents.Invoke(intents.AddMessage{Text: "Could not list bookmarks", Err: err})
	}
	switch {
	case track && tracked:
		return intents.Invoke(intents.AddMessage{Text: symbol + " is already tracked"})
	case !track && !tracked:
		return intents.Invoke(intents.AddMessage{Text: symbol + " is not tracked"})
	case track:
		return m.context.RunCommand(jj.BookmarkTrack(obj.Value, remote), common.Refresh)
	default:
		return m.context.RunCommand(jj.BookmarkUntrack(obj.Value, remote), common.Refresh)
	}
}

// isRemoteBookmarkTracked looks up the tracking state with `jj bookmark list`
// because the log output does not show it.
func (m *Model) isRemoteBookmarkTracked(name string, remote string) (bool, error) {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return false, err
	}
	for _, b := range jj.ParseBookmarkListOutput(string(output)) {
		if b.Name != name {
			continue
		}
		for _, r := range b.Remotes {
			if r.Remote == remote {
				return r.Tracked, nil
			}
		}
	}
	return false, nil
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

// describeFocusedRevision opens the editor for the focused revision only,
// unlike describe, which applies to the checked revisions.
func (m *Model) describeFocusedRevision() tea.Cmd {
	obj := m.focusedTextObject()
	revision := m.SelectedRevision()
	if obj == nil || obj.Kind != textObjectDescription || revision == nil {
		return nil
	}
	return m.startDescribe(intents.Describe{Selected: jj.NewSelectedRevisions(revision)})
}

// forgetFocusedBookmark forgets the bookmark without marking a deletion to
// push. Remote and tracked bookmarks are forgotten with --include-remotes,
// otherwise their remote bookmarks stay behind; they come back on the next
// fetch if they still exist on the remote.
func (m *Model) forgetFocusedBookmark() tea.Cmd {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil
	}
	includeRemotes := obj.Remote != ""
	if !includeRemotes {
		tracked, err := m.isBookmarkTracked(obj.Value)
		if err != nil {
			return intents.Invoke(intents.AddMessage{Text: "Could not list bookmarks", Err: err})
		}
		includeRemotes = tracked
	}
	if includeRemotes {
		return m.context.RunCommand(jj.BookmarkForgetIncludingRemotes(obj.Value), common.Refresh)
	}
	return m.context.RunCommand(jj.BookmarkForget(obj.Value), common.Refresh)
}

// isBookmarkTracked reports whether the local bookmark tracks any remote.
func (m *Model) isBookmarkTracked(name string) (bool, error) {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return false, err
	}
	for _, b := range jj.ParseBookmarkListOutput(string(output)) {
		if b.Name != name {
			continue
		}
		for _, r := range b.Remotes {
			if r.Tracked {
				return true, nil
			}
		}
	}
	return false, nil
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
