package bookmarkpane

import (
	"strings"

	"github.com/idursun/jjui/internal/jj"
)

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
	m.visibleRows = m.tree.buildVisibleRows(m.currentFilterText(), m.activeRemoteFilter())
	if resetCursor || m.cursor >= len(m.visibleRows) {
		m.cursor = 0
	}
	m.listRenderer.StartLine = 0
}

func (m *Model) activeRemoteFilter() string {
	if m.selectedRemoteIdx < 0 || m.selectedRemoteIdx >= len(m.remoteNames) {
		return allRemoteFilter
	}
	return m.remoteNames[m.selectedRemoteIdx]
}

func (m *Model) cycleRemotes(delta int) {
	if len(m.remoteNames) == 0 {
		return
	}
	next := m.selectedRemoteIdx + delta
	for next < 0 {
		next += len(m.remoteNames)
	}
	next %= len(m.remoteNames)
	if next == m.selectedRemoteIdx {
		return
	}
	m.selectedRemoteIdx = next
	m.applyFilters(true)
}

func (m *Model) syncRemoteNamesWithTree() {
	current := m.activeRemoteFilter()
	remoteNames := []string{allRemoteFilter, localRemoteFilter}
	seen := map[string]bool{
		allRemoteFilter:   true,
		localRemoteFilter: true,
	}
	for _, item := range m.tree.Items {
		for _, remote := range item.Bookmark.Remotes {
			if seen[remote.Remote] {
				continue
			}
			seen[remote.Remote] = true
			remoteNames = append(remoteNames, remote.Remote)
		}
	}

	m.remoteNames = remoteNames
	m.selectedRemoteIdx = 0
	for idx, remote := range remoteNames {
		if remote == current {
			m.selectedRemoteIdx = idx
			break
		}
	}
}

func (m *Model) visibleHeight() int {
	if m.lastListHeight > 0 {
		return m.lastListHeight
	}
	return 8
}

func (m *Model) selectedRow() (visibleRow, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visibleRows) {
		return visibleRow{}, false
	}
	return m.visibleRows[m.cursor], true
}

func (m *Model) bookmarkItem(index int) (bookmarkTreeItem, bool) {
	if index < 0 || index >= len(m.tree.Items) {
		return bookmarkTreeItem{}, false
	}
	return m.tree.Items[index], true
}

func (m *Model) rowNode(row visibleRow) (bookmarkRowNode, bool) {
	item, ok := m.bookmarkItem(row.BookmarkIndex)
	if !ok {
		return bookmarkRowNode{}, false
	}
	if row.Kind == refKindRemote {
		return item.remoteNode(row.RemoteIndex)
	}
	return item.localNode()
}

func (m *Model) selectedBookmark() (jj.Bookmark, bool) {
	row, ok := m.selectedRow()
	if !ok {
		return jj.Bookmark{}, false
	}
	item, ok := m.bookmarkItem(row.BookmarkIndex)
	if !ok {
		return jj.Bookmark{}, false
	}
	return item.Bookmark, true
}

func (m *Model) selectedBookmarkAndNode() (jj.Bookmark, bookmarkRowNode, bool) {
	row, ok := m.selectedRow()
	if !ok {
		return jj.Bookmark{}, bookmarkRowNode{}, false
	}
	item, ok := m.bookmarkItem(row.BookmarkIndex)
	if !ok {
		return jj.Bookmark{}, bookmarkRowNode{}, false
	}
	node, ok := m.rowNode(row)
	if !ok {
		return jj.Bookmark{}, bookmarkRowNode{}, false
	}
	return item.Bookmark, node, true
}

func (m *Model) selectedNode() (bookmarkRowNode, bool) {
	row, ok := m.selectedRow()
	if !ok {
		return bookmarkRowNode{}, false
	}
	return m.rowNode(row)
}

func (m *Model) selectedLocalBookmark() (jj.Bookmark, bookmarkRowNode, bool) {
	bookmark, node, ok := m.selectedBookmarkAndNode()
	if !ok || node.IsRemote() || bookmark.Local == nil || !bookmark.Local.Present {
		return jj.Bookmark{}, bookmarkRowNode{}, false
	}
	return bookmark, node, true
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
	if target == "" {
		return false
	}
	for idx, row := range m.visibleRows {
		node, ok := m.rowNode(row)
		if !ok {
			continue
		}
		if node.Target() == target || node.CommitID == target {
			m.cursor = idx
			m.ensureCursorVisible = true
			return true
		}
	}
	return false
}

func (m *Model) toggleSelectCurrent() {
	target, ok := m.selectedTarget()
	if !ok {
		return
	}
	if m.selected[target] {
		delete(m.selected, target)
	} else {
		m.selected[target] = true
	}
}

func (m *Model) clearSelections() {
	clear(m.selected)
}

func (m *Model) syncSelectionsWithTree() {
	if len(m.selected) == 0 {
		return
	}

	validTargets := make(map[string]bool, len(m.tree.Items))
	for _, item := range m.tree.Items {
		if item.Bookmark.Local != nil {
			validTargets[item.Bookmark.Name] = true
		}
		for _, remote := range item.Bookmark.Remotes {
			validTargets[item.Bookmark.Name+"@"+remote.Remote] = true
		}
	}
	for target := range m.selected {
		if !validTargets[target] {
			delete(m.selected, target)
		}
	}
}

type bookmarkSelection struct {
	bookmark jj.Bookmark
	node     bookmarkRowNode
}

func (m *Model) selectionsForBookmarkOperation() []bookmarkSelection {
	if len(m.selected) == 0 {
		bookmark, node, ok := m.selectedBookmarkAndNode()
		if !ok {
			return nil
		}
		return []bookmarkSelection{{bookmark: bookmark, node: node}}
	}

	selections := make([]bookmarkSelection, 0, len(m.selected))
	seen := make(map[string]bool, len(m.selected))
	for _, item := range m.tree.Items {
		if item.Bookmark.Local != nil {
			node, _ := item.localNode()
			target := node.Target()
			if m.selected[target] && !seen[target] {
				seen[target] = true
				selections = append(selections, bookmarkSelection{bookmark: item.Bookmark, node: node})
			}
		}
		for remoteIndex := range item.Bookmark.Remotes {
			node, _ := item.remoteNode(remoteIndex)
			target := node.Target()
			if !m.selected[target] || seen[target] {
				continue
			}
			seen[target] = true
			selections = append(selections, bookmarkSelection{bookmark: item.Bookmark, node: node})
		}
	}
	return selections
}
