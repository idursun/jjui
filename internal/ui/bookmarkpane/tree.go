package bookmarkpane

import (
	"strings"

	"github.com/idursun/jjui/internal/jj"
)

type refKind int

const (
	refKindLocal refKind = iota
	refKindRemote
)

type bookmarkRefNode struct {
	Kind     refKind
	Name     string
	Remote   string
	Tracked  bool
	Conflict bool
	CommitID string
}

func (n bookmarkRefNode) IsRemote() bool {
	return n.Kind == refKindRemote
}

func (n bookmarkRefNode) Target() string {
	if n.IsRemote() {
		return n.Name + "@" + n.Remote
	}
	return n.Name
}

type bookmarkTreeItem struct {
	Name       string
	Conflict   bool
	Local      *bookmarkRefNode
	Remotes    []bookmarkRefNode
	Expanded   bool
	RemoteOnly bool
}

func (i bookmarkTreeItem) primaryNode() bookmarkRefNode {
	if i.Local != nil {
		return *i.Local
	}
	if len(i.Remotes) > 0 {
		return i.Remotes[0]
	}
	return bookmarkRefNode{Name: i.Name}
}

func (i bookmarkTreeItem) commitID() string {
	return i.primaryNode().CommitID
}

type bookmarkTree struct {
	Items []bookmarkTreeItem
}

type visibleRow struct {
	BookmarkIndex int
	RemoteIndex   int
	Node          bookmarkRefNode
	Depth         int
	Expanded      bool
	HasChildren   bool
}

func loadBookmarkTree(output string, expanded map[string]bool) bookmarkTree {
	bookmarks := jj.ParseBookmarkListOutput(output)
	items := make([]bookmarkTreeItem, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		item := bookmarkTreeItem{
			Name:       bookmark.Name,
			Conflict:   bookmark.Conflict,
			Expanded:   expanded[bookmark.Name],
			RemoteOnly: bookmark.Local == nil,
		}
		if bookmark.Local != nil {
			item.Local = &bookmarkRefNode{
				Kind:     refKindLocal,
				Name:     bookmark.Name,
				Tracked:  bookmark.Local.Tracked,
				Conflict: bookmark.Conflict,
				CommitID: bookmark.Local.CommitId,
			}
		}
		for _, remote := range bookmark.Remotes {
			item.Remotes = append(item.Remotes, bookmarkRefNode{
				Kind:     refKindRemote,
				Name:     bookmark.Name,
				Remote:   remote.Remote,
				Tracked:  remote.Tracked,
				Conflict: bookmark.Conflict,
				CommitID: remote.CommitId,
			})
		}
		items = append(items, item)
	}
	return bookmarkTree{Items: items}
}

func (t bookmarkTree) buildVisibleRows(filterText string) []visibleRow {
	filterText = strings.ToLower(strings.TrimSpace(filterText))
	rows := make([]visibleRow, 0, len(t.Items))
	for bookmarkIndex, item := range t.Items {
		if !bookmarkMatches(item, filterText) {
			continue
		}
		primary := item.primaryNode()
		rows = append(rows, visibleRow{
			BookmarkIndex: bookmarkIndex,
			Node:          primary,
			Depth:         0,
			Expanded:      item.Expanded,
			HasChildren:   len(item.Remotes) > 0,
		})
		if !item.Expanded {
			continue
		}
		for remoteIndex, remote := range item.Remotes {
			if filterText != "" && !nodeMatches(remote, filterText) && !nodeMatches(primary, filterText) {
				continue
			}
			rows = append(rows, visibleRow{
				BookmarkIndex: bookmarkIndex,
				RemoteIndex:   remoteIndex,
				Node:          remote,
				Depth:         1,
			})
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
	for _, remote := range item.Remotes {
		if nodeMatches(remote, filterText) {
			return true
		}
	}
	return false
}

func nodeMatches(node bookmarkRefNode, filterText string) bool {
	if filterText == "" {
		return true
	}
	haystacks := []string{node.Name, node.Target(), node.CommitID}
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
