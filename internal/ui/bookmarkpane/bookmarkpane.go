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

type Callbacks struct {
	CurrentRevision   func() *jj.Commit
	VisibleCommitIDs  func() []string
	RevealVisible     func(string) tea.Cmd
	ShowInRevisions   func(target, commitID string) tea.Cmd
	FocusRevisions    func() tea.Cmd
	BeginMoveBookmark func(name string) tea.Cmd
}

type rowsLoadedMsg struct {
	tree bookmarkTree
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
	title           lipgloss.Style
	text            lipgloss.Style
	dimmed          lipgloss.Style
	selected        lipgloss.Style
	localBadge      lipgloss.Style
	remoteBadge     lipgloss.Style
	remoteNameBadge lipgloss.Style
	trackedBadge    lipgloss.Style
	conflict        lipgloss.Style
	filterPrompt    lipgloss.Style
	childGuide      lipgloss.Style
}

type Model struct {
	context              *context.MainContext
	callbacks            Callbacks
	visible              bool
	focused              bool
	tree                 bookmarkTree
	visibleRows          []visibleRow
	expanded             map[string]bool
	cursor               int
	listRenderer         *render.ListRenderer
	ensureCursorVisible  bool
	filterInput          textinput.Model
	filterState          filterState
	filterText           string
	pendingInput         pendingInputKind
	pendingSelectionHint string
	resetSelectionOnLoad bool
	styles               styles
}

var (
	_ common.ImmediateModel = (*Model)(nil)
	_ common.Focusable      = (*Model)(nil)
	_ common.Editable       = (*Model)(nil)
)

func (m *Model) Init() tea.Cmd { return nil }

func NewModel(c *context.MainContext, callbacks Callbacks) *Model {
	palette := common.DefaultPalette
	s := styles{
		title:           palette.Get("title"),
		text:            palette.Get("picker text"),
		dimmed:          palette.Get("picker dimmed"),
		selected:        palette.Get("revisions selected"),
		localBadge:      palette.Get("picker bookmark"),
		remoteBadge:     palette.Get("picker dimmed"),
		remoteNameBadge: palette.Get("picker matched"),
		trackedBadge:    palette.Get("status text"),
		conflict:        palette.Get("error"),
		filterPrompt:    palette.Get("picker matched"),
		childGuide:      palette.Get("picker dimmed"),
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
		context:      c,
		callbacks:    callbacks,
		expanded:     make(map[string]bool),
		listRenderer: render.NewListRenderer(itemScrollMsg{}),
		filterInput:  filterInput,
		styles:       s,
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
	m.resetSelectionOnLoad = true
	m.cursor = 0
	m.ensureCursorVisible = true
	m.listRenderer.StartLine = 0
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
		m.tree = msg.tree
		m.applyFilters(m.resetSelectionOnLoad)
		switch {
		case m.pendingSelectionHint != "":
			m.selectTarget(m.pendingSelectionHint)
			m.pendingSelectionHint = ""
		case hadSelection && !m.resetSelectionOnLoad:
			m.selectTarget(previousTarget)
		}
		m.resetSelectionOnLoad = false
		return nil
	case itemClickedMsg:
		if msg.index >= 0 && msg.index < len(m.visibleRows) {
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
	case common.RefreshMsg:
		if m.visible {
			return m.loadRows
		}
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
	case intents.BookmarkViewRevealInRevisions:
		return m.showSelectedInRevisions()
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if !m.visible || box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	dl.AddFill(box.R, ' ', m.styles.text, render.ZMenuContent)

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
	dl.Text(box.R.Min.X, box.R.Min.Y, render.ZMenuContent).
		Styled("Bookmarks", m.styles.title).
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
		len(m.visibleRows),
		m.cursor,
		m.ensureCursorVisible,
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			if index < 0 || index >= len(m.visibleRows) {
				return
			}
			row := m.visibleRows[index]
			group := m.tree.Items[row.BookmarkIndex]
			if index == m.cursor && m.focused {
				dl.AddHighlight(rect, m.styles.selected, render.ZMenuContent+1)
			}
			tb := dl.Text(rect.Min.X, rect.Min.Y, render.ZMenuContent)
			if row.Depth > 0 {
				tb.Styled("  ", m.styles.text).
					Styled("└─ ", m.styles.childGuide).
					Styled(fmt.Sprintf(" %s ", row.Node.Remote), m.styles.remoteNameBadge).
					Styled(" ", m.styles.text).
					Styled(row.Node.Target(), m.styles.text)
				if row.Node.Tracked {
					tb.Styled(" ", m.styles.text).Styled("tracked", m.styles.trackedBadge)
				}
				if row.Node.Conflict {
					tb.Styled(" ", m.styles.text).Styled("conflict", m.styles.conflict)
				}
				if row.Node.CommitID != "" {
					tb.Styled(" ", m.styles.text).Styled(row.Node.CommitID, m.styles.dimmed)
				}
				tb.Done()
				return
			}

			label := " local "
			style := m.styles.localBadge
			if group.RemoteOnly {
				label = " remote "
				style = m.styles.remoteBadge
			}
			prefix := "  "
			if row.HasChildren {
				if row.Expanded {
					prefix = "▾ "
				} else {
					prefix = "▸ "
				}
			}
			tb.Styled(prefix, m.styles.childGuide).
				Styled(label, style).
				Styled(" ", m.styles.text).
				Styled(row.Node.Name, m.styles.text)
			for i, remote := range group.Remotes {
				separator := " "
				if i == 0 {
					separator = "  "
				}
				tb.Styled(separator, m.styles.text).Styled(remote.Remote, m.styles.remoteNameBadge)
			}
			if row.Node.Tracked {
				tb.Styled(" ", m.styles.text).Styled("tracked", m.styles.trackedBadge)
			}
			if row.Node.Conflict {
				tb.Styled(" ", m.styles.text).Styled("conflict", m.styles.conflict)
			}
			if row.Node.CommitID != "" {
				tb.Styled(" ", m.styles.text).Styled(row.Node.CommitID, m.styles.dimmed)
			}
			tb.Done()
		},
		func(index int, _ tea.Mouse) tea.Msg { return itemClickedMsg{index: index} },
	)
	m.listRenderer.RegisterScroll(dl, box)
	m.ensureCursorVisible = false
}

