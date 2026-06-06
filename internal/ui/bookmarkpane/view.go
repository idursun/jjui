package bookmarkpane

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

func (m *Model) renderTitle(dl *render.DisplayContext, box layout.Box) {
	dl.Text(box.R.Min.X, box.R.Min.Y, render.ZMenuContent).
		Styled("Bookmarks", m.styles.title).
		Done()
}

func (m *Model) renderRemotes(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZMenuContent).
		Styled("Remotes: ", m.styles.title)
	for idx, remoteName := range m.remoteNames {
		style := m.styles.dimmed
		if idx == m.selectedRemoteIdx {
			style = common.DefaultPalette.Get("bookmarks menu selected")
		}
		tb.Clickable(remoteName, style, RemoteClickedMsg{Index: idx}).Styled(" ", m.styles.text)
	}
	tb.Done()
}

func (m *Model) renderFilter(dl *render.DisplayContext, box layout.Box) {
	if m.filterState == filterEditing {
		menuTextStyle := common.DefaultPalette.Get("bookmarks menu text")
		menuMatchedStyle := common.DefaultPalette.Get("bookmarks menu matched")
		fis := m.filterInput.Styles()
		fis.Focused.Prompt = menuMatchedStyle.PaddingLeft(1)
		fis.Focused.Text = menuTextStyle
		fis.Blurred.Prompt = menuMatchedStyle.PaddingLeft(1)
		fis.Blurred.Text = menuTextStyle
		m.filterInput.SetStyles(fis)
		m.filterInput.SetWidth(max(box.R.Dx()-2, 0))
		dl.AddDraw(box.R, m.filterInput.View(), render.ZMenuContent)
		dl.SetCursorInRect(m.filterInput.Cursor(), box.R, 0, 0)
		return
	}
	filterText := m.currentFilterText()
	if filterText == "" {
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
	m.lastListHeight = box.R.Dy()
	m.listRenderer.Render(
		dl,
		box,
		len(m.visibleRows),
		m.cursor,
		m.ensureCursorVisible,
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			m.renderListRow(dl, index, rect)
		},
		func(index int, _ tea.Mouse) tea.Msg { return ItemClickedMsg{Index: index} },
	)
	m.listRenderer.RegisterScroll(dl, box)
	m.ensureCursorVisible = false
}

func (m *Model) renderConfirmation(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 || m.confirmation == nil {
		return
	}
	m.confirmation.Styles.Border = common.DefaultPalette.GetBorder("confirmation border", lipgloss.NormalBorder()).Padding(1)
	v := m.confirmation.View()
	w, h := lipgloss.Size(v)
	pw, ph := box.R.Dx(), box.R.Dy()
	sx := box.R.Min.X + max((pw-w)/2, 0)
	sy := box.R.Min.Y + max((ph-h)/2, 0)
	frame := layout.Rect(sx, sy, w, h)
	dl.AddBackdrop(box.R, render.ZDialogs-1)
	m.confirmation.ViewRect(dl, layout.Box{R: frame})
}

func (m *Model) RenderOverlay(dl *render.DisplayContext, box layout.Box) {
	m.renderConfirmation(dl, box)
}

func (m *Model) renderListRow(dl *render.DisplayContext, index int, rect layout.Rectangle) {
	if index < 0 || index >= len(m.visibleRows) {
		return
	}
	row := m.visibleRows[index]
	group, ok := m.bookmarkItem(row.BookmarkIndex)
	if !ok {
		return
	}
	node, ok := m.rowNode(row)
	if !ok {
		return
	}
	if index == m.cursor && m.Focused() {
		dl.AddHighlight(rect, m.styles.selected, render.ZMenuContent+1)
	}

	tb := dl.Text(rect.Min.X, rect.Min.Y, render.ZMenuContent)
	if m.selected[node.Target()] {
		tb.Styled("✓ ", m.styles.selected)
	} else {
		tb.Styled("  ", m.styles.text)
	}
	if row.Depth > 0 {
		m.renderRemoteChildRow(tb, node)
		tb.Done()
		return
	}

	m.renderTopLevelRow(tb, row, group, node)
	tb.Done()
}

func (m *Model) renderRemoteChildRow(tb *render.TextBuilder, node bookmarkRowNode) {
	tb.Styled("     ", m.styles.text).
		Styled(fmt.Sprintf("@%s", node.Remote), m.styles.remoteBookmarkName).
		Styled("  ", m.styles.text).
		Styled(node.Target(), m.styles.text)
	m.renderRowMetadata(tb, node)
}

func (m *Model) renderTopLevelRow(tb *render.TextBuilder, row visibleRow, group bookmarkTreeItem, node bookmarkRowNode) {
	label := " local "
	style := m.styles.localBookmark
	if node.IsRemote() {
		label = " remote "
		style = m.styles.remoteBookmark
	} else if node.Deleted {
		label = " deleted "
		style = m.styles.deleted
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
		Styled(node.Name, m.styles.text)
	if node.IsRemote() {
		tb.Styled("  ", m.styles.text).Styled(node.Remote, m.styles.remoteBookmarkName)
	} else {
		// Show every remote name tracking this bookmark.
		for i, remote := range group.Bookmark.Remotes {
			separator := " "
			if i == 0 {
				separator = "  "
			}
			tb.Styled(separator, m.styles.text).Styled(remote.Remote, m.styles.remoteBookmarkName)
		}
	}
	m.renderRowMetadata(tb, node)
}

func (m *Model) renderRowMetadata(tb *render.TextBuilder, node bookmarkRowNode) {
	if node.Tracked {
		tb.Styled(" ", m.styles.text).Styled("tracked", m.styles.trackedBookmark)
	}
	if node.Deleted {
		tb.Styled(" ", m.styles.text).Styled("deleted", m.styles.deleted)
	}
	if node.Conflict {
		tb.Styled(" ", m.styles.text).Styled("conflict", m.styles.conflict)
	}
	if node.CommitID != "" {
		tb.Styled(" ", m.styles.text).Styled(node.CommitID, m.styles.dimmed)
	}
}
