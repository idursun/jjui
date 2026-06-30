package bookmarkpane

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type RevealRevisionMsg struct {
	CommitID string
}

type BeginMoveBookmarkMsg struct {
	Name string
}

type BeginCreateBookmarkMsg struct{}

type PaneClickedMsg struct{}

type RevisionContextMsg struct {
	CurrentCommitID  string
	VisibleCommitIDs []string
}

type rowsLoadedMsg struct {
	tree bookmarkTree
}

type ItemClickedMsg struct {
	Index int
}

type RemoteClickedMsg struct {
	Index int
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

type rowSelectionMode int

const (
	selectionResetTop rowSelectionMode = iota
	selectionPreserve
)

// paneMode is the single source of truth for the pane's input precedence:
// an open confirmation takes priority over filter editing, which takes
// priority over normal navigation.
type paneMode int

const (
	modeNormal paneMode = iota
	modeFilter
	modeConfirm
)

type styles struct {
	title              lipgloss.Style
	text               lipgloss.Style
	dimmed             lipgloss.Style
	selected           lipgloss.Style
	localBookmark      lipgloss.Style
	remoteBookmark     lipgloss.Style
	remoteBookmarkName lipgloss.Style
	trackedBookmark    lipgloss.Style
	deleted            lipgloss.Style
	conflict           lipgloss.Style
	filterPrompt       lipgloss.Style
	childGuide         lipgloss.Style
}

type Model struct {
	context              *context.MainContext
	focused              bool
	currentCommitID      string
	visibleCommitIDs     []string
	tree                 bookmarkTree
	visibleRows          []visibleRow
	remoteNames          []string
	selectedRemoteIdx    int
	expanded             map[string]bool
	cursor               int
	selected             map[string]bool
	listRenderer         *render.ListRenderer
	lastListHeight       int
	ensureCursorVisible  bool
	filterInput          textinput.Model
	filterState          filterState
	filterText           string
	pendingInput         pendingInputKind
	pendingSelectionHint string
	selectionMode        rowSelectionMode
	styles               styles
	confirmation         *confirmation.Model
}

var (
	_ common.ImmediateModel = (*Model)(nil)
	_ common.Editable       = (*Model)(nil)
	_ common.ScopeProvider  = (*Model)(nil)
)

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) OnShow() tea.Cmd {
	m.SetFocused(false)
	m.confirmation = nil
	m.pendingSelectionHint = ""
	m.selectionMode = selectionResetTop
	m.cursor = 0
	m.lastListHeight = 0
	m.ensureCursorVisible = true
	m.listRenderer.StartLine = 0
	m.clearSelections()
	return m.loadRows
}

func (m *Model) OnHide() {
	m.SetFocused(false)
	m.confirmation = nil
	m.lastListHeight = 0
	m.clearSelections()
}

func (m *Model) Scopes() []common.Scope {
	if !m.Focused() {
		return nil
	}

	leak := common.LeakGlobal
	if m.IsEditing() {
		leak = common.LeakNone
	}
	return []common.Scope{{
		Name:    m.ScopeName(),
		Leak:    leak,
		Handler: m,
	}}
}

func New(c *context.MainContext) *Model {
	palette := common.DefaultPalette
	s := styles{
		title:              palette.Get("title"),
		text:               palette.Get("picker text"),
		dimmed:             palette.Get("picker dimmed"),
		selected:           palette.Get("revisions selected"),
		localBookmark:      palette.Get("picker bookmark"),
		remoteBookmark:     palette.Get("picker dimmed"),
		remoteBookmarkName: palette.Get("picker matched"),
		trackedBookmark:    palette.Get("status text"),
		deleted:            palette.Get("deleted"),
		conflict:           palette.Get("error"),
		filterPrompt:       palette.Get("picker matched"),
		childGuide:         palette.Get("picker dimmed"),
	}

	filterInput := textinput.New()
	filterInput.Prompt = "Filter: "
	filterInput.SetVirtualCursor(false)
	fis := filterInput.Styles()
	fis.Focused.Prompt = s.filterPrompt
	fis.Focused.Text = s.text
	fis.Blurred.Prompt = s.filterPrompt
	fis.Blurred.Text = s.text
	filterInput.SetStyles(fis)

	m := &Model{
		context:      c,
		remoteNames:  []string{allRemoteFilter, localRemoteFilter},
		expanded:     make(map[string]bool),
		selected:     make(map[string]bool),
		listRenderer: render.NewListRenderer(itemScrollMsg{}),
		filterInput:  filterInput,
		styles:       s,
	}
	m.listRenderer.Z = render.ZMenuContent
	return m
}

