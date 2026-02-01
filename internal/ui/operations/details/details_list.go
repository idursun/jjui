package details

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		node := d.currentTreeNode()
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
	node := d.currentTreeNode()
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
