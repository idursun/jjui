package bookmarkpane

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type remoteRef struct {
	Remote   string
	Tracked  bool
	Conflict bool
	CommitID string
}

type bookmarkRow struct {
	Name       string
	Local      *remoteRef
	Remotes    []remoteRef
	Conflict   bool
	Expanded   bool
	RemoteOnly bool
}

type visibleEntry struct {
	BookmarkIndex int
	RemoteIndex   int
	IsRemote      bool
}

func (e visibleEntry) IsLocal() bool {
	return !e.IsRemote
}

type rowsLoadedMsg struct {
	rows []bookmarkRow
}

type itemClickedMsg struct {
	index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m itemScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

type pendingInputKind int

const (
	pendingInputNone pendingInputKind = iota
	pendingInputRename
)

type filterState int

const (
	filterOff filterState = iota
	filterEditing
	filterApplied
)

type styles struct {
	title        lipgloss.Style
	text         lipgloss.Style
	dimmed       lipgloss.Style
	selected     lipgloss.Style
	localBadge   lipgloss.Style
	remoteBadge  lipgloss.Style
	trackedBadge lipgloss.Style
	conflict     lipgloss.Style
	filterPrompt lipgloss.Style
	childGuide   lipgloss.Style
}

type Model struct {
	context              *context.MainContext
	getCurrentRevision   func() *jj.Commit
	revealRevision       func(string) tea.Cmd
	visible              bool
	focused              bool
	rows                 []bookmarkRow
	visibleEntries       []visibleEntry
	cursor               int
	listRenderer         *render.ListRenderer
	ensureCursorVisible  bool
	filterInput          textinput.Model
	filterState          filterState
	filterText           string
	pendingInput         pendingInputKind
	pendingSelectionHint string
	styles               styles
}

var _ common.ImmediateModel = (*Model)(nil)
var _ common.Focusable = (*Model)(nil)
var _ common.Editable = (*Model)(nil)

func (m *Model) Init() tea.Cmd { return nil }

func NewModel(c *context.MainContext, currentRevision func() *jj.Commit, revealRevision func(string) tea.Cmd) *Model {
	palette := common.DefaultPalette
	s := styles{
		title:        palette.Get("title"),
		text:         palette.Get("picker text"),
		dimmed:       palette.Get("picker dimmed"),
		selected:     palette.Get("picker selected"),
		localBadge:   palette.Get("picker bookmark"),
		remoteBadge:  palette.Get("picker dimmed"),
		trackedBadge: palette.Get("status text"),
		conflict:     palette.Get("error"),
		filterPrompt: palette.Get("picker matched"),
		childGuide:   palette.Get("picker dimmed"),
	}

	filterInput := textinput.New()
	filterInput.Prompt = "Filter: "
	filterInput.Focus()
	fis := filterInput.Styles()
	fis.Focused.Prompt = s.filterPrompt
	fis.Focused.Text = s.text
	fis.Blurred.Prompt = s.filterPrompt
	fis.Blurred.Text = s.text
	filterInput.SetStyles(fis)

	m := &Model{
		context:            c,
		getCurrentRevision: currentRevision,
		revealRevision:     revealRevision,
		listRenderer:       render.NewListRenderer(itemScrollMsg{}),
		filterInput:        filterInput,
		styles:             s,
	}
	m.listRenderer.Z = render.ZMenuContent
	return m
}

func (m *Model) Visible() bool { return m.visible }

func (m *Model) IsFocused() bool { return m.focused }

func (m *Model) IsEditing() bool { return m.filterState == filterEditing }

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused && m.filterState == filterEditing {
		m.filterState = filterApplied
		m.filterText = strings.TrimSpace(m.filterInput.Value())
		m.filterInput.Blur()
	}
}

func (m *Model) Open() tea.Cmd {
	m.visible = true
	m.focused = true
	m.pendingSelectionHint = ""
	if current := m.getCurrentRevision(); current != nil {
		m.pendingSelectionHint = current.CommitId
	}
	return m.loadRows
}

