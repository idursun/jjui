package revset

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

const (
	remoteAll   = "all"
	remoteLocal = "local"
)

var (
	scopeBookmarks        = bindings.ScopeName(actions.ScopeRevset + ".bookmarks")
	scopeBookmarksFilter  = bindings.ScopeName(actions.ScopeRevset + ".bookmarks.filter")
	scopeBookmarksRemotes = bindings.ScopeName(actions.ScopeRevset + ".bookmarks.remotes")
	scopeRemotesEditor    = bindings.ScopeName(actions.ScopeRevset + ".bookmarks.remotes.editor")
	scopeRemotesConfirm   = bindings.ScopeName(actions.ScopeRevset + ".bookmarks.remotes.confirmation")
)

// bookmarksView is a revset built from a remote and a name filter. It lasts
// until a revset it did not build is set.
type bookmarksView struct {
	// issued holds every revset the view built, because a fast typist gets
	// ahead of the echoes of the earlier ones.
	issued       map[string]bool
	remotes      []string // all, local, then the git remotes
	remoteIdx    int
	filter       textinput.Model
	filtering    bool
	focused      bool // the cursor is on the remotes line
	renaming     bool // the focused remote's name is being edited
	rename       textinput.Model
	confirm      *confirmation.Model // asks before removing the focused remote
	filterBefore string
	revset       string
	restore      string
}

type selectRemoteMsg struct {
	index int
}

// remotesChangedMsg reloads the remotes, keeping the cursor on focus when it
// is still there.
type remotesChangedMsg struct {
	focus string
}

type removeRemoteMsg struct {
	name string
}

func remotesChanged(focus string) tea.Cmd {
	return func() tea.Msg { return remotesChangedMsg{focus: focus} }
}

// inputScope is the scope of the input that holds the keys, if one is open.
func (v *bookmarksView) inputScope() (bindings.ScopeName, bool) {
	switch {
	case v.filtering:
		return scopeBookmarksFilter, true
	case v.renaming:
		return scopeRemotesEditor, true
	case v.confirm != nil:
		return scopeRemotesConfirm, true
	}
	return "", false
}

// gitRemote returns the focused remote unless it is all or local.
func (v *bookmarksView) gitRemote() (string, bool) {
	remote := v.remotes[v.remoteIdx]
	return remote, v.focused && remote != remoteLocal && remote != remoteAll
}

func bookmarksRevset(filter, remote string) string {
	var args []string
	if filter != "" {
		args = append(args, "substring-i:"+quoteRevsetString(filter))
	}
	switch remote {
	case remoteAll:
		a := strings.Join(args, ", ")
		return "bookmarks(" + a + ") | remote_bookmarks(" + a + ")"
	case remoteLocal:
		return "bookmarks(" + strings.Join(args, ", ") + ")"
	default:
		args = append(args, "remote=exact:"+quoteRevsetString(remote))
		return "remote_bookmarks(" + strings.Join(args, ", ") + ")"
	}
}

func quoteRevsetString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func (m *Model) InBookmarksView() bool {
	return m.bookmarks != nil
}

func (m *Model) RemotesFocused() bool {
	return m.bookmarks != nil && m.bookmarks.focused
}

// FocusedRemote returns the preview object kind and value of the focused
// entry on the remotes line.
func (m *Model) FocusedRemote() (kind, value string, ok bool) {
	if !m.RemotesFocused() {
		return "", "", false
	}
	switch remote := m.bookmarks.remotes[m.bookmarks.remoteIdx]; remote {
	case remoteAll:
		return "remote_all", remote, true
	case remoteLocal:
		return "remote_local", remote, true
	default:
		return "remote", jj.ExactStringPattern(remote), true
	}
}

// BookmarksScopes are the keys of the bookmarks view outside the filter input.
// They must sit before the revisions scope so that tab, / and esc reach them.
func (m *Model) BookmarksScopes() []common.Scope {
	if m.bookmarks == nil {
		return nil
	}
	if _, ok := m.bookmarks.inputScope(); ok {
		return nil
	}
	if m.bookmarks.focused {
		// LeakGlobal keeps the log's keys away from the remotes line.
		return []common.Scope{
			{Name: scopeBookmarksRemotes, Leak: common.LeakAll, Handler: m},
			{Name: scopeBookmarks, Leak: common.LeakGlobal, Handler: m},
		}
	}
	return []common.Scope{{Name: scopeBookmarks, Leak: common.LeakAll, Handler: m}}
}

// IsStaleRevset reports a revset the bookmarks view issued and has moved on
// from. Its commands run concurrently, so a quick second change can arrive
// before the first.
func (m *Model) IsStaleRevset(revset string) bool {
	v := m.bookmarks
	return v != nil && v.issued[revset] && revset != v.revset
}

