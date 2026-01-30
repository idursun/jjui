package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
)

type TreeNode interface {
	ID() string
	Name() string
	Children() []TreeNode
}

type TreeVisibleItem struct {
	Node  TreeNode
	Depth int
}

type TreeRenderFunc func(dl *DisplayContext, node TreeNode, depth int, isSelected bool, rect cellbuf.Rectangle)

type TreeClickFunc func(node TreeNode, visibleIndex int) tea.Msg

type TreeListConfig struct {
	Root            TreeNode
	ScrollMsg       tea.Msg
	DefaultExpanded bool
	Expanded        map[string]bool
	Selectable      func(TreeNode) bool
	RenderNode      TreeRenderFunc
	ClickMessage    TreeClickFunc
	AutoSelect      bool
}

type TreeList struct {
	root            TreeNode
	visible         []TreeVisibleItem
	selectedIndex   int
	listRenderer    *ListRenderer
	defaultExpanded bool
	expanded        map[string]bool
	selectable      func(TreeNode) bool
	renderNode      TreeRenderFunc
	clickMessage    TreeClickFunc
	ensureCursor    bool
}

func NewTreeList(config TreeListConfig) *TreeList {
	tl := &TreeList{
		root:            config.Root,
		selectedIndex:   -1,
		listRenderer:    NewListRenderer(config.ScrollMsg),
		defaultExpanded: config.DefaultExpanded,
		expanded:        config.Expanded,
		selectable:      config.Selectable,
		renderNode:      config.RenderNode,
		clickMessage:    config.ClickMessage,
		ensureCursor:    true,
	}
	tl.flatten()
	if config.AutoSelect {
		tl.SelectFirstSelectable()
	}
	return tl
}

func (t *TreeList) SetRoot(root TreeNode) {
	t.root = root
	t.flatten()
}

func (t *TreeList) SetRenderNode(renderNode TreeRenderFunc) {
	t.renderNode = renderNode
}

func (t *TreeList) SetClickMessage(clickMessage TreeClickFunc) {
	t.clickMessage = clickMessage
}

func (t *TreeList) SetSelectable(selectable func(TreeNode) bool) {
	t.selectable = selectable
}

func (t *TreeList) SetEnsureCursorVisible(ensure bool) {
	t.ensureCursor = ensure
}

func (t *TreeList) SetScrollOffset(offset int) {
	t.listRenderer.SetScrollOffset(offset)
}

func (t *TreeList) GetScrollOffset() int {
	return t.listRenderer.GetScrollOffset()
}

func (t *TreeList) VisibleItems() []TreeVisibleItem {
	return t.visible
}

func (t *TreeList) SelectedVisibleIndex() int {
	return t.selectedIndex
}

func (t *TreeList) SelectedNode() TreeNode {
	if t.selectedIndex >= 0 && t.selectedIndex < len(t.visible) {
		return t.visible[t.selectedIndex].Node
	}
	return nil
}

func (t *TreeList) SelectedID() string {
	node := t.SelectedNode()
	if node == nil {
		return ""
	}
	return node.ID()
}

func (t *TreeList) SelectFirstSelectable() {
	t.selectedIndex = t.firstSelectableIndex()
}

func (t *TreeList) SetSelectedByID(id string) {
	if id == "" {
		return
	}
	if t.root == nil {
		return
	}
	t.expandPathToID(t.root, id)
	t.flatten()
	if idx := t.findVisibleIndexByID(id); idx >= 0 {
		if t.isSelectable(t.visible[idx].Node) {
			t.selectedIndex = idx
		} else {
			t.selectedIndex = t.nearestSelectableIndex(idx)
		}
	}
}

func (t *TreeList) MoveUp() {
	if len(t.visible) == 0 {
		return
	}
	if t.selectedIndex < 0 {
		t.selectedIndex = t.lastSelectableIndex()
		return
	}
	prev := t.prevSelectableIndex(t.selectedIndex)
	if prev >= 0 {
		t.selectedIndex = prev
	}
}

func (t *TreeList) MoveDown() {
	if len(t.visible) == 0 {
		return
	}
	if t.selectedIndex < 0 {
		t.selectedIndex = t.firstSelectableIndex()
		return
	}
	next := t.nextSelectableIndex(t.selectedIndex)
	if next >= 0 {
		t.selectedIndex = next
	}
}

func (t *TreeList) ToggleExpand(visibleIndex int) {
	if visibleIndex < 0 || visibleIndex >= len(t.visible) {
		return
	}
	node := t.visible[visibleIndex].Node
	if !t.hasChildren(node) {
		return
	}

	currentSelectedID := t.SelectedID()
	t.setExpanded(node, !t.isExpanded(node))
	t.flatten()

	if currentSelectedID != "" {
		if idx := t.findVisibleIndexByID(currentSelectedID); idx >= 0 {
			if t.isSelectable(t.visible[idx].Node) {
				t.selectedIndex = idx
				return
			}
		}
	}
	t.selectedIndex = t.nearestSelectableIndex(visibleIndex)
}