func (m *Model) Close() {
	m.visible = false
	m.focused = false
	m.pendingInput = pendingInputNone
	if m.filterState == filterEditing {
		m.filterInput.Blur()
		m.filterState = filterApplied
		m.filterText = strings.TrimSpace(m.filterInput.Value())
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.visible && m.pendingInput == pendingInputNone {
		return nil
	}

	switch msg := msg.(type) {
	case rowsLoadedMsg:
		previousTarget, hadSelection := m.selectedTarget()
		m.rows = msg.rows
		m.applyFilters(false)
		switch {
		case m.pendingSelectionHint != "":
			m.selectTarget(m.pendingSelectionHint)
			m.pendingSelectionHint = ""
		case hadSelection:
			m.selectTarget(previousTarget)
		}
		return nil
	case itemClickedMsg:
		if msg.index >= 0 && msg.index < len(m.visibleEntries) {
			m.cursor = msg.index
			m.ensureCursorVisible = true
		}
		return nil
	case itemScrollMsg:
		if msg.Horizontal {
			return nil
		}
		m.ensureCursorVisible = false
		m.listRenderer.StartLine += msg.Delta
		if m.listRenderer.StartLine < 0 {
			m.listRenderer.StartLine = 0
		}
		return nil
	case input.SelectedMsg:
		if m.pendingInput == pendingInputRename {
			m.pendingInput = pendingInputNone
			value := strings.TrimSpace(msg.Value)
			selected, ok := m.selectedBookmark()
			if !ok || value == "" || value == selected.Name || selected.Local == nil {
				return nil
			}
			m.pendingSelectionHint = value
			return m.context.RunCommand(jj.BookmarkRename(selected.Name, value), common.Refresh, m.loadRows)
		}
	case input.CancelledMsg:
		m.pendingInput = pendingInputNone
		return nil
	case intents.Intent:
		return m.handleIntent(msg)
	case tea.KeyMsg:
		if m.filterState == filterEditing {
			updated, cmd := m.filterInput.Update(msg)
			changed := updated.Value() != m.filterInput.Value()
			m.filterInput = updated
			if changed {
				m.applyFilters(true)
			}
			return cmd
		}
	}
	return nil
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.BookmarkViewNavigate:
		if intent.IsPage {
			height := max(1, m.visibleHeight())
			m.ensureCursorVisible = false
			m.listRenderer.StartLine += intent.Delta * height
			if m.listRenderer.StartLine < 0 {
				m.listRenderer.StartLine = 0
			}
			return nil
		}
		m.moveCursor(intent.Delta)
		return nil
	case intents.BookmarkViewOpenFilter:
		m.filterState = filterEditing
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		return textinput.Blink
	case intents.BookmarkViewToggleExpand:
		m.toggleExpandSelected()
		return nil
	case intents.Apply:
		if m.filterState == filterEditing {
			m.filterState = filterApplied
			m.filterText = strings.TrimSpace(m.filterInput.Value())
			m.filterInput.Blur()
			m.applyFilters(true)
			return nil
		}
		return m.revealSelected()
	case intents.Cancel:
		if m.filterState == filterEditing {
			m.filterInput.SetValue("")
			m.filterText = ""
			m.filterState = filterOff
			m.filterInput.Blur()
			m.applyFilters(true)
			return nil
		}
		if m.currentFilterText() != "" {
			m.filterInput.SetValue("")
			m.filterText = ""
			m.filterState = filterOff
			m.applyFilters(true)
			return nil
		}
		m.Close()
		return nil
	case intents.BookmarkViewEdit:
		return m.editSelected()
	case intents.BookmarkViewNew:
		return m.newFromSelected()
	case intents.BookmarkViewRename:
		return m.renameSelected()
	case intents.BookmarkViewDelete:
		return m.deleteSelected()
	case intents.BookmarkViewForget:
		return m.forgetSelected()
	case intents.BookmarkViewTrack:
		return m.trackSelected()
	case intents.BookmarkViewUntrack:
		return m.untrackSelected()
	case intents.BookmarkViewMove:
		return m.moveSelected()
	case intents.BookmarkViewReveal:
		return m.revealSelected()
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if !m.visible || box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	content := box
	if content.R.Dx() <= 0 || content.R.Dy() <= 0 {
		return
	}

	titleBox, content := content.CutTop(1)
	m.renderTitle(dl, titleBox)
	_, content = content.CutTop(1)
	filterBox, listBox := content.CutTop(1)
	m.renderFilter(dl, filterBox)
	_, listBox = listBox.CutTop(1)
	m.renderList(dl, listBox)
}

func (m *Model) renderTitle(dl *render.DisplayContext, box layout.Box) {
	title := "Bookmarks"
	if current := m.getCurrentRevision(); current != nil {
		title = fmt.Sprintf("Bookmarks (%s)", current.GetChangeId())
	}
	dl.Text(box.R.Min.X, box.R.Min.Y, render.ZMenuContent).
		Styled(title, m.styles.title).
		Done()
}

func (m *Model) renderFilter(dl *render.DisplayContext, box layout.Box) {
	m.filterInput.SetWidth(max(box.R.Dx()-1, 0))
	if m.filterState == filterEditing {
		dl.AddDraw(box.R, m.filterInput.View(), render.ZMenuContent)
		return
	}
	filterText := m.currentFilterText()
	if filterText == "" {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZMenuContent).
			Styled("Filter: /", m.styles.dimmed).
			Done()
		return
	}
	dl.Text(box.R.Min.X, box.R.Min.Y, render.ZMenuContent).
		Styled("Filter: ", m.styles.filterPrompt).
		Styled(filterText, m.styles.text).
		Done()
}

