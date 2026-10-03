package bookmarkpane

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

func (m *Model) renderTitle(dl *render.DisplayContext, box layout.Box) {
	dl.Text(box.R.Min.X, box.R.Min.Y, render.ZPreview).
		Styled(" Bookmarks ", m.styles.title).
		Done()
}

func (m *Model) renderRemotes(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	palette := theme.DefaultPalette
	textStyle := palette.Get("bookmarks", "remote", "text", false)
	dimmedStyle := palette.Get("bookmarks", "remote", "dimmed", false)
	titleStyle := palette.Get("bookmarks", "remote", "title", false)
	selectedStyle := palette.GetBlended("bookmarks", "remote", "", true)
	dl.AddFill(box.R, ' ', textStyle, render.ZPreview)
	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZPreview).
		Styled(" ", textStyle).
		Styled("Remotes: ", titleStyle)
	for idx, remoteName := range m.remoteNames {
		style := dimmedStyle
		if idx == m.selectedRemoteIdx {
			style = selectedStyle
		}
		tb.Clickable(remoteName, style, RemoteClickedMsg{Index: idx}).Styled(" ", textStyle)
	}
	tb.Done()
}

func (m *Model) renderFilter(dl *render.DisplayContext, box layout.Box) {
	if m.filterState == filterEditing {
		menuTextStyle := theme.DefaultPalette.Get("bookmarks", "input", "text", false)
		menuMatchedStyle := theme.DefaultPalette.Get("bookmarks", "input", "matched", false)
		fis := m.filterInput.Styles()
		fis.Focused.Prompt = menuMatchedStyle.PaddingLeft(1)
		fis.Focused.Text = menuTextStyle
		fis.Blurred.Prompt = menuMatchedStyle.PaddingLeft(1)
		fis.Blurred.Text = menuTextStyle
		m.filterInput.SetStyles(fis)
		m.filterInput.SetWidth(max(box.R.Dx()-2, 0))
		dl.AddDraw(box.R, m.filterInput.View(), render.ZPreview)
		dl.SetCursorInRect(m.filterInput.Cursor(), box.R, 0, 0)
		return
	}
	filterText := m.currentFilterText()
	if filterText == "" {
		return
	}
	dl.Text(box.R.Min.X, box.R.Min.Y, render.ZPreview).
		Styled(" ", m.styles.text).
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
		func(index int, mouse tea.Mouse) tea.Msg {
			return ItemClickedMsg{
				Index: index,
				Ctrl:  mouse.Mod&tea.ModCtrl != 0,
				Alt:   mouse.Mod&tea.ModAlt != 0,
			}
		},
	)
	m.listRenderer.RegisterScroll(dl, box)
	m.ensureCursorVisible = false
}

func (m *Model) renderConfirmation(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 || m.confirmation == nil {
		return
	}
	m.confirmation.Styles.Border = theme.DefaultPalette.GetBorder("confirmation", "", "border", false, lipgloss.NormalBorder()).Padding(1)
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
	s := m.styles
	if index == m.cursor && m.Focused() {
		s = newStyles(true)
		dl.AddFill(rect, ' ', s.text, render.ZPreview)
	}

	tb := dl.Text(rect.Min.X, rect.Min.Y, render.ZPreview)
	if m.selected[node.Target()] {
		tb.Styled("✓ ", s.selected)
	} else {
		tb.Styled("  ", s.text)
	}
	if row.Depth > 0 {
		m.renderRemoteChildRow(tb, node, s)
		tb.Done()
		return
	}

	m.renderTopLevelRow(tb, row, group, node, s)
	tb.Done()
}

func (m *Model) renderRemoteChildRow(tb *render.TextBuilder, node bookmarkRowNode, s styles) {
	tb.Styled("     ", s.text).
		Styled(fmt.Sprintf("@%s", node.Remote), s.remoteBookmarkName).
		Styled("  ", s.text).
		Styled(node.Target(), s.text)
	m.renderRowMetadata(tb, node, s)
}

func (m *Model) renderTopLevelRow(tb *render.TextBuilder, row visibleRow, group bookmarkTreeItem, node bookmarkRowNode, s styles) {
	label := " local "
	style := s.localBookmark
	if node.IsRemote() {
		label = " remote "
		style = s.remoteBookmark
	} else if node.Deleted {
		label = " deleted "
		style = s.deleted
	}

	prefix := "  "
	if row.HasChildren {
		if row.Expanded {
			prefix = "▾ "
		} else {
			prefix = "▸ "
		}
	}

	tb.Styled(prefix, s.childGuide).
		Styled(label, style).
		Styled(" ", s.text).
		Styled(node.Name, s.text)
	if node.IsRemote() {
		tb.Styled("  ", s.text).Styled(node.Remote, s.remoteBookmarkName)
	} else {
		// Show every remote name tracking this bookmark.
		for i, remote := range group.Bookmark.Remotes {
			separator := " "
			if i == 0 {
				separator = "  "
			}
			tb.Styled(separator, s.text).Styled(remote.Remote, s.remoteBookmarkName)
		}
	}
	m.renderRowMetadata(tb, node, s)
}

func (m *Model) renderRowMetadata(tb *render.TextBuilder, node bookmarkRowNode, s styles) {
	if node.Tracked {
		tb.Styled(" ", s.text).Styled("tracked", s.trackedBookmark)
	}
	if node.Deleted {
		tb.Styled(" ", s.text).Styled("deleted", s.deleted)
	}
	if node.Conflict {
		tb.Styled(" ", s.text).Styled("conflict", s.conflict)
	}
	if node.CommitID != "" {
		tb.Styled(" ", s.text).Styled(node.CommitID, s.dimmed)
	}
}
