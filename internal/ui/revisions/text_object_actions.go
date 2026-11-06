package revisions

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
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
