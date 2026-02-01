package details

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type FileClickedMsg struct {
	Index int
}

type TreeToggleMsg struct {
	VisibleIndex int
}

type FileListScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (f FileListScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	return FileListScrollMsg{Delta: delta, Horizontal: horizontal}
}

type DetailsList struct {
	files            []*item
	cursor           int
	listRenderer     *render.ListRenderer
	treeRoot         *detailsTreeNode
	treeFileNodes    map[string]*detailsTreeNode
	treeView         bool
	selectedHint     string
	unselectedHint   string
	styles           styles
	ensureCursorView bool

	// Tree state (caller-owned for stateless tree rendering)
	treeSelectedID   string
	treeExpanded     map[string]bool
	treeScrollOffset int
	treeEnsureCursor bool
	treeLastOutput   render.TreeOutput
}

func NewDetailsList(styles styles) *DetailsList {
	d := &DetailsList{
		files:          []*item{},
		cursor:         -1,
		selectedHint:   "",
		unselectedHint: "",
		styles:         styles,
		treeView:       false,
		treeExpanded:   make(map[string]bool),
	}
	d.listRenderer = render.NewListRenderer(FileListScrollMsg{})
	return d
}

func (d *DetailsList) setItems(files []*item) {
	selectedFile := d.selectedFileName()
	d.files = files
	if d.cursor >= len(d.files) {
		d.cursor = len(d.files) - 1
	}
	if d.cursor < 0 {
		d.cursor = 0
	}
	d.listRenderer.SetScrollOffset(0)
	d.ensureCursorView = true
	d.buildTree()
	if selectedFile != "" {
		d.setSelectionByFileName(selectedFile)
	}
	d.syncTreeToCursor()
}

func (d *DetailsList) cursorUp() {
	if d.treeView {
		d.treeMovePrev()
		return
	}
	if d.cursor > 0 {
		d.cursor--
		d.ensureCursorView = true
		d.syncTreeToCursor()
	}
}

func (d *DetailsList) cursorDown() {
	if d.treeView {
		d.treeMoveNext()
		return
	}
	if d.cursor < len(d.files)-1 {
		d.cursor++
		d.ensureCursorView = true
		d.syncTreeToCursor()
	}
}

func (d *DetailsList) setCursor(index int) {
	if index >= 0 && index < len(d.files) {
		d.cursor = index
		d.ensureCursorView = true
		d.syncTreeToCursor()
	}
}

func (d *DetailsList) current() *item {
	if len(d.files) == 0 {
		return nil
	}
	if d.treeView {
		node := d.selectedTreeNode()
		if node != nil {
			return node.item
		}
	}
	return d.files[d.cursor]
}

// RenderFileList dispatches to the appropriate renderer based on view mode
func (d *DetailsList) RenderFileList(dl *render.DisplayContext, viewRect layout.Box) {
	if len(d.files) == 0 {
		return
	}
	if d.treeView {
		d.renderTreeView(dl, viewRect)
		return
	}
	d.renderFlatList(dl, viewRect)
}

// renderFlatList renders files as a flat scrollable list
func (d *DetailsList) renderFlatList(dl *render.DisplayContext, viewRect layout.Box) {
	// Measure function - all items have height 1
	measure := func(index int) int {
		return 1
	}

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
		item := d.files[index]
		isSelected := index == d.cursor

		baseStyle := d.getStatusStyle(item.status)
		if isSelected {
			baseStyle = baseStyle.Bold(true).Background(d.styles.Selected.GetBackground())
		} else {
			baseStyle = baseStyle.Background(d.styles.Text.GetBackground())
		}
		background := lipgloss.NewStyle().Background(baseStyle.GetBackground())
		dl.AddFill(rect, ' ', background, 0)

		tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
		d.renderItemContent(tb, item, isSelected, item.name, baseStyle)
		tb.Done()

		// Add highlight for selected item
		if isSelected {
			style := d.getStatusStyle(item.status).Bold(true).Background(d.styles.Selected.GetBackground())
			dl.AddHighlight(rect, style, 1)
		}
	}

	// Click message factory
	clickMsg := func(index int) render.ClickMessage {
		return FileClickedMsg{Index: index}
	}

	// Use the generic list renderer
	d.listRenderer.Render(
		dl,
		viewRect,
		len(d.files),
		d.cursor,
		d.ensureCursorView,
		measure,
		renderItem,
		clickMsg,
	)
	d.listRenderer.RegisterScroll(dl, viewRect)
}

