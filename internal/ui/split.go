package ui

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	bookmarkop "github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/split"
)

const (
	previewContentID  = "preview"
	bookmarkContentID = "bookmark-pane"
)

func (m *Model) initSplitContainer() {
	state := split.NewSplitState(config.Current.Preview.WidthPercentage)
	position, err := config.GetPreviewPosition(config.Current)
	if err != nil {
		log.Fatal(err)
	}
	state.SetPlacement(splitPlacementFromPreviewConfig(position))

	m.splitContainer = split.NewSplitContainer(state)
	m.splitContainer.OnPrimaryFocus = m.revisions.SetFocused
	m.splitContainer.RegisterContent(previewContentID, preview.New(m.context))
	m.splitContainer.RegisterContent(bookmarkContentID, bookmarkpane.New(m.context))
	if config.Current.Preview.ShowAtStart {
		_, _ = m.splitContainer.ShowContent(previewContentID)
	}
}

func splitPlacementFromPreviewConfig(position config.PreviewPosition) split.Placement {
	switch position {
	case config.PreviewPositionBottom:
		return split.PlacementBottom
	case config.PreviewPositionRight:
		return split.PlacementRight
	default:
		return split.PlacementAuto
	}
}

func (m *Model) selectionChanged() tea.Cmd {
	return common.SelectionChanged(m.context.SelectedItem)
}

func (m *Model) handleSplitMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case common.ShowPreview:
		cmd, shown := m.splitContainer.ShowContent(previewContentID)
		m.splitContainer.FocusPrimary()
		if shown {
			return tea.Batch(cmd, m.selectionChanged()), true
		}
		return cmd, true
	case revisions.ItemClickedMsg, revisions.PaneClickedMsg:
		m.splitContainer.FocusPrimary()
	case bookmarkpane.PaneClickedMsg, bookmarkpane.ItemClickedMsg, bookmarkpane.RemoteClickedMsg:
		if m.splitContainer.ActiveID() == bookmarkContentID {
			m.splitContainer.FocusSplitContent()
		}
	case bookmarkpane.RevealRevisionMsg:
		m.splitContainer.FocusPrimary()
		return m.revisions.RevealRevision(msg.CommitID), true
	case bookmarkpane.BeginMoveBookmarkMsg:
		m.splitContainer.FocusPrimary()
		return common.RestoreOperation(bookmarkop.NewMoveBookmarkOperation(m.context, msg.Name, m.revisions.SelectedRevision())), true
	case bookmarkop.MoveBookmarkCancelledMsg:
		if m.splitContainer.ActiveID() == bookmarkContentID {
			m.splitContainer.FocusSplitContent()
		}
		return nil, true
	case bookmarkpane.BeginCreateBookmarkMsg:
		m.splitContainer.FocusPrimary()
		return common.RestoreOperation(bookmarkop.NewCreateBookmarkOperation(m.context, m.revisions.SelectedRevision())), true
	case split.SplitDragMsg:
		m.splitContainer.StartDrag(msg)
	}
	return nil, false
}

func (m *Model) handleSplitIntent(intent intents.Intent) (tea.Cmd, bool) {
	if m.splitContainer.ContentFocused() {
		if cmd, handled := common.RouteIntent(m.splitContainer.Scopes(nil), intent); handled {
			return cmd, true
		}
	}

	switch msg := intent.(type) {
	case intents.ToggleBookmarkPane:
		if m.splitContainer.ActiveID() == bookmarkContentID {
			m.splitContainer.Close()
			return nil, true
		}
		m.syncBookmarkPaneContext()
		cmd, _ := m.splitContainer.ShowContent(bookmarkContentID)
		m.splitContainer.FocusSplitContent()
		return cmd, true
	case intents.FocusNextPane:
		if m.splitContainer.ActiveID() == bookmarkContentID {
			m.splitContainer.ToggleFocus()
		}
		return nil, true
	case intents.PreviewToggle:
		cmd, _ := m.splitContainer.ToggleContent(previewContentID)
		m.splitContainer.FocusPrimary()
		return tea.Batch(cmd, m.selectionChanged()), true
	case intents.PreviewToggleBottom:
		cmd, shown := m.splitContainer.ShowContent(previewContentID)
		m.splitContainer.FocusPrimary()
		m.splitContainer.TogglePosition()
		if shown {
			return tea.Batch(cmd, m.selectionChanged()), true
		}
		return cmd, true
	case intents.PreviewExpand:
		m.splitContainer.Resize(config.Current.Preview.WidthIncrementPercentage)
		return nil, true
	case intents.PreviewShrink:
		m.splitContainer.Resize(-config.Current.Preview.WidthIncrementPercentage)
		return nil, true
	case intents.PreviewShow:
		cmd, _ := m.splitContainer.ShowContent(previewContentID)
		m.splitContainer.FocusPrimary()
		return tea.Batch(cmd, m.splitContainer.Update(msg)), true
	}
	return nil, false
}

func (m *Model) updateSplit(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case common.UpdateRevisionsSuccessMsg, common.SelectionChangedMsg:
		m.syncBookmarkPaneContext()
	}
	return m.splitContainer.Update(msg)
}

func (m *Model) splitScopes(primary []common.Scope) []common.Scope {
	return m.splitContainer.Scopes(primary)
}

func (m *Model) renderSplit(primary common.ImmediateModel, box layout.Box) {
	m.splitContainer.Render(m.displayContext, box, primary)
}

func (m *Model) updateSplitAutoPosition() {
	m.splitContainer.SetAutoPosition(m.height >= m.width/2)
}

// The bookmark pane needs revision context for proximity sorting and for
// deciding whether a bookmark target can be revealed in the current list.
func (m *Model) syncBookmarkPaneContext() {
	currentCommitID := ""
	if selected := m.revisions.SelectedRevision(); selected != nil {
		currentCommitID = selected.CommitId
	}
	m.splitContainer.UpdateContent(bookmarkContentID, bookmarkpane.RevisionContextMsg{
		CurrentCommitID:  currentCommitID,
		VisibleCommitIDs: m.revisions.GetCommitIds(),
	})
}

func (m *Model) handleSplitMouseMsg(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		m.splitContainer.EndDrag()
	case tea.MouseMotionMsg:
		mouse := msg.Mouse()
		return m.splitContainer.DragTo(mouse.X, mouse.Y)
	}
	return false
}