func (m *Model) renderList(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	m.listRenderer.Render(
		dl,
		box,
		len(m.visibleEntries),
		m.cursor,
		m.ensureCursorVisible,
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			if index < 0 || index >= len(m.visibleEntries) {
				return
			}
			entry := m.visibleEntries[index]
			row := m.rows[entry.BookmarkIndex]
			if index == m.cursor && m.focused {
				dl.AddHighlight(rect, m.styles.selected, render.ZMenuContent+1)
			}
			tb := dl.Text(rect.Min.X, rect.Min.Y, render.ZMenuContent)
			if entry.IsRemote {
				remote := row.Remotes[entry.RemoteIndex]
				tb.Styled("  ", m.styles.text).
					Styled("└─ ", m.styles.childGuide).
					Styled(fmt.Sprintf(" %s ", remote.Remote), m.styles.remoteBadge).
					Styled(" ", m.styles.text).
					Styled(row.Name+"@"+remote.Remote, m.styles.text)
				if remote.Tracked {
					tb.Styled(" ", m.styles.text).Styled("tracked", m.styles.trackedBadge)
				}
				if remote.CommitID != "" {
					tb.Styled(" ", m.styles.text).Styled(remote.CommitID, m.styles.dimmed)
				}
				tb.Done()
				return
			}

			label := " local "
			style := m.styles.localBadge
			if row.RemoteOnly {
				label = " remote "
				style = m.styles.remoteBadge
			}
			prefix := "  "
			if len(row.Remotes) > 0 {
				if row.Expanded {
					prefix = "▾ "
				} else {
					prefix = "▸ "
				}
			}
			tb.Styled(prefix, m.styles.childGuide).
				Styled(label, style).
				Styled(" ", m.styles.text).
				Styled(row.Name, m.styles.text)
			if row.Local != nil && row.Local.Tracked {
				tb.Styled(" ", m.styles.text).Styled("tracked", m.styles.trackedBadge)
			}
			if row.Conflict {
				tb.Styled(" ", m.styles.text).Styled("conflict", m.styles.conflict)
			}
			commitID := row.commitID()
			if commitID != "" {
				tb.Styled(" ", m.styles.text).Styled(commitID, m.styles.dimmed)
			}
			tb.Done()
		},
		func(index int, _ tea.Mouse) tea.Msg { return itemClickedMsg{index: index} },
	)
	m.listRenderer.RegisterScroll(dl, box)
	m.ensureCursorVisible = false
}

func (m *Model) moveCursor(delta int) {
	if len(m.visibleEntries) == 0 {
		m.cursor = 0
		return
	}
	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.visibleEntries) {
		next = len(m.visibleEntries) - 1
	}
	if next != m.cursor {
		m.cursor = next
		m.ensureCursorVisible = true
	}
}

func (m *Model) currentFilterText() string {
	if m.filterState == filterEditing {
		return strings.TrimSpace(m.filterInput.Value())
	}
	return strings.TrimSpace(m.filterText)
}