// renderItemContent renders a single item to a string
func (d *DetailsList) renderItemContent(tb *render.TextBuilder, item *item, isCurrent bool, displayName string, style lipgloss.Style) {
	// Build title with checkbox
	title := item.TitleWithName(displayName)
	if item.selected {
		title = "✓" + title
	} else {
		title = " " + title
	}

	tb.Styled(title, style.PaddingRight(1))

	// Add conflict marker
	if item.conflict {
		tb.Styled("conflict ", d.styles.Conflict)
	}

	// Add hint
	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.selected || isCurrent {
			hint = d.selectedHint
		}
	}
	if hint != "" {
		tb.Styled(hint, d.styles.Dimmed)
	}
}

func (d *DetailsList) getStatusStyle(s status) lipgloss.Style {
	switch s {
	case Added:
		return d.styles.Added
	case Deleted:
		return d.styles.Deleted
	case Modified:
		return d.styles.Modified
	case Renamed:
		return d.styles.Renamed
	case Copied:
		return d.styles.Copied
	default:
		return d.styles.Text
	}
}

// Scroll handles mouse wheel scrolling
func (d *DetailsList) Scroll(delta int) {
	if d.treeView {
		d.treeEnsureCursor = false
		d.treeScrollOffset += delta
		return
	}
	d.ensureCursorView = false
	d.listRenderer.SetScrollOffset(d.listRenderer.GetScrollOffset() + delta)
}

func (d *DetailsList) Len() int {
	if d.treeView {
		// Compute flattened tree for accurate count (needed for height calculations before rendering)
		items := render.FlattenTree(d.treeRoot, d.treeExpanded, true)
		return len(items)
	}
	return len(d.files)
}

func (d *DetailsList) ToggleTreeView() {
	selectedFile := d.selectedFileName()
	d.treeView = !d.treeView
	if d.treeView {
		d.treeEnsureCursor = true
		if selectedFile != "" {
			d.setSelectionByFileName(selectedFile)
		} else {
			d.treeSelectFirst()
		}
		return
	}
	d.ensureCursorView = true
	if selectedFile != "" {
		d.setSelectionByFileName(selectedFile)
	}
}

// renderTreeView renders files in a tree structure
func (d *DetailsList) renderTreeView(dl *render.DisplayContext, viewRect layout.Box) {
	if d.treeRoot == nil {
		return
	}
	d.treeLastOutput = render.RenderTree(dl, viewRect, render.TreeInput{
		Root:          d.treeRoot,
		SelectedID:    d.treeSelectedID,
		Expanded:      d.treeExpanded,
		DefaultExpand: true,
		ScrollOffset:  d.treeScrollOffset,
		EnsureCursor:  d.treeEnsureCursor,
		Selectable:    d.isTreeSelectable,
		RenderNode:    d.renderTreeNode,
		ClickMessage:  d.treeClickMessage,
		ScrollMsg:     FileListScrollMsg{},
	})
	d.treeScrollOffset = d.treeLastOutput.ScrollOffset
	d.treeEnsureCursor = false
}

func (d *DetailsList) isTreeSelectable(node render.TreeNode) bool {
	// All nodes are selectable (both files and directories)
	_, ok := node.(*detailsTreeNode)
	return ok
}

