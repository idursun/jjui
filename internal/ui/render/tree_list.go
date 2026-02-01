package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
)

// TreeNode represents a node in a tree structure.
type TreeNode interface {
	ID() string
	Name() string
	Children() []TreeNode
}

// TreeVisibleItem represents a visible node in a flattened tree.
type TreeVisibleItem struct {
	Node  TreeNode
	Depth int
}

// TreeRenderFunc renders a single tree node.
// isExpanded indicates whether the node is currently expanded (only meaningful for nodes with children).
type TreeRenderFunc func(dl *DisplayContext, node TreeNode, depth int, isSelected bool, isExpanded bool, rect cellbuf.Rectangle)

// TreeClickFunc returns a message when a tree node is clicked.
type TreeClickFunc func(node TreeNode, visibleIndex int) tea.Msg

// TreeInput holds all the state and configuration needed to render a tree.
// The caller owns this and passes it to the render function.
type TreeInput struct {
	// Tree structure
	Root TreeNode

	// State (caller-owned)
	SelectedID    string          // ID of the selected node
	Expanded      map[string]bool // node ID → expanded state
	DefaultExpand bool            // default for nodes not in Expanded map
	ScrollOffset  int
	EnsureCursor  bool // scroll to make selection visible

	// Callbacks
	Selectable   func(TreeNode) bool
	RenderNode   TreeRenderFunc
	ClickMessage TreeClickFunc
	ScrollMsg    tea.Msg
}

// TreeOutput contains computed information after rendering.
// Caller uses this to update their state.
type TreeOutput struct {
	VisibleItems  []TreeVisibleItem
	SelectedIndex int // index in VisibleItems, -1 if none
	ScrollOffset  int // adjusted scroll offset (if EnsureCursor was true)
}

// RenderTree flattens and renders a tree in one call.
// Returns output that caller uses to update their state.
func RenderTree(
	dl *DisplayContext,
	box layout.Box,
	input TreeInput,
) TreeOutput {
	output := TreeOutput{
		SelectedIndex: -1,
		ScrollOffset:  input.ScrollOffset,
	}

	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return output
	}

	emptyStyle := lipgloss.NewStyle()
	dl.AddFill(cellbuf.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), box.R.Dy()), ' ', emptyStyle, 0)

	// Flatten the tree
	output.VisibleItems = FlattenTree(input.Root, input.Expanded, input.DefaultExpand)

	if len(output.VisibleItems) == 0 || input.RenderNode == nil {
		return output
	}

	// Find selected index
	output.SelectedIndex = FindVisibleIndexByID(output.VisibleItems, input.SelectedID)

	clickMsg := input.ClickMessage
	if clickMsg == nil {
		clickMsg = func(node TreeNode, visibleIndex int) tea.Msg {
			return nil
		}
	}

	// Create a temporary list renderer for scroll management and rendering
	listRenderer := &ListRenderer{
		StartLine: input.ScrollOffset,
		ScrollMsg: input.ScrollMsg,
	}

	listRenderer.Render(
		dl,
		box,
		len(output.VisibleItems),
		output.SelectedIndex,
		input.EnsureCursor,
		func(index int) int {
			return 1
		},
		func(dl *DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= len(output.VisibleItems) {
				return
			}
			item := output.VisibleItems[index]
			isExpanded := IsNodeExpanded(item.Node, input.Expanded, input.DefaultExpand)
			input.RenderNode(dl, item.Node, item.Depth, index == output.SelectedIndex, isExpanded, rect)
		},
		func(index int) tea.Msg {
			if index < 0 || index >= len(output.VisibleItems) {
				return nil
			}
			return clickMsg(output.VisibleItems[index].Node, index)
		},
	)

	listRenderer.RegisterScroll(dl, box)
	output.ScrollOffset = listRenderer.GetScrollOffset()

	return output
}

// FlattenTree flattens a tree into a slice of visible items.
func FlattenTree(root TreeNode, expanded map[string]bool, defaultExpand bool) []TreeVisibleItem {
	if root == nil {
		return nil
	}
	var items []TreeVisibleItem
	flattenNode(root, -1, expanded, defaultExpand, &items)
	return items
}