func (m *Model) Focused() bool { return m.focused }

func (m *Model) mode() paneMode {
	switch {
	case m.confirmation != nil:
		return modeConfirm
	case m.filterState == filterEditing:
		return modeFilter
	default:
		return modeNormal
	}
}

func (m *Model) IsEditing() bool { return m.mode() != modeNormal }

func (m *Model) ScopeName() keybindings.ScopeName {
	switch m.mode() {
	case modeConfirm:
		return keybindings.ScopeName(actions.ScopeBookmarkPaneConfirmation)
	case modeFilter:
		return keybindings.ScopeName(actions.ScopeBookmarkPaneFilter)
	default:
		return keybindings.ScopeName(actions.ScopeBookmarkPane)
	}
}

func (m *Model) startFilterEditing() tea.Cmd {
	m.filterState = filterEditing
	m.filterInput.Focus()
	m.filterInput.CursorEnd()
	return textinput.Blink
}

func (m *Model) finishFilterEditing() {
	m.filterText = strings.TrimSpace(m.filterInput.Value())
	m.filterInput.Blur()
	if m.filterText == "" {
		m.filterState = filterOff
		return
	}
	m.filterState = filterApplied
}

func (m *Model) clearFilter(resetCursor bool) {
	m.filterInput.SetValue("")
	m.filterText = ""
	m.filterState = filterOff
	m.filterInput.Blur()
	m.applyFilters(resetCursor)
}

func (m *Model) SyncRevisionContext(currentCommitID string, visibleCommitIDs []string) {
	if m.currentCommitID == currentCommitID && slices.Equal(m.visibleCommitIDs, visibleCommitIDs) {
		return
	}

	selectedTarget, hadSelection := m.selectedTarget()
	startLine := m.listRenderer.StartLine

	m.currentCommitID = currentCommitID
	m.visibleCommitIDs = slices.Clone(visibleCommitIDs)
	m.tree.sort(currentCommitID, visibleCommitIDs)
	m.syncRemoteNamesWithTree()
	m.syncSelectionsWithTree()
	m.visibleRows = m.tree.buildVisibleRows(m.currentFilterText(), m.activeRemoteFilter())

	selectionRestored := hadSelection && m.selectTarget(selectedTarget)
	if len(m.visibleRows) == 0 {
		m.cursor = 0
	} else if !selectionRestored && m.cursor >= len(m.visibleRows) {
		m.cursor = len(m.visibleRows) - 1
	}
	m.listRenderer.StartLine = startLine
}

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused && m.filterState == filterEditing {
		m.finishFilterEditing()
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case confirmation.CloseMsg:
		m.confirmation = nil
		return nil
	case confirmation.SelectOptionMsg:
		if m.mode() == modeConfirm {
			return m.confirmation.Update(msg)
		}
		return nil
	case RevisionContextMsg:
		m.SyncRevisionContext(msg.CurrentCommitID, msg.VisibleCommitIDs)
		return nil
	case rowsLoadedMsg:
		previousTarget, hadSelection := m.selectedTarget()
		m.tree = msg.tree
		m.syncRemoteNamesWithTree()
		m.syncSelectionsWithTree()
		m.applyFilters(m.selectionMode == selectionResetTop)
		switch {
		case m.pendingSelectionHint != "":
			m.selectTarget(m.pendingSelectionHint)
			m.pendingSelectionHint = ""
		case hadSelection && m.selectionMode == selectionPreserve:
			m.selectTarget(previousTarget)
		}
		m.selectionMode = selectionPreserve
		return nil
	case PaneClickedMsg:
		return nil
	case ItemClickedMsg:
		if m.mode() == modeConfirm {
			return nil
		}
		if msg.Index >= 0 && msg.Index < len(m.visibleRows) {
			m.cursor = msg.Index
			m.ensureCursorVisible = true
		}
		return nil
	case RemoteClickedMsg:
		if m.mode() == modeConfirm {
			return nil
		}
		if msg.Index >= 0 && msg.Index < len(m.remoteNames) {
			m.selectedRemoteIdx = msg.Index
			m.applyFilters(true)
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
			if !ok || value == "" || value == selected.Name || selected.Local == nil || !selected.Local.Present {
				return nil
			}
			m.pendingSelectionHint = value
			return m.context.RunCommand(jj.BookmarkRename(selected.Name, value), common.Refresh)
		}
		return nil
	case input.CancelledMsg:
		m.pendingInput = pendingInputNone
		return nil
	case common.RefreshMsg:
		m.selectionMode = selectionPreserve
		return m.loadRows
	case tea.WindowSizeMsg:
		m.lastListHeight = 0
		return nil
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case tea.KeyMsg:
		switch m.mode() {
		case modeConfirm:
			return m.confirmation.Update(msg)
		case modeFilter:
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

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	if !m.Focused() {
		return nil, false
	}
	return m.handleBookmarkIntent(intent)
}

func (m *Model) handleBookmarkIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch m.mode() {
	// when confirmation is active for bookmark operations like rename, delete, etc.
	case modeConfirm:
		switch intent.(type) {
		case intents.Apply, intents.Cancel, intents.OptionSelect:
			return m.confirmation.Update(intent), true
		}
		return nil, false
	// when filtering is active triggered by `/`
	case modeFilter:
		switch intent.(type) {
		case intents.Apply:
			m.finishFilterEditing()
			m.applyFilters(true)
			return nil, true
		case intents.Cancel:
			m.clearFilter(true)
			return nil, true
		}
		return nil, false
	default:
		return m.handleNormalIntent(intent)
	}
}