func (m *Model) applyFilters(resetCursor bool) {
	filterText := strings.ToLower(m.currentFilterText())
	m.visibleEntries = m.visibleEntries[:0]
	for idx, row := range m.rows {
		if !m.bookmarkMatches(row, filterText) {
			continue
		}
		m.visibleEntries = append(m.visibleEntries, visibleEntry{BookmarkIndex: idx})
		if row.Expanded {
			for remoteIndex, remote := range row.Remotes {
				if filterText == "" || strings.Contains(strings.ToLower(row.Name+" "+remote.Remote+" "+row.Name+"@"+remote.Remote), filterText) {
					m.visibleEntries = append(m.visibleEntries, visibleEntry{
						BookmarkIndex: idx,
						RemoteIndex:   remoteIndex,
						IsRemote:      true,
					})
				}
			}
		}
	}
	if resetCursor || m.cursor >= len(m.visibleEntries) {
		m.cursor = 0
	}
	m.listRenderer.StartLine = 0
}

func (m *Model) bookmarkMatches(row bookmarkRow, filterText string) bool {
	if filterText == "" {
		return true
	}
	if strings.Contains(strings.ToLower(row.Name), filterText) {
		return true
	}
	for _, remote := range row.Remotes {
		if strings.Contains(strings.ToLower(remote.Remote), filterText) || strings.Contains(strings.ToLower(row.Name+"@"+remote.Remote), filterText) {
			return true
		}
	}
	return false
}

func (m *Model) loadRows() tea.Msg {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return rowsLoadedMsg{}
	}
	bookmarks := jj.ParseBookmarkListOutput(string(output))
	rows := make([]bookmarkRow, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		row := bookmarkRow{
			Name:       bookmark.Name,
			Conflict:   bookmark.Conflict,
			RemoteOnly: bookmark.Local == nil,
		}
		if bookmark.Local != nil {
			row.Local = &remoteRef{
				Remote:   ".",
				Tracked:  bookmark.Local.Tracked,
				Conflict: bookmark.Conflict,
				CommitID: bookmark.Local.CommitId,
			}
		}
		for _, remote := range bookmark.Remotes {
			row.Remotes = append(row.Remotes, remoteRef{
				Remote:   remote.Remote,
				Tracked:  remote.Tracked,
				Conflict: bookmark.Conflict,
				CommitID: remote.CommitId,
			})
		}
		rows = append(rows, row)
	}
	return rowsLoadedMsg{rows: rows}
}

func (m *Model) visibleHeight() int { return 8 }

func (m *Model) selectedEntry() (visibleEntry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visibleEntries) {
		return visibleEntry{}, false
	}
	return m.visibleEntries[m.cursor], true
}

func (m *Model) selectedBookmark() (bookmarkRow, bool) {
	entry, ok := m.selectedEntry()
	if !ok || entry.BookmarkIndex < 0 || entry.BookmarkIndex >= len(m.rows) {
		return bookmarkRow{}, false
	}
	return m.rows[entry.BookmarkIndex], true
}

func (m *Model) selectedTarget() (string, bool) {
	entry, ok := m.selectedEntry()
	if !ok {
		return "", false
	}
	row := m.rows[entry.BookmarkIndex]
	if entry.IsRemote {
		remote := row.Remotes[entry.RemoteIndex]
		return row.Name + "@" + remote.Remote, true
	}
	return row.Name, true
}

func (m *Model) selectedCommitID() string {
	entry, ok := m.selectedEntry()
	if !ok {
		return ""
	}
	row := m.rows[entry.BookmarkIndex]
	if entry.IsRemote {
		return row.Remotes[entry.RemoteIndex].CommitID
	}
	return row.commitID()
}

func (m *Model) selectTarget(target string) bool {
	for idx, entry := range m.visibleEntries {
		row := m.rows[entry.BookmarkIndex]
		if entry.IsRemote {
			remote := row.Remotes[entry.RemoteIndex]
			if strings.EqualFold(row.Name+"@"+remote.Remote, target) || strings.EqualFold(remote.CommitID, target) {
				m.cursor = idx
				m.ensureCursorVisible = true
				return true
			}
			continue
		}
		if strings.EqualFold(row.Name, target) || strings.EqualFold(row.commitID(), target) {
			m.cursor = idx
			m.ensureCursorVisible = true
			return true
		}
	}
	return false
}