func (m *Model) moveCursor(delta int) {
	if len(m.visibleRows) == 0 {
		m.cursor = 0
		return
	}
	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.visibleRows) {
		next = len(m.visibleRows) - 1
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
	m.visibleRows = m.tree.buildVisibleRows(m.currentFilterText())
	if resetCursor || m.cursor >= len(m.visibleRows) {
		m.cursor = 0
	}
	m.listRenderer.StartLine = 0
}

func (m *Model) loadRows() tea.Msg {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return rowsLoadedMsg{}
	}

	var currentCommitID string
	if m.callbacks.CurrentRevision != nil {
		if current := m.callbacks.CurrentRevision(); current != nil {
			currentCommitID = current.CommitId
		}
	}

	var visibleCommitIDs []string
	if m.callbacks.VisibleCommitIDs != nil {
		visibleCommitIDs = m.callbacks.VisibleCommitIDs()
	}

	return rowsLoadedMsg{tree: loadBookmarkTree(string(output), m.expanded, currentCommitID, visibleCommitIDs)}
}

func (m *Model) visibleHeight() int { return 8 }

func (m *Model) selectedRow() (visibleRow, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visibleRows) {
		return visibleRow{}, false
	}
	return m.visibleRows[m.cursor], true
}

func (m *Model) selectedBookmark() (bookmarkTreeItem, bool) {
	row, ok := m.selectedRow()
	if !ok || row.BookmarkIndex < 0 || row.BookmarkIndex >= len(m.tree.Items) {
		return bookmarkTreeItem{}, false
	}
	return m.tree.Items[row.BookmarkIndex], true
}

func (m *Model) selectedNode() (bookmarkRefNode, bool) {
	row, ok := m.selectedRow()
	if !ok {
		return bookmarkRefNode{}, false
	}
	return row.Node, true
}

