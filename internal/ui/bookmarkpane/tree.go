package bookmarkpane

import (
	"math"
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/jj"
)

type refKind int

const (
	refKindLocal refKind = iota
	refKindRemote
)

const (
	allRemoteFilter   = "all"
	localRemoteFilter = "local"
)

type bookmarkRowNode struct {
	Kind     refKind
	Name     string
	Remote   string
	Tracked  bool
	Conflict bool
	Deleted  bool
	CommitID string
}

func (n bookmarkRowNode) IsRemote() bool {
	return n.Kind == refKindRemote
}

func (n bookmarkRowNode) Target() string {
	if n.IsRemote() {
		return n.Name + "@" + n.Remote
	}
	return n.Name
}

func (n bookmarkRowNode) bookmarkRemote(bookmark jj.Bookmark) (jj.BookmarkRemote, bool) {
	if !n.IsRemote() {
		return jj.BookmarkRemote{}, false
	}
	for _, remote := range bookmark.Remotes {
		if remote.Remote == n.Remote {
			return remote, true
		}
	}
	return jj.BookmarkRemote{}, false
}

type bookmarkTreeItem struct {
	Bookmark jj.Bookmark
	Expanded bool
}

func (i bookmarkTreeItem) localNode() (bookmarkRowNode, bool) {
	if i.Bookmark.Local == nil {
		return bookmarkRowNode{}, false
	}
	return bookmarkRowNode{
		Kind:     refKindLocal,
		Name:     i.Bookmark.Name,
		Tracked:  i.Bookmark.Local.Tracked,
		Conflict: i.Bookmark.Conflict,
		Deleted:  !i.Bookmark.Local.Present,
		CommitID: i.Bookmark.Local.CommitId,
	}, true
}

func (i bookmarkTreeItem) remoteNode(index int) (bookmarkRowNode, bool) {
	if index < 0 || index >= len(i.Bookmark.Remotes) {
		return bookmarkRowNode{}, false
	}
	remote := i.Bookmark.Remotes[index]
	return bookmarkRowNode{
		Kind:     refKindRemote,
		Name:     i.Bookmark.Name,
		Remote:   remote.Remote,
		Tracked:  remote.Tracked,
		Conflict: i.Bookmark.Conflict,
		Deleted:  !remote.Present,
		CommitID: remote.CommitId,
	}, true
}

func (i bookmarkTreeItem) primaryNode() bookmarkRowNode {
	if local, ok := i.localNode(); ok {
		return local
	}
	if remote, ok := i.remoteNode(0); ok {
		return remote
	}
	return bookmarkRowNode{Name: i.Bookmark.Name}
}

func (i bookmarkTreeItem) remoteOnly() bool {
	return i.Bookmark.Local == nil
}

type bookmarkTree struct {
	Items []bookmarkTreeItem
}

type visibleRow struct {
	BookmarkIndex int
	Kind          refKind
	RemoteIndex   int
	Depth         int
	Expanded      bool
	HasChildren   bool
}

func loadBookmarkTree(output string, expanded map[string]bool, currentCommitID string, visibleCommitIDs []string) bookmarkTree {
	bookmarks := jj.ParseBookmarkListOutput(output)
	items := make([]bookmarkTreeItem, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		items = append(items, bookmarkTreeItem{
			Bookmark: bookmark,
			Expanded: expanded[bookmark.Name],
		})
	}

	tree := bookmarkTree{Items: items}
	tree.sort(currentCommitID, visibleCommitIDs)
	return tree
}

func (t *bookmarkTree) sort(currentCommitID string, visibleCommitIDs []string) {
	distanceMap := calcDistanceMap(currentCommitID, visibleCommitIDs)
	slices.SortFunc(t.Items, func(a, b bookmarkTreeItem) int {
		return compareBookmarkTreeItems(a, b, distanceMap)
	})
}

func (t bookmarkTree) buildVisibleRows(filterText, remoteFilter string) []visibleRow {
	filterText = strings.ToLower(strings.TrimSpace(filterText))
	if remoteFilter != "" && remoteFilter != allRemoteFilter {
		return t.buildVisibleRowsForRemote(filterText, remoteFilter)
	}

	rows := make([]visibleRow, 0, len(t.Items))
	for bookmarkIndex, item := range t.Items {
		if !bookmarkMatches(item, filterText) {
			continue
		}
		if item.remoteOnly() {
			for remoteIndex := range item.Bookmark.Remotes {
				remote, _ := item.remoteNode(remoteIndex)
				if !nodeMatches(remote, filterText) {
					continue
				}
				rows = append(rows, visibleRow{
					BookmarkIndex: bookmarkIndex,
					Kind:          refKindRemote,
					RemoteIndex:   remoteIndex,
					Depth:         0,
				})
			}
			continue
		}
		primary := item.primaryNode()
		row := visibleRow{
			BookmarkIndex: bookmarkIndex,
			Kind:          refKindLocal,
			RemoteIndex:   -1,
			Depth:         0,
			Expanded:      item.Expanded,
			HasChildren:   len(item.Bookmark.Remotes) > 0,
		}
		rows = append(rows, row)
		if !item.Expanded {
			continue
		}
		for remoteIndex := range item.Bookmark.Remotes {
			remote, _ := item.remoteNode(remoteIndex)
			if filterText != "" && !nodeMatches(remote, filterText) && !nodeMatches(primary, filterText) {
				continue
			}
			rows = append(rows, visibleRow{
				BookmarkIndex: bookmarkIndex,
				Kind:          refKindRemote,
				RemoteIndex:   remoteIndex,
				Depth:         1,
			})
		}
	}
	return rows
}