func (m *Model) handleNormalIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Apply:
		return m.revealSelected(), true
	case intents.Cancel:
		if m.currentFilterText() != "" {
			m.clearFilter(true)
			return nil, true
		}
		if len(m.selected) > 0 {
			m.clearSelections()
			return nil, true
		}
		return common.Close, true
	case intents.OptionSelect:
		return nil, false
	case intents.BookmarkPaneNavigate:
		if intent.IsPage {
			height := max(1, m.visibleHeight())
			m.ensureCursorVisible = false
			m.listRenderer.StartLine += intent.Delta * height
			if m.listRenderer.StartLine < 0 {
				m.listRenderer.StartLine = 0
			}
			return nil, true
		}
		m.moveCursor(intent.Delta)
		return nil, true
	case intents.BookmarkPaneOpenFilter:
		return m.startFilterEditing(), true
	case intents.BookmarkPaneCycleRemotes:
		m.cycleRemotes(intent.Delta)
		return nil, true
	case intents.BookmarkPaneToggleExpand:
		m.toggleExpandSelected()
		return nil, true
	case intents.BookmarkPaneEdit:
		return m.editSelected(), true
	case intents.BookmarkPaneNew:
		return m.newFromSelected(), true
	case intents.BookmarkPaneRename:
		return m.renameSelected(), true
	case intents.BookmarkPaneDelete:
		return m.deleteSelected(), true
	case intents.BookmarkPaneForget:
		return m.forgetSelected(), true
	case intents.BookmarkPaneCreate:
		return m.createSelected(), true
	case intents.BookmarkPaneTrack:
		return m.trackSelected(), true
	case intents.BookmarkPaneUntrack:
		return m.untrackSelected(), true
	case intents.BookmarkPaneMove:
		return m.moveSelected(), true
	case intents.BookmarkPaneShowInRevision:
		return m.revealSelected(), true
	case intents.BookmarkPaneSetRevset:
		return m.showSelectedInRevisions(), true
	case intents.BookmarkPaneToggleSelect:
		m.toggleSelectCurrent()
		return nil, true
	case intents.BookmarkPanePush:
		return m.pushSelected(), true
	case intents.BookmarkPaneFetch:
		return m.fetchSelected(), true
	}
	return nil, false
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	dl.AddInteraction(box.R, PaneClickedMsg{}, render.InteractionClick, render.ZMenuBorder)
	dl.AddFill(box.R, ' ', m.styles.text, render.ZMenuContent)

	content := box
	if content.R.Dx() <= 0 || content.R.Dy() <= 0 {
		return
	}

	titleBox, content := content.CutTop(1)
	m.renderTitle(dl, titleBox)
	remoteBox, content := content.CutTop(1)
	m.renderRemotes(dl, remoteBox)
	var listBox layout.Box
	if m.filterState == filterEditing || m.currentFilterText() != "" {
		filterBox, rest := content.CutTop(1)
		m.renderFilter(dl, filterBox)
		_, listBox = rest.CutTop(1)
	} else {
		_, listBox = content.CutTop(1)
	}
	m.renderList(dl, listBox)
}

func (m *Model) loadRows() tea.Msg {
	output, err := m.context.RunCommandImmediate(jj.BookmarkListAll())
	if err != nil {
		return rowsLoadedMsg{}
	}

	return rowsLoadedMsg{tree: loadBookmarkTree(string(output), m.expanded, m.currentCommitID, m.visibleCommitIDs)}
}
