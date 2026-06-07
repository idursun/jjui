package ui

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/split"
)

func (m *Model) initSplitPanel() {
	state := split.NewSplitState(config.Current.Preview.WidthPercentage)
	previewPositionCfg, err := config.GetPreviewPosition(config.Current)
	if err != nil {
		log.Fatal(err)
	}
	state.SetPosition(previewPositionCfg)
	m.bookmarkPane = bookmarkpane.New(m.context)
	m.splitPanel = split.NewSplitPanel(state, m.bookmarkPane, m.previewModel)
}

func (m *Model) handlePanelIntent(intent intents.Intent) (tea.Cmd, bool) {
	if _, ok := intent.(intents.Cancel); ok && m.panelCancelShouldPreempt() {
		if cmd, handled := m.dismissRootInterruptions(); handled {
			return cmd, true
		}
	}
	if m.bookmarkPane != nil {
		if cmd, handled := m.bookmarkPane.HandleIntent(intent); handled {
			return cmd, true
		}
	}
	switch intent.(type) {
	case intents.ToggleBookmarkPane:
		if m.bookmarkPane.Visible() {
			return m.closeBookmarkPane(), true
		}
		return m.openBookmarkPane(), true
	case intents.FocusNextPane:
		m.splitPanel.ToggleFocus()
		return nil, true
	}
	return nil, false
}

func (m *Model) panelCancelShouldPreempt() bool {
	return m.bookmarkPane != nil && m.bookmarkPane.Focused() && !m.bookmarkPane.IsEditing()
}

func (m *Model) dismissRootInterruptions() (tea.Cmd, bool) {
	switch {
	case m.flash.Any():
		m.flash.DeleteOldest()
		return nil, true
	case m.status.StatusExpanded():
		m.status.ToggleStatusExpand()
		return nil, true
	default:
		return nil, false
	}
}

func (m *Model) openBookmarkPane() tea.Cmd {
	if m.bookmarkPane == nil {
		return nil
	}
	m.syncBookmarkPaneContext()
	m.splitPanel.OpenBookmark()
	m.splitPanel.FocusSecondary()
	return m.bookmarkPane.Open()
}

func (m *Model) closeBookmarkPane() tea.Cmd {
	if m.bookmarkPane == nil {
		return nil
	}
	m.splitPanel.CloseBookmark()
	return m.bookmarkPane.Close()
}

func (m *Model) syncBookmarkPaneContext() {
	if m.bookmarkPane == nil {
		return
	}
	if selected := m.revisions.SelectedRevision(); selected != nil {
		m.bookmarkPane.SetCurrentCommitID(selected.CommitId)
	} else {
		m.bookmarkPane.SetCurrentCommitID("")
	}
	m.bookmarkPane.SetVisibleCommitIDs(m.revisions.GetCommitIds())
}

func (m *Model) syncFocus() {
	focused := m.splitPanel.FocusedSecondary()
	m.revisions.SetFocused(!focused)
	if m.bookmarkPane != nil {
		m.bookmarkPane.SetFocused(focused)
	}
}