func (t bookmarkTree) buildVisibleRowsForRemote(filterText, remoteFilter string) []visibleRow {
	rows := make([]visibleRow, 0, len(t.Items))
	for bookmarkIndex, item := range t.Items {
		if remoteFilter == localRemoteFilter {
			node, ok := item.localNode()
			if !ok || !nodeMatches(node, filterText) {
				continue
			}
			rows = append(rows, visibleRow{
				BookmarkIndex: bookmarkIndex,
				Kind:          refKindLocal,
				RemoteIndex:   -1,
				Depth:         0,
			})
			continue
		}

		for remoteIndex, remote := range item.Bookmark.Remotes {
			if remote.Remote != remoteFilter {
				continue
			}
			node, _ := item.remoteNode(remoteIndex)
			if !nodeMatches(node, filterText) {
				break
			}
			rows = append(rows, visibleRow{
				BookmarkIndex: bookmarkIndex,
				Kind:          refKindRemote,
				RemoteIndex:   remoteIndex,
				Depth:         0,
			})
			break
		}
	}
	return rows
}

func bookmarkMatches(item bookmarkTreeItem, filterText string) bool {
	if filterText == "" {
		return true
	}
	if nodeMatches(item.primaryNode(), filterText) {
		return true
	}
	for remoteIndex := range item.Bookmark.Remotes {
		remote, _ := item.remoteNode(remoteIndex)
		if nodeMatches(remote, filterText) {
			return true
		}
	}
	return false
}

func nodeMatches(node bookmarkRowNode, filterText string) bool {
	if filterText == "" {
		return true
	}
	haystacks := []string{node.Name, node.Target(), node.CommitID}
	if node.Deleted {
		haystacks = append(haystacks, "deleted")
	}
	if node.IsRemote() {
		haystacks = append(haystacks, node.Remote)
	}
	for _, haystack := range haystacks {
		if strings.Contains(strings.ToLower(haystack), filterText) {
			return true
		}
	}
	return false
}

func compareBookmarkTreeItems(a, b bookmarkTreeItem, distanceMap map[string]int) int {
	if rankA, rankB := bookmarkSortRank(a), bookmarkSortRank(b); rankA != rankB {
		return rankA - rankB
	}
	commitIDA := ""
	if a.Bookmark.Local != nil {
		commitIDA = a.Bookmark.Local.CommitId
	}
	commitIDB := ""
	if b.Bookmark.Local != nil {
		commitIDB = b.Bookmark.Local.CommitId
	}
	if distCmp := compareDistance(bookmarkDistance(distanceMap, commitIDA), bookmarkDistance(distanceMap, commitIDB)); distCmp != 0 {
		return distCmp
	}
	return strings.Compare(a.Bookmark.Name, b.Bookmark.Name)
}

func bookmarkSortRank(item bookmarkTreeItem) int {
	switch {
	case item.Bookmark.Local != nil && item.Bookmark.Local.Present:
		return 0
	case item.Bookmark.Local != nil:
		return 1
	default:
		return 2
	}
}

func bookmarkDistance(distanceMap map[string]int, commitID string) int {
	if dist, ok := distanceMap[commitID]; ok {
		return dist
	}
	return math.MinInt32
}

func compareDistance(a, b int) int {
	if a == b {
		return 0
	}
	if a >= 0 && b >= 0 {
		return a - b
	}
	if a < 0 && b < 0 {
		return b - a
	}
	return b - a
}

func calcDistanceMap(current string, commitIDs []string) map[string]int {
	distanceMap := make(map[string]int, len(commitIDs))
	if current == "" {
		for _, commitID := range commitIDs {
			distanceMap[commitID] = math.MinInt32
		}
		return distanceMap
	}

	currentPos := -1
	for i, commitID := range commitIDs {
		if commitID == current {
			currentPos = i
			break
		}
	}
	for i, commitID := range commitIDs {
		distanceMap[commitID] = i - currentPos
	}
	return distanceMap
}