func (m *Model) QueryState(name string) (any, bool) {
	switch name {
	case "bookmarks.can_fetch":
		v := m.bookmarks
		return v != nil && v.focused && v.remotes[v.remoteIdx] != remoteLocal, true
	case "bookmarks.can_push", "bookmarks.can_edit_remote":
		if m.bookmarks == nil {
			return false, true
		}
		_, ok := m.bookmarks.gitRemote()
		return ok, true
	}
	return nil, false
}

func (m *Model) showBookmarks() tea.Cmd {
	restore := m.context.CurrentRevset
	if m.bookmarks != nil {
		restore = m.bookmarks.restore
	}
	m.bookmarks = &bookmarksView{remotes: m.loadRemotes(), filter: newBookmarksInput(""), restore: restore, issued: map[string]bool{}}
	return m.applyBookmarks()
}

func newBookmarksInput(value string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetVirtualCursor(false)
	ti.SetValue(value)
	ti.CursorEnd()
	return ti
}

func (m *Model) loadRemotes() []string {
	remotes := []string{remoteAll, remoteLocal}
	if out, err := m.context.RunCommandImmediate(jj.GitRemoteList()); err == nil {
		remotes = append(remotes, jj.ParseRemoteListOutput(string(out))...)
	}
	return remotes
}

// reloadRemotes keeps the cursor on focus, or on the same place when focus
// is gone.
func (m *Model) reloadRemotes(focus string) {
	v := m.bookmarks
	v.remotes = m.loadRemotes()
	v.remoteIdx = min(v.remoteIdx, len(v.remotes)-1)
	if i := slices.Index(v.remotes, focus); i >= 0 {
		v.remoteIdx = i
	}
}

func (m *Model) handleRemoteInputIntent(intent intents.Intent) (tea.Cmd, bool) {
	v := m.bookmarks
	if v.confirm != nil {
		switch intent.(type) {
		case intents.Apply, intents.Cancel, intents.OptionSelect:
			return v.confirm.Update(intent), true
		}
		return nil, false
	}
	switch intent.(type) {
	case intents.Cancel:
		v.renaming = false
		return nil, true
	case intents.Apply:
		v.renaming = false
		oldName := v.remotes[v.remoteIdx]
		newName := strings.TrimSpace(v.rename.Value())
		if newName == "" || newName == oldName {
			return nil, true
		}
		return m.context.RunCommand(jj.GitRemoteRename(oldName, newName), remotesChanged(newName)), true
	}
	return nil, false
}

func (m *Model) applyBookmarks() tea.Cmd {
	v := m.bookmarks
	v.revset = bookmarksRevset(strings.TrimSpace(v.filter.Value()), v.remotes[v.remoteIdx])
	v.issued[v.revset] = true
	return common.UpdateRevSet(v.revset)
}