func (t *TreeList) IsExpanded(node TreeNode) bool {
	return t.isExpanded(node)
}

func (t *TreeList) ViewRect(dl *DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	emptyStyle := lipgloss.NewStyle()
	dl.AddFill(cellbuf.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), box.R.Dy()), ' ', emptyStyle, 0)

	if len(t.visible) == 0 || t.renderNode == nil {
		return
	}

	clickMsg := t.clickMessage
	if clickMsg == nil {
		clickMsg = func(node TreeNode, visibleIndex int) tea.Msg {
			return nil
		}
	}

	t.listRenderer.Render(
		dl,
		box,
		len(t.visible),
		t.selectedIndex,
		t.ensureCursor,
		func(index int) int {
			return 1
		},
		func(dl *DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= len(t.visible) {
				return
			}
			item := t.visible[index]
			t.renderNode(dl, item.Node, item.Depth, index == t.selectedIndex, rect)
		},
		func(index int) tea.Msg {
			if index < 0 || index >= len(t.visible) {
				return nil
			}
			return clickMsg(t.visible[index].Node, index)
		},
	)

	t.listRenderer.RegisterScroll(dl, box)
}

func (t *TreeList) flatten() {
	t.visible = nil
	if t.root == nil {
		t.selectedIndex = -1
		return
	}
	t.flattenNode(t.root, -1)
	if t.selectedIndex >= len(t.visible) {
		t.selectedIndex = t.firstSelectableIndex()
	}
}

func (t *TreeList) flattenNode(node TreeNode, depth int) {
	if depth >= 0 {
		t.visible = append(t.visible, TreeVisibleItem{Node: node, Depth: depth})
	}
	if t.hasChildren(node) && (depth < 0 || t.isExpanded(node)) {
		for _, child := range node.Children() {
			t.flattenNode(child, depth+1)
		}
	}
}

func (t *TreeList) isExpanded(node TreeNode) bool {
	if node == nil || !t.hasChildren(node) {
		return false
	}
	if t.expanded != nil {
		if expanded, ok := t.expanded[node.ID()]; ok {
			return expanded
		}
	}
	return t.defaultExpanded
}

func (t *TreeList) setExpanded(node TreeNode, expanded bool) {
	if node == nil || !t.hasChildren(node) {
		return
	}
	if t.expanded == nil {
		t.expanded = make(map[string]bool)
	}
	t.expanded[node.ID()] = expanded
}

func (t *TreeList) expandPathToID(node TreeNode, id string) bool {
	if node == nil {
		return false
	}
	if node.ID() == id {
		return true
	}
	if !t.hasChildren(node) {
		return false
	}
	for _, child := range node.Children() {
		if t.expandPathToID(child, id) {
			t.setExpanded(node, true)
			return true
		}
	}
	return false
}

func (t *TreeList) isSelectable(node TreeNode) bool {
	if node == nil {
		return false
	}
	if t.selectable == nil {
		return true
	}
	return t.selectable(node)
}

func (t *TreeList) hasChildren(node TreeNode) bool {
	if node == nil {
		return false
	}
	return len(node.Children()) > 0
}

func (t *TreeList) firstSelectableIndex() int {
	for i, item := range t.visible {
		if t.isSelectable(item.Node) {
			return i
		}
	}
	return -1
}

func (t *TreeList) lastSelectableIndex() int {
	for i := len(t.visible) - 1; i >= 0; i-- {
		if t.isSelectable(t.visible[i].Node) {
			return i
		}
	}
	return -1
}

func (t *TreeList) nextSelectableIndex(from int) int {
	for i := from + 1; i < len(t.visible); i++ {
		if t.isSelectable(t.visible[i].Node) {
			return i
		}
	}
	return -1
}

func (t *TreeList) prevSelectableIndex(from int) int {
	for i := from - 1; i >= 0; i-- {
		if t.isSelectable(t.visible[i].Node) {
			return i
		}
	}
	return -1
}

func (t *TreeList) nearestSelectableIndex(from int) int {
	if len(t.visible) == 0 {
		return -1
	}
	if from >= len(t.visible) {
		from = len(t.visible) - 1
	}
	if from < 0 {
		from = 0
	}
	for i := from; i < len(t.visible); i++ {
		if t.isSelectable(t.visible[i].Node) {
			return i
		}
	}
	for i := from - 1; i >= 0; i-- {
		if t.isSelectable(t.visible[i].Node) {
			return i
		}
	}
	return -1
}

func (t *TreeList) findVisibleIndexByID(id string) int {
	for i, item := range t.visible {
		if item.Node.ID() == id {
			return i
		}
	}
	return -1
}
