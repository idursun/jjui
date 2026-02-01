package details

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// detailsTreeNode represents a node in the file tree
type detailsTreeNode struct {
	name          string
	path          string
	children      []*detailsTreeNode
	childrenNodes []render.TreeNode
	item          *item
	fileIndex     int
}

func (n *detailsTreeNode) ID() string {
	return n.path
}

func (n *detailsTreeNode) Name() string {
	return n.name
}

func (n *detailsTreeNode) Children() []render.TreeNode {
	if len(n.children) == 0 {
		return nil
	}
	if n.childrenNodes == nil {
		children := make([]render.TreeNode, 0, len(n.children))
		for _, child := range n.children {
			children = append(children, child)
		}
		n.childrenNodes = children
	}
	return n.childrenNodes
}

// treeViewStyles holds the styles for tree view rendering
type treeViewStyles struct {
	Background         lipgloss.Style
	SelectedBackground lipgloss.Style
	Directory          lipgloss.Style
	DirectorySelected  lipgloss.Style
	Added              lipgloss.Style
	Deleted            lipgloss.Style
	Modified           lipgloss.Style
	Renamed            lipgloss.Style
	Copied             lipgloss.Style
}

func newTreeViewStyles() treeViewStyles {
	selected := common.DefaultPalette.Get("diff selected")
	return treeViewStyles{
		Background:         lipgloss.NewStyle(),
		SelectedBackground: lipgloss.NewStyle().Background(selected.GetBackground()).Inherit(selected),
		Directory:          common.DefaultPalette.Get("diff directory text"),
		DirectorySelected:  common.DefaultPalette.Get("diff directory selected"),
		Added:              common.DefaultPalette.Get("diff added"),
		Deleted:            common.DefaultPalette.Get("diff removed"),
		Modified:           common.DefaultPalette.Get("diff modified"),
		Renamed:            common.DefaultPalette.Get("diff renamed"),
		Copied:             common.DefaultPalette.Get("diff copied"),
	}
}

func (s treeViewStyles) styleForStatus(st status) lipgloss.Style {
	switch st {
	case Added:
		return s.Added
	case Deleted:
		return s.Deleted
	case Modified:
		return s.Modified
	case Renamed:
		return s.Renamed
	case Copied:
		return s.Copied
	default:
		return lipgloss.NewStyle()
	}
}

// buildTree builds the tree structure from the flat file list
func (d *DetailsList) buildTree() {
	d.treeRoot = buildDetailsTree(d.files)
	d.treeFileNodes = make(map[string]*detailsTreeNode)
	indexDetailsTreeFiles(d.treeRoot, d.treeFileNodes)
	d.treeScrollOffset = 0
	d.treeEnsureCursor = true
}

// renderTreeView renders files in a tree structure
func (d *DetailsList) renderTreeView(dl *render.DisplayContext, viewRect layout.Box) {
	if d.treeRoot == nil {
		return
	}
	treeStyles := newTreeViewStyles()
	d.treeLastOutput = render.RenderTree(dl, viewRect, render.TreeInput{
		Root:          d.treeRoot,
		SelectedID:    d.treeSelectedID,
		Expanded:      d.treeExpanded,
		DefaultExpand: true,
		ScrollOffset:  d.treeScrollOffset,
		EnsureCursor:  d.treeEnsureCursor,
		Selectable:    nil, // all nodes are selectable
		RenderNode: func(dl *render.DisplayContext, node render.TreeNode, depth int, isSelected bool, isExpanded bool, rect cellbuf.Rectangle) {
			d.renderTreeNode(dl, node, depth, isSelected, isExpanded, rect, treeStyles)
		},
		ClickMessage: d.treeClickMessage,
		ScrollMsg:    FileListScrollMsg{},
	})
	d.treeScrollOffset = d.treeLastOutput.ScrollOffset
	d.treeEnsureCursor = false
}