func (m *Model) handleBookmarksIntent(intent intents.Intent) (tea.Cmd, bool) {
	if intent, ok := intent.(intents.RevsetBookmarks); ok {
		cmd := m.showBookmarks()
		m.bookmarks.focused = intent.FocusRemotes
		return cmd, true
	}
	v := m.bookmarks
	if v == nil {
		return nil, false
	}
	if v.renaming || v.confirm != nil {
		return m.handleRemoteInputIntent(intent)
	}
	switch intent := intent.(type) {
	case intents.RevsetBookmarksRemotesRename:
		remote, ok := v.gitRemote()
		if !ok {
			return nil, true
		}
		v.renaming = true
		v.rename = newBookmarksInput(remote)
		return v.rename.Focus(), true
	case intents.RevsetBookmarksRemotesRemove:
		remote, ok := v.gitRemote()
		if !ok {
			return nil, true
		}
		v.confirm = confirmation.New(
			[]string{"Remove remote " + remote + " and forget its bookmarks?"},
			confirmation.WithZIndex(render.ZDialogs),
			confirmation.WithOption("Yes", func() tea.Msg { return removeRemoteMsg{name: remote} }, key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
			confirmation.WithOption("No", confirmation.Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		)
		return v.confirm.Init(), true
	case intents.RevsetBookmarksCycleRemote:
		v.remoteIdx = (v.remoteIdx + intent.Delta + len(v.remotes)) % len(v.remotes)
		return m.applyBookmarks(), true
	case intents.RevsetBookmarksFocusRemotes:
		v.focused = true
		return nil, true
	case intents.RevsetBookmarksRemotesLeave:
		v.focused = false
		return nil, true
	case intents.RevsetBookmarksRemotesFetch:
		var flags []string
		if intent.Tracked {
			flags = append(flags, "--tracked")
		}
		switch remote := v.remotes[v.remoteIdx]; remote {
		case remoteLocal:
			return nil, true
		case remoteAll:
			flags = append(flags, "--all-remotes")
		default:
			flags = append(flags, "--remote", remote)
		}
		return m.context.RunCommand(jj.GitFetch(flags...), common.Refresh), true
	case intents.RevsetBookmarksRemotesPush:
		remote := v.remotes[v.remoteIdx]
		if remote == remoteLocal || remote == remoteAll {
			return nil, true
		}
		var flags []string
		if intent.All {
			flags = append(flags, "--all")
		}
		if intent.Tracked {
			flags = append(flags, "--tracked")
		}
		if intent.Deleted {
			flags = append(flags, "--deleted")
		}
		flags = append(flags, "--remote", remote)
		return m.context.RunCommand(jj.GitPush(flags...), common.Refresh), true
	case intents.RevsetBookmarksFilter:
		v.filtering = true
		v.filterBefore = v.filter.Value()
		v.filter.CursorEnd()
		return v.filter.Focus(), true
	case intents.RevsetBookmarksFilterAccept:
		v.filtering = false
		v.filter.Blur()
		return nil, true
	case intents.RevsetBookmarksFilterCancel:
		v.filtering = false
		v.filter.Blur()
		if v.filter.Value() == v.filterBefore {
			return nil, true
		}
		v.filter.SetValue(v.filterBefore)
		return m.applyBookmarks(), true
	case intents.RevsetBookmarksExit:
		m.bookmarks = nil
		return common.UpdateRevSet(v.restore), true
	}
	return nil, false
}

// updateBookmarks handles the messages the bookmarks view cares about while
// it is open. It reports false for messages it leaves to the revset editor.
func (m *Model) updateBookmarks(msg tea.Msg) (tea.Cmd, bool) {
	v := m.bookmarks
	switch msg := msg.(type) {
	case common.UpdateRevSetMsg:
		if !v.issued[string(msg)] {
			m.bookmarks = nil
			return nil, false
		}
		return nil, true
	case selectRemoteMsg:
		if v.renaming || v.confirm != nil {
			return nil, true
		}
		v.remoteIdx = msg.index
		return m.applyBookmarks(), true
	case remotesChangedMsg:
		m.reloadRemotes(msg.focus)
		return m.applyBookmarks(), true
	case removeRemoteMsg:
		v.confirm = nil
		return m.context.RunCommand(jj.GitRemoteRemove(msg.name), remotesChanged("")), true
	case confirmation.CloseMsg:
		if v.confirm == nil {
			return nil, false
		}
		v.confirm = nil
		return nil, true
	case confirmation.SelectOptionMsg:
		if v.confirm == nil {
			return nil, false
		}
		return v.confirm.Update(msg), true
	case tea.KeyMsg:
		if v.confirm != nil {
			return v.confirm.Update(msg), true
		}
		if v.renaming {
			var cmd tea.Cmd
			v.rename, cmd = v.rename.Update(msg)
			return cmd, true
		}
		if !v.filtering {
			return nil, false
		}
		before := v.filter.Value()
		var cmd tea.Cmd
		v.filter, cmd = v.filter.Update(msg)
		if v.filter.Value() == before {
			return cmd, true
		}
		return tea.Batch(cmd, m.applyBookmarks()), true
	}
	return nil, false
}

// viewBookmarks draws the remotes and the filter that built the revset above.
func (m *Model) viewBookmarks(dl *render.DisplayContext, box layout.Box) {
	v := m.bookmarks
	titleStyle := theme.DefaultPalette.Get("revset", "", "title", false)
	textStyle := theme.DefaultPalette.Get("revset", "", "text", false)
	dimmed := theme.DefaultPalette.Get("revset", "completion", "dimmed", false)

	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZFuzzyInput)
	tb.Styled("remote: ", titleStyle)
	for i, remote := range v.remotes {
		if i == v.remoteIdx && v.renaming {
			x, _ := tb.Measure()
			tb.Write(" " + v.rename.View() + " ")
			dl.SetCursorInRect(v.rename.Cursor(), box.R, x+1, 0)
			continue
		}
		style := dimmed
		if i == v.remoteIdx {
			style = textStyle
			if v.focused {
				x, _ := tb.Measure()
				style = style.Inherit(theme.DefaultPalette.Get("revisions", "", "object", true))
				// Set before the log draws, so its passive cursor gives way.
				dl.SetCursorAt(remoteCursor(), box.R.Min.X+x+1, box.R.Min.Y)
			}
		}
		tb.Clickable(" "+remote+" ", style, selectRemoteMsg{index: i})
	}
	tb.Styled("   filter: ", titleStyle)
	cursorX, _ := tb.Measure()
	if v.filtering {
		tb.Write(v.filter.View())
		dl.SetCursorInRect(v.filter.Cursor(), box.R, cursorX, 0)
	} else {
		tb.Styled(v.filter.Value(), textStyle)
	}
	tb.Done()

	if v.confirm != nil {
		w, h := lipgloss.Size(v.confirm.View())
		v.confirm.ViewRect(dl, layout.NewBox(layout.Rect(box.R.Min.X, box.R.Max.Y, w, h)))
	}
}

// remoteCursor matches the cursor of a focused text object in the log.
func remoteCursor() *tea.Cursor {
	cursor := tea.NewCursor(0, 0)
	cursor.Shape = tea.CursorBlock
	cursor.Blink = true
	return cursor
}