func flattenNode(node TreeNode, depth int, expanded map[string]bool, defaultExpand bool, items *[]TreeVisibleItem) {
	if depth >= 0 {
		*items = append(*items, TreeVisibleItem{Node: node, Depth: depth})
	}
	if hasChildren(node) && (depth < 0 || IsNodeExpanded(node, expanded, defaultExpand)) {
		for _, child := range node.Children() {
			flattenNode(child, depth+1, expanded, defaultExpand, items)
		}
	}
}

// IsNodeExpanded returns whether a node is expanded.
func IsNodeExpanded(node TreeNode, expanded map[string]bool, defaultExpand bool) bool {
	if node == nil || !hasChildren(node) {
		return false
	}
	if expanded != nil {
		if exp, ok := expanded[node.ID()]; ok {
			return exp
		}
	}
	return defaultExpand
}

// HasChildren returns whether a node has children.
func HasChildren(node TreeNode) bool {
	return hasChildren(node)
}

func hasChildren(node TreeNode) bool {
	if node == nil {
		return false
	}
	return len(node.Children()) > 0
}

// FindVisibleIndexByID finds the index of a node by its ID in the visible items.
func FindVisibleIndexByID(items []TreeVisibleItem, id string) int {
	if id == "" {
		return -1
	}
	for i, item := range items {
		if item.Node.ID() == id {
			return i
		}
	}
	return -1
}

// NextSelectableIndex finds the next selectable item after the given index.
func NextSelectableIndex(items []TreeVisibleItem, from int, selectable func(TreeNode) bool) int {
	for i := from + 1; i < len(items); i++ {
		if isSelectable(items[i].Node, selectable) {
			return i
		}
	}
	return -1
}

// PrevSelectableIndex finds the previous selectable item before the given index.
func PrevSelectableIndex(items []TreeVisibleItem, from int, selectable func(TreeNode) bool) int {
	for i := from - 1; i >= 0; i-- {
		if isSelectable(items[i].Node, selectable) {
			return i
		}
	}
	return -1
}

// FirstSelectableIndex finds the first selectable item.
func FirstSelectableIndex(items []TreeVisibleItem, selectable func(TreeNode) bool) int {
	for i, item := range items {
		if isSelectable(item.Node, selectable) {
			return i
		}
	}
	return -1
}

// LastSelectableIndex finds the last selectable item.
func LastSelectableIndex(items []TreeVisibleItem, selectable func(TreeNode) bool) int {
	for i := len(items) - 1; i >= 0; i-- {
		if isSelectable(items[i].Node, selectable) {
			return i
		}
	}
	return -1
}

// NearestSelectableIndex finds the nearest selectable item to the given index.
func NearestSelectableIndex(items []TreeVisibleItem, from int, selectable func(TreeNode) bool) int {
	if len(items) == 0 {
		return -1
	}
	if from >= len(items) {
		from = len(items) - 1
	}
	if from < 0 {
		from = 0
	}
	for i := from; i < len(items); i++ {
		if isSelectable(items[i].Node, selectable) {
			return i
		}
	}
	for i := from - 1; i >= 0; i-- {
		if isSelectable(items[i].Node, selectable) {
			return i
		}
	}
	return -1
}

func isSelectable(node TreeNode, selectable func(TreeNode) bool) bool {
	if node == nil {
		return false
	}
	if selectable == nil {
		return true
	}
	return selectable(node)
}

// ExpandPathToNode expands all ancestors of targetID in the expanded map.
// Returns true if the target was found.
func ExpandPathToNode(root TreeNode, targetID string, expanded map[string]bool) bool {
	if root == nil || targetID == "" {
		return false
	}
	return expandPath(root, targetID, expanded)
}

func expandPath(node TreeNode, targetID string, expanded map[string]bool) bool {
	if node == nil {
		return false
	}
	if node.ID() == targetID {
		return true
	}
	if !hasChildren(node) {
		return false
	}
	for _, child := range node.Children() {
		if expandPath(child, targetID, expanded) {
			if expanded == nil {
				return true
			}
			expanded[node.ID()] = true
			return true
		}
	}
	return false
}

// ToggleExpanded toggles the expanded state of a node in the expanded map.
// Returns the new expanded state.
func ToggleExpanded(nodeID string, expanded map[string]bool, defaultExpand bool) bool {
	if expanded == nil {
		return defaultExpand
	}
	current, ok := expanded[nodeID]
	if !ok {
		current = defaultExpand
	}
	expanded[nodeID] = !current
	return !current
}