func (d *DetailsList) renderTreeNode(
	dl *render.DisplayContext,
	node render.TreeNode,
	depth int,
	isSelected bool,
	isExpanded bool,
	rect cellbuf.Rectangle,
	styles treeViewStyles,
) {
	detailsNode, ok := node.(*detailsTreeNode)
	if !ok || detailsNode == nil {
		return
	}

	width := rect.Dx()
	if width <= 0 {
		return
	}

	background := styles.Background
	if isSelected {
		background = styles.SelectedBackground
	}
	dl.AddFill(rect, ' ', background, 0)

	indent := strings.Repeat("  ", depth)
	tb := dl.Text(rect.Min.X, rect.Min.Y, 0)

	if detailsNode.item == nil {
		// Directory node
		arrow := "▾ "
		if !isExpanded {
			arrow = "▸ "
		}
		label := indent + arrow + detailsNode.name + "/"
		if len(label) > width {
			label = label[:width]
		}
		dirStyle := styles.Directory
		if isSelected {
			dirStyle = styles.DirectorySelected
		}
		tb.Styled(label, dirStyle)
		tb.Done()
		if isSelected {
			dl.AddHighlight(rect, dirStyle.Reverse(true), 1)
		}
		return
	}

	item := detailsNode.item
	itemStyle := styles.styleForStatus(item.status)
	if isSelected {
		itemStyle = itemStyle.Inherit(styles.SelectedBackground)
	}
	if len(indent) > 0 {
		tb.Write(indent)
	}
	d.renderItemContent(tb, item, isSelected, detailsNode.name, itemStyle)
	tb.Done()

	if isSelected {
		dl.AddHighlight(rect, itemStyle, 1)
	}
}

func (d *DetailsList) treeClickMessage(node render.TreeNode, visibleIndex int) render.ClickMessage {
	detailsNode, ok := node.(*detailsTreeNode)
	if !ok || detailsNode == nil {
		return nil
	}
	if detailsNode.item == nil {
		return TreeToggleMsg{VisibleIndex: visibleIndex}
	}
	return FileClickedMsg{Index: detailsNode.fileIndex}
}

// Tree navigation helpers

func (d *DetailsList) treeMoveNext() {
	items := d.treeLastOutput.VisibleItems
	if len(items) == 0 {
		return
	}
	currentIdx := d.treeLastOutput.SelectedIndex
	if currentIdx < 0 {
		nextIdx := render.FirstSelectableIndex(items, nil)
		if nextIdx >= 0 {
			d.treeSelectedID = items[nextIdx].Node.ID()
			d.treeEnsureCursor = true
			d.syncCursorToTree()
		}
		return
	}
	nextIdx := render.NextSelectableIndex(items, currentIdx, nil)
	if nextIdx >= 0 {
		d.treeSelectedID = items[nextIdx].Node.ID()
		d.treeEnsureCursor = true
		d.syncCursorToTree()
	}
}

func (d *DetailsList) treeMovePrev() {
	items := d.treeLastOutput.VisibleItems
	if len(items) == 0 {
		return
	}
	currentIdx := d.treeLastOutput.SelectedIndex
	if currentIdx < 0 {
		prevIdx := render.LastSelectableIndex(items, nil)
		if prevIdx >= 0 {
			d.treeSelectedID = items[prevIdx].Node.ID()
			d.treeEnsureCursor = true
			d.syncCursorToTree()
		}
		return
	}
	prevIdx := render.PrevSelectableIndex(items, currentIdx, nil)
	if prevIdx >= 0 {
		d.treeSelectedID = items[prevIdx].Node.ID()
		d.treeEnsureCursor = true
		d.syncCursorToTree()
	}
}

func (d *DetailsList) treeSelectFirst() {
	items := render.FlattenTree(d.treeRoot, d.treeExpanded, true)
	if len(items) == 0 {
		return
	}
	firstIdx := render.FirstSelectableIndex(items, nil)
	if firstIdx >= 0 {
		d.treeSelectedID = items[firstIdx].Node.ID()
		d.treeEnsureCursor = true
		d.syncCursorToTree()
	}
}

// ToggleTreeExpand toggles the expand state of the tree node at the given visible index.
func (d *DetailsList) ToggleTreeExpand(visibleIndex int) {
	items := d.treeLastOutput.VisibleItems
	if visibleIndex < 0 || visibleIndex >= len(items) {
		return
	}
	node := items[visibleIndex].Node
	if !render.HasChildren(node) {
		return
	}
	render.ToggleExpanded(node.ID(), d.treeExpanded, true)
}

// currentTreeNode returns the currently selected tree node, or nil if none.
func (d *DetailsList) currentTreeNode() *detailsTreeNode {
	if d.treeSelectedID == "" {
		return nil
	}
	idx := render.FindVisibleIndexByID(d.treeLastOutput.VisibleItems, d.treeSelectedID)
	if idx < 0 || idx >= len(d.treeLastOutput.VisibleItems) {
		return nil
	}
	node, ok := d.treeLastOutput.VisibleItems[idx].Node.(*detailsTreeNode)
	if !ok {
		return nil
	}
	return node
}

// IsCurrentNodeDirectory returns true if the current tree node is a directory.
func (d *DetailsList) IsCurrentNodeDirectory() bool {
	node := d.currentTreeNode()
	return node != nil && node.item == nil
}

// IsCurrentNodeExpanded returns true if the current tree node is expanded.
func (d *DetailsList) IsCurrentNodeExpanded() bool {
	node := d.currentTreeNode()
	if node == nil || node.item != nil {
		return false
	}
	return render.IsNodeExpanded(node, d.treeExpanded, true)
}

