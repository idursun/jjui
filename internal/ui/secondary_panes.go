package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
)

type secondaryPaneKind int

const (
	secondaryPaneNone secondaryPaneKind = iota
	secondaryPanePreview
	secondaryPaneBookmark
)

func (m *Model) initSplit() {
	m.previewSplit = newSplit(
		newSplitState(config.Current.Preview.WidthPercentage),
		nil,
		m.previewModel,
	)
	m.bookmarkSplit = newSplit(
		newSplitState(45),
		nil,
		m.bookmarkPane,
	)
}

func (m *Model) renderSplit(primary common.ImmediateModel, box layout.Box) {
	switch m.secondaryPaneActive {
	case secondaryPaneBookmark:
		if m.bookmarkSplit == nil {
			return
		}
		m.bookmarkSplit.Primary = primary
		m.bookmarkSplit.Secondary = m.bookmarkPane
		m.bookmarkSplit.Vertical = false
		m.bookmarkSplit.SeparatorVisible = true
		m.bookmarkSplit.Render(m.displayContext, box)
	case secondaryPanePreview:
		if m.previewSplit == nil {
			return
		}
		m.previewSplit.Primary = primary
		m.previewSplit.Secondary = m.previewModel
		m.previewSplit.Render(m.displayContext, box)
	default:
		primary.ViewRect(m.displayContext, box)
	}
}

func (m *Model) previewVisible() bool {
	return m.secondaryPaneActive == secondaryPanePreview && m.previewModel != nil && m.previewModel.Visible()
}

func (m *Model) bookmarkVisible() bool {
	return m.secondaryPaneActive == secondaryPaneBookmark && m.bookmarkPane != nil && m.bookmarkPane.Visible()
}

func (m *Model) bookmarkEditing() bool {
	return m.bookmarkVisible() && m.bookmarkPane != nil && m.bookmarkPane.IsEditing()
}

func (m *Model) syncPreviewSplitOrientation() {
	if m.previewSplit == nil || m.previewModel == nil {
		return
	}
	m.previewSplit.Vertical = m.previewModel.AtBottom()
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

func (m *Model) setBookmarkPaneFocused(focused bool) {
	m.bookmarkPaneFocused = focused
	if m.bookmarkPane != nil {
		m.bookmarkPane.SetFocused(focused)
	}
	m.revisions.SetFocused(!focused)
}

func (m *Model) focusBookmarkPane() {
	if !m.bookmarkVisible() {
		return
	}
	m.setBookmarkPaneFocused(true)
}

func (m *Model) focusNextPane() {
	if !m.bookmarkVisible() {
		return
	}
	m.setBookmarkPaneFocused(!m.bookmarkPaneFocused)
}

func (m *Model) showPreview() {
	if m.previewModel == nil {
		return
	}
	m.previewModel.SetVisible(true)
	m.secondaryPaneActive = secondaryPanePreview
}

func (m *Model) hidePreview() {
	if m.previewModel == nil {
		return
	}
	m.previewModel.SetVisible(false)
	if m.secondaryPaneActive == secondaryPanePreview {
		m.secondaryPaneActive = secondaryPaneNone
	}
}

func (m *Model) togglePreview() {
	if m.previewVisible() {
		m.hidePreview()
		return
	}
	m.showPreview()
}

func (m *Model) openBookmarkPane() tea.Cmd {
	m.syncBookmarkPaneContext()
	m.secondaryRestoreOnClose = secondaryPaneNone
	if m.previewVisible() {
		m.secondaryRestoreOnClose = secondaryPanePreview
		m.previewModel.SetVisible(false)
	}
	m.secondaryPaneActive = secondaryPaneBookmark
	m.setBookmarkPaneFocused(true)
	return m.bookmarkPane.Open()
}

func (m *Model) closeBookmarkPane() tea.Cmd {
	m.bookmarkPane.Close()
	m.bookmarkPaneFocused = false
	m.secondaryPaneActive = secondaryPaneNone
	m.revisions.SetFocused(true)
	if m.secondaryRestoreOnClose == secondaryPanePreview {
		m.previewModel.SetVisible(true)
		m.secondaryPaneActive = secondaryPanePreview
	}
	m.secondaryRestoreOnClose = secondaryPaneNone
	if m.bookmarkRevsetRestore != "" && m.context.CurrentRevset == m.bookmarkRevsetApplied {
		restore := m.bookmarkRevsetRestore
		m.bookmarkRevsetRestore = ""
		m.bookmarkRevsetApplied = ""
		return common.UpdateRevSet(restore)
	}
	m.bookmarkRevsetRestore = ""
	m.bookmarkRevsetApplied = ""
	return nil
}

func (m *Model) showBookmarkTarget(target, commitID string) tea.Cmd {
	revision := target
	if revision == "" {
		revision = commitID
	}
	if revision == "" {
		return nil
	}
	if m.bookmarkRevsetRestore == "" {
		m.bookmarkRevsetRestore = m.context.CurrentRevset
	}
	m.bookmarkRevsetApplied = fmt.Sprintf("::%s", revision)
	return common.UpdateRevSet(m.bookmarkRevsetApplied)
}