func (d *DetailsList) renderTreeNode(
	dl *render.DisplayContext,
	node render.TreeNode,
	depth int,
	isSelected bool,
	isExpanded bool,
	rect cellbuf.Rectangle,
) {
	detailsNode, ok := node.(*detailsTreeNode)
	if !ok || detailsNode == nil {
		return
	}

	width := rect.Dx()
	if width <= 0 {
		return
	}

	baseStyle := d.styles.Text
	if isSelected {
		baseStyle = baseStyle.Background(d.styles.Selected.GetBackground())
	} else {
		baseStyle = baseStyle.Background(d.styles.Text.GetBackground())
	}
	background := lipgloss.NewStyle().Background(baseStyle.GetBackground())
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
		dirStyle := d.styles.Dimmed.Background(baseStyle.GetBackground())
		tb.Styled(label, dirStyle)
		tb.Done()
		if isSelected {
			dl.AddHighlight(rect, dirStyle.Reverse(true), 1)
		}
		return
	}

	item := detailsNode.item
	itemStyle := d.getStatusStyle(item.status)
	if isSelected {
		itemStyle = itemStyle.Bold(true).Background(d.styles.Selected.GetBackground())
	} else {
		itemStyle = itemStyle.Background(d.styles.Text.GetBackground())
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

func (d *DetailsList) buildTree() {
	d.treeRoot = buildDetailsTree(d.files)
	d.treeFileNodes = make(map[string]*detailsTreeNode)
	indexDetailsTreeFiles(d.treeRoot, d.treeFileNodes)
	d.treeScrollOffset = 0
	d.treeEnsureCursor = true
}

func (d *DetailsList) selectedFileName() string {
	if current := d.current(); current != nil {
		return current.fileName
	}
	if d.cursor >= 0 && d.cursor < len(d.files) {
		return d.files[d.cursor].fileName
	}
	return ""
}

func (d *DetailsList) setSelectionByFileName(fileName string) {
	if fileName == "" {
		return
	}
	if idx := d.findFileIndex(fileName); idx >= 0 {
		d.cursor = idx
		d.ensureCursorView = true
	}
	if node, ok := d.treeFileNodes[fileName]; ok {
		d.treeEnsureCursor = true
		render.ExpandPathToNode(d.treeRoot, node.ID(), d.treeExpanded)
		d.treeSelectedID = node.ID()
	}
}

func (d *DetailsList) findFileIndex(fileName string) int {
	for i, file := range d.files {
		if file.fileName == fileName {
			return i
		}
	}
	return -1
}

func (d *DetailsList) selectedTreeNode() *detailsTreeNode {
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

func (d *DetailsList) syncTreeToCursor() {
	if d.cursor < 0 || d.cursor >= len(d.files) {
		return
	}
	fileName := d.files[d.cursor].fileName
	if node, ok := d.treeFileNodes[fileName]; ok {
		d.treeEnsureCursor = true
		render.ExpandPathToNode(d.treeRoot, node.ID(), d.treeExpanded)
		d.treeSelectedID = node.ID()
	}
}

func (d *DetailsList) syncCursorToTree() {
	node := d.selectedTreeNode()
	if node == nil || node.item == nil {
		return
	}
	if node.fileIndex >= 0 && node.fileIndex < len(d.files) {
		d.cursor = node.fileIndex
		d.ensureCursorView = true
	}
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}

// Tree navigation helpers
func (d *DetailsList) treeMoveNext() {
	items := d.treeLastOutput.VisibleItems
	if len(items) == 0 {
		return
	}
	currentIdx := d.treeLastOutput.SelectedIndex
	if currentIdx < 0 {
		nextIdx := render.FirstSelectableIndex(items, d.isTreeSelectable)
		if nextIdx >= 0 {
			d.treeSelectedID = items[nextIdx].Node.ID()
			d.treeEnsureCursor = true
			d.syncCursorToTree()
		}
		return
	}
	nextIdx := render.NextSelectableIndex(items, currentIdx, d.isTreeSelectable)
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
		prevIdx := render.LastSelectableIndex(items, d.isTreeSelectable)
		if prevIdx >= 0 {
			d.treeSelectedID = items[prevIdx].Node.ID()
			d.treeEnsureCursor = true
			d.syncCursorToTree()
		}
		return
	}
	prevIdx := render.PrevSelectableIndex(items, currentIdx, d.isTreeSelectable)
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
	firstIdx := render.FirstSelectableIndex(items, d.isTreeSelectable)
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
