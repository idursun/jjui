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

func (m *Model) pushFocusedBookmark() tea.Cmd {
	obj, cmd := m.focusedLocalBookmark("pushed")
	if obj == nil {
		return cmd
	}
	return m.context.RunCommand(jj.GitPush("--bookmark", jj.ExactStringPattern(obj.Value)), common.Refresh)
}

// trackFocusedBookmark looks up the tracking state with `jj bookmark list`
// because the log output does not show it.
func (m *Model) trackFocusedBookmark(track bool) tea.Cmd {
	obj := m.focusedTextObject()
	if obj == nil || obj.Kind != textObjectBookmark {
		return nil
	}
	if obj.Remote == "" {
		return intents.Invoke(intents.AddMessage{Text: "Focus a remote bookmark to track or untrack it"})
	}
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return intents.Invoke(intents.AddMessage{Text: "Could not list bookmarks", Err: err})
	}
	for _, b := range jj.ParseBookmarkListOutput(string(output)) {
		if b.Name != obj.Value {
			continue
		}
		for _, remote := range b.Remotes {
			if remote.Remote != obj.Remote {
				continue
			}
			switch {
			case track && remote.Tracked:
				return intents.Invoke(intents.AddMessage{Text: obj.Value + "@" + obj.Remote + " is already tracked"})
			case !track && !remote.Tracked:
				return intents.Invoke(intents.AddMessage{Text: obj.Value + "@" + obj.Remote + " is not tracked"})
			case track:
				return m.context.RunCommand(jj.BookmarkTrack(obj.Value, obj.Remote), common.Refresh)
			default:
				return m.context.RunCommand(jj.BookmarkUntrack(obj.Value, obj.Remote), common.Refresh)
			}
		}
	}
	return intents.Invoke(intents.AddMessage{Text: "Bookmark " + obj.Value + "@" + obj.Remote + " not found"})
}
