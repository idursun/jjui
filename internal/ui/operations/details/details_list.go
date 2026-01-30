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
	treeList         *render.TreeList
	treeRoot         *detailsTreeNode
	treeFileNodes    map[string]*detailsTreeNode
	treeView         bool
	selectedHint     string
	unselectedHint   string
	styles           styles
	ensureCursorView bool
}

func NewDetailsList(styles styles) *DetailsList {
	d := &DetailsList{
		files:          []*item{},
		cursor:         -1,
		selectedHint:   "",
		unselectedHint: "",
		styles:         styles,
		treeView:       false,
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
		if d.treeList != nil {
			d.treeList.MoveUp()
			d.treeList.SetEnsureCursorVisible(true)
			d.syncCursorToTree()
		}
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
		if d.treeList != nil {
			d.treeList.MoveDown()
			d.treeList.SetEnsureCursorVisible(true)
			d.syncCursorToTree()
		}
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

// RenderFileList renders the file list to a DisplayContext
func (d *DetailsList) RenderFileList(dl *render.DisplayContext, viewRect layout.Box) {
	if len(d.files) == 0 {
		return
	}
	if d.treeView {
		d.renderTreeList(dl, viewRect)
		return
	}

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
	if d.treeView && d.treeList != nil {
		d.treeList.SetEnsureCursorVisible(false)
		d.treeList.SetScrollOffset(d.treeList.GetScrollOffset() + delta)
		return
	}
	d.ensureCursorView = false
	d.listRenderer.SetScrollOffset(d.listRenderer.GetScrollOffset() + delta)
}

func (d *DetailsList) Len() int {
	if d.treeView && d.treeList != nil {
		return len(d.treeList.VisibleItems())
	}
	return len(d.files)
}

func (d *DetailsList) ToggleTreeView() {
	selectedFile := d.selectedFileName()
	d.treeView = !d.treeView
	if d.treeView {
		if d.treeList != nil {
			d.treeList.SetEnsureCursorVisible(true)
		}
		if selectedFile != "" {
			d.setSelectionByFileName(selectedFile)
		} else if d.treeList != nil {
			d.treeList.SelectFirstSelectable()
			d.syncCursorToTree()
		}
		return
	}
	d.ensureCursorView = true
	if selectedFile != "" {
		d.setSelectionByFileName(selectedFile)
	}
}

func (d *DetailsList) renderTreeList(dl *render.DisplayContext, viewRect layout.Box) {
	if d.treeList == nil {
		return
	}
	d.treeList.ViewRect(dl, viewRect)
}

func (d *DetailsList) renderTreeNode(
	dl *render.DisplayContext,
	node render.TreeNode,
	depth int,
	isSelected bool,
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
		arrow := "▾ "
		if d.treeList != nil && !d.treeList.IsExpanded(node) {
			arrow = "▸ "
		}
		label := indent + arrow + detailsNode.name + "/"
		if len(label) > width {
			label = label[:width]
		}
		tb.Styled(label, d.styles.Dimmed.Background(baseStyle.GetBackground()))
		tb.Done()
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

	if d.treeList == nil {
		d.treeList = render.NewTreeList(render.TreeListConfig{
			Root:            d.treeRoot,
			DefaultExpanded: true,
			ScrollMsg:       FileListScrollMsg{},
			Selectable: func(node render.TreeNode) bool {
				detailsNode, ok := node.(*detailsTreeNode)
				return ok && detailsNode.item != nil
			},
		})
	}

	d.treeList.SetRoot(d.treeRoot)
	d.treeList.SetRenderNode(d.renderTreeNode)
	d.treeList.SetClickMessage(d.treeClickMessage)
	d.treeList.SetEnsureCursorVisible(true)
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
	if d.treeList != nil {
		if node, ok := d.treeFileNodes[fileName]; ok {
			d.treeList.SetEnsureCursorVisible(true)
			d.treeList.SetSelectedByID(node.ID())
		}
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
	if d.treeList == nil {
		return nil
	}
	node, ok := d.treeList.SelectedNode().(*detailsTreeNode)
	if !ok {
		return nil
	}
	return node
}

func (d *DetailsList) syncTreeToCursor() {
	if d.treeList == nil || d.cursor < 0 || d.cursor >= len(d.files) {
		return
	}
	fileName := d.files[d.cursor].fileName
	if node, ok := d.treeFileNodes[fileName]; ok {
		d.treeList.SetEnsureCursorVisible(true)
		d.treeList.SetSelectedByID(node.ID())
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