func (m *Model) selectedTarget() (string, bool) {
	node, ok := m.selectedNode()
	if !ok {
		return "", false
	}
	return node.Target(), true
}

func (m *Model) selectedCommitID() string {
	node, ok := m.selectedNode()
	if ok {
		return node.CommitID
	}
	return ""
}

func (m *Model) selectTarget(target string) bool {
	for idx, row := range m.visibleRows {
		if strings.EqualFold(row.Node.Target(), target) || strings.EqualFold(row.Node.CommitID, target) {
			m.cursor = idx
			m.ensureCursorVisible = true
			return true
		}
	}
	return false
}

func (m *Model) toggleExpandSelected() {
	row, ok := m.selectedRow()
	if !ok || row.Depth > 0 {
		return
	}
	group := &m.tree.Items[row.BookmarkIndex]
	if len(group.Remotes) == 0 {
		return
	}
	target, _ := m.selectedTarget()
	group.Expanded = !group.Expanded
	m.expanded[group.Name] = group.Expanded
	m.applyFilters(false)
	m.selectTarget(target)
}

func (m *Model) revealSelected() tea.Cmd {
	commitID := m.selectedCommitID()
	target, _ := m.selectedTarget()
	if commitID == "" || m.callbacks.RevealVisible == nil {
		return nil
	}
	if m.callbacks.CurrentRevision != nil {
		if current := m.callbacks.CurrentRevision(); current != nil && current.CommitId == commitID {
			return intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("Already at bookmark %s", target)})
		}
	}
	if cmd := m.callbacks.RevealVisible(commitID); cmd != nil {
		if m.callbacks.FocusRevisions != nil {
			return tea.Batch(cmd, m.callbacks.FocusRevisions())
		}
		return cmd
	}
	return intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("Bookmark %s is not visible in the current revisions list", target)})
}

func (m *Model) showSelectedInRevisions() tea.Cmd {
	target, ok := m.selectedTarget()
	if !ok || m.callbacks.ShowInRevisions == nil {
		return nil
	}
	return m.callbacks.ShowInRevisions(target, m.selectedCommitID())
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
	selected, selectedOK := m.selectedNode()
	if !ok || !selectedOK || selected.IsRemote() || row.Local == nil {
		return nil
	}
	m.pendingInput = pendingInputRename
	return input.ShowWithTitle("Rename bookmark", "", row.Name)
}

func (m *Model) deleteSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	selected, selectedOK := m.selectedNode()
	if !ok || !selectedOK || selected.IsRemote() || row.Local == nil {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkDelete(row.Name), common.Refresh, m.loadRows)
}

func (m *Model) forgetSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	selected, selectedOK := m.selectedNode()
	if !ok || !selectedOK || selected.IsRemote() || row.Local == nil {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkForget(row.Name), common.Refresh, m.loadRows)
}

func (m *Model) trackSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	selected, selectedOK := m.selectedNode()
	if !ok || !selectedOK {
		return nil
	}
	if selected.IsRemote() {
		if selected.Remote == "" || selected.Tracked {
			return nil
		}
		return m.context.RunCommand(jj.BookmarkTrack(row.Name, selected.Remote), common.Refresh, m.loadRows)
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
	selected, selectedOK := m.selectedNode()
	if !ok || !selectedOK || !selected.IsRemote() {
		return nil
	}
	if selected.Remote == "" || !selected.Tracked {
		return nil
	}
	return m.context.RunCommand(jj.BookmarkUntrack(row.Name, selected.Remote), common.Refresh, m.loadRows)
}

func (m *Model) moveSelected() tea.Cmd {
	row, ok := m.selectedBookmark()
	selected, selectedOK := m.selectedNode()
	if !ok || !selectedOK || selected.IsRemote() || row.Local == nil || m.callbacks.BeginMoveBookmark == nil {
		return nil
	}
	m.pendingSelectionHint = row.Name
	return m.callbacks.BeginMoveBookmark(row.Name)
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