// ExpandCurrentNode expands the current directory node.
func (d *DetailsList) ExpandCurrentNode() {
	node := d.currentTreeNode()
	if node == nil || node.item != nil {
		return
	}
	if !render.HasChildren(node) {
		return
	}
	d.treeExpanded[node.ID()] = true
}

// CollapseCurrentNode collapses the current directory node.
func (d *DetailsList) CollapseCurrentNode() {
	node := d.currentTreeNode()
	if node == nil || node.item != nil {
		return
	}
	if !render.HasChildren(node) {
		return
	}
	d.treeExpanded[node.ID()] = false
}

// ToggleSelectChildren toggles selection of all items under the current directory.
// Returns the items that were toggled for updating the context.
func (d *DetailsList) ToggleSelectChildren() []*item {
	node := d.currentTreeNode()
	if node == nil || node.item != nil {
		return nil
	}
	items := collectDescendantItems(node)
	if len(items) == 0 {
		return nil
	}

	// Determine new state: if any are unselected, select all; otherwise unselect all
	anyUnselected := false
	for _, it := range items {
		if !it.selected {
			anyUnselected = true
			break
		}
	}

	newState := anyUnselected
	for _, it := range items {
		it.selected = newState
	}
	return items
}

// GetCurrentDirectoryFiles returns all files under the current directory node.
// Returns nil if not on a directory or not in tree view.
func (d *DetailsList) GetCurrentDirectoryFiles() []*item {
	if !d.treeView {
		return nil
	}
	node := d.currentTreeNode()
	if node == nil || node.item != nil {
		return nil
	}
	return collectDescendantItems(node)
}

// SelectCurrentDirectoryFiles marks all files under the current directory as selected.
// Returns the items that were selected, or nil if not on a directory.
// This is used to apply "virtual selection" for operations on directories.
func (d *DetailsList) SelectCurrentDirectoryFiles() []*item {
	items := d.GetCurrentDirectoryFiles()
	if items == nil {
		return nil
	}
	for _, it := range items {
		it.selected = true
	}
	return items
}

// Tree building functions

func buildDetailsTree(items []*item) *detailsTreeNode {
	root := &detailsTreeNode{
		name:      "",
		path:      "",
		fileIndex: -1,
	}

	for i, it := range items {
		parts := strings.Split(it.fileName, "/")
		current := root
		currentPath := ""
		for j, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			if j == len(parts)-1 {
				current.children = append(current.children, &detailsTreeNode{
					name:      part,
					path:      currentPath,
					item:      it,
					fileIndex: i,
				})
			} else {
				found := false
				for _, child := range current.children {
					if child.item == nil && child.name == part {
						current = child
						found = true
						break
					}
				}
				if !found {
					dir := &detailsTreeNode{
						name:      part,
						path:      currentPath,
						fileIndex: -1,
					}
					current.children = append(current.children, dir)
					current = dir
				}
			}
		}
	}

	sortDetailsTree(root)
	collapseDetailsSingleChildDirs(root)
	return root
}

func sortDetailsTree(node *detailsTreeNode) {
	if len(node.children) == 0 {
		return
	}

	sort.SliceStable(node.children, func(i, j int) bool {
		a, b := node.children[i], node.children[j]
		aDir := len(a.children) > 0
		bDir := len(b.children) > 0
		if aDir != bDir {
			return aDir
		}
		return a.name < b.name
	})

	for _, child := range node.children {
		sortDetailsTree(child)
	}
}

func collapseDetailsSingleChildDirs(node *detailsTreeNode) {
	for i, child := range node.children {
		if len(child.children) > 0 && child.item == nil {
			for len(child.children) == 1 && child.children[0].item == nil {
				grandchild := child.children[0]
				child.name = child.name + "/" + grandchild.name
				child.path = grandchild.path
				child.children = grandchild.children
				child.childrenNodes = nil
			}
			node.children[i] = child
			collapseDetailsSingleChildDirs(child)
		}
	}
}

func indexDetailsTreeFiles(node *detailsTreeNode, files map[string]*detailsTreeNode) {
	if node == nil {
		return
	}
	if node.item != nil {
		files[node.item.fileName] = node
		return
	}
	for _, child := range node.children {
		indexDetailsTreeFiles(child, files)
	}
}

// collectDescendantItems collects all file items under a directory node.
func collectDescendantItems(node *detailsTreeNode) []*item {
	if node == nil {
		return nil
	}
	if node.item != nil {
		return []*item{node.item}
	}
	var items []*item
	for _, child := range node.children {
		items = append(items, collectDescendantItems(child)...)
	}
	return items
}