func (m *Model) toggleExpandSelected() {
	entry, ok := m.selectedEntry()
	if !ok || entry.IsRemote {
		return
	}
	row := &m.rows[entry.BookmarkIndex]
	if len(row.Remotes) == 0 {
		return
	}
	target, _ := m.selectedTarget()
	row.Expanded = !row.Expanded
	m.applyFilters(false)
	m.selectTarget(target)
}

func (m *Model) revealSelected() tea.Cmd {
	commitID := m.selectedCommitID()
	target, _ := m.selectedTarget()
	if commitID == "" || m.revealRevision == nil {
		return nil
	}
	if cmd := m.revealRevision(commitID); cmd != nil {
		return cmd
	}
	return intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("Bookmark %s is not visible in the current revisions list", target)})
}

func (m *Model) editSelected() tea.Cmd {
	target, ok := m.selectedTarget()
	if !ok {
		return nil
	}
	return m.context.RunCommand(jj.Edit(target, false), common.Refresh, m.loadRows)
}

func (m *Model) newFromSelected() tea.Cmd {
	target, ok := m.selectedTarget()
	if !ok {
		return nil
	}
	return m.context.RunCommand(jj.New(jj.NewSelectedRevisions(&jj.Commit{ChangeId: target})), common.Refresh, m.loadRows)
}

func (m *Model) renameSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	entry, entryOK := m.selectedEntry()
	if !ok || !entryOK || entry.IsRemote || row.Local == nil {
		return nil
	}
	m.pendingInput = pendingInputRename
	return input.ShowWithTitle("Rename bookmark", "", row.Name)
}

func (m *Model) deleteSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	entry, entryOK := m.selectedEntry()
	if !ok || !entryOK || entry.IsRemote || row.Local == nil {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkDelete(row.Name), common.Refresh, m.loadRows)
}

func (m *Model) forgetSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	entry, entryOK := m.selectedEntry()
	if !ok || !entryOK || entry.IsRemote || row.Local == nil {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkForget(row.Name), common.Refresh, m.loadRows)
}

func (m *Model) trackSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	entry, entryOK := m.selectedEntry()
	if !ok || !entryOK {
		return nil
	}
	if entry.IsRemote {
		remote := row.Remotes[entry.RemoteIndex]
		if remote.Remote == "" || remote.Tracked {
			return nil
		}
		return m.context.RunCommand(jj.BookmarkTrack(row.Name, remote.Remote), common.Refresh, m.loadRows)
	}
	if row.Local == nil {
		return nil
	}
	remote := m.defaultTrackRemote()
	if remote == "" {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkTrack(row.Name, remote), common.Refresh, m.loadRows)
}

func (m *Model) untrackSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	entry, entryOK := m.selectedEntry()
	if !ok || !entryOK || !entry.IsRemote {
		return nil
	}
	remote := row.Remotes[entry.RemoteIndex]
	if remote.Remote == "" || !remote.Tracked {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkUntrack(row.Name, remote.Remote), common.Refresh, m.loadRows)
}

func (m *Model) moveSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	entry, entryOK := m.selectedEntry()
	current := m.getCurrentRevision()
	if !ok || !entryOK || entry.IsRemote || row.Local == nil || current == nil {
		return nil
	}
	m.pendingSelectionHint = row.Name
	return m.context.RunCommand(jj.BookmarkMove(current.GetChangeId(), row.Name, "--allow-backwards"), common.Refresh, m.loadRows)
}

func (m *Model) defaultTrackRemote() string {
	output, err := m.context.RunCommandImmediate(jj.GitRemoteList())
	if err != nil {
		return ""
	}
	remotes := jj.ParseRemoteListOutput(string(output))
	for _, remote := range remotes {
		if remote == "origin" {
			return remote
		}
	}
	if len(remotes) > 0 {
		return remotes[0]
	}
	return ""
}

func (r bookmarkRow) commitID() string {
	if r.Local != nil {
		return r.Local.CommitID
	}
	if len(r.Remotes) > 0 {
		return r.Remotes[0].CommitID
	}
	return ""
}
