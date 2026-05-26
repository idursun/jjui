package details

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	detailsObjectFile = "file"
)

type detailsTextObject struct {
	common.FocusedObject
	x     int
	width int
}

type FileClickedMsg struct {
	Index int
	Ctrl  bool
	Alt   bool
}

type FileListScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (f FileListScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	return FileListScrollMsg{Delta: delta, Horizontal: horizontal}
}

type DetailsList struct {
	files              []*item
	cursor             int
	listRenderer       *render.ListRenderer
	selectedHint       string
	unselectedHint     string
	ensureCursorView   bool
	focusedObjectKind  string
	focusedObjectIndex int
}

func NewDetailsList() *DetailsList {
	d := &DetailsList{
		files:          []*item{},
		cursor:         -1,
		selectedHint:   "",
		unselectedHint: "",
	}
	d.listRenderer = render.NewListRenderer(FileListScrollMsg{})
	return d
}

func (d *DetailsList) setItems(files []*item) {
	d.files = files
	if d.cursor >= len(d.files) {
		d.cursor = len(d.files) - 1
	}
	if d.cursor < 0 {
		d.cursor = 0
	}
	if len(d.files) == 0 {
		d.focusedObjectKind = ""
		d.focusedObjectIndex = 0
	} else if d.focusedObjectKind == "" {
		d.focusedObjectKind = detailsObjectFile
		d.focusedObjectIndex = 0
	}
	d.listRenderer.SetScrollOffset(0)
	d.ensureCursorView = true
}

func (d *DetailsList) navigate(delta int, page bool) {
	if d.Len() == 0 {
		return
	}

	// Calculate step (convert page scroll to item count)
	step := delta
	if page {
		firstRowIndex := d.listRenderer.GetFirstRowIndex()
		lastRowIndex := d.listRenderer.GetLastRowIndex()
		span := max(lastRowIndex-firstRowIndex-1, 1)
		if step < 0 {
			step = -span
		} else {
			step = span
		}
	}

	// Calculate new cursor position
	totalItems := len(d.files)
	newCursor := d.cursor + step
	if newCursor < 0 {
		newCursor = 0
	} else if newCursor >= totalItems {
		newCursor = totalItems - 1
	}

	d.setCursor(newCursor)
}

func (d *DetailsList) setCursor(index int) {
	if index >= 0 && index < len(d.files) {
		d.cursor = index
		d.ensureCursorView = true
	}
}

func (d *DetailsList) navigateFocusedObject(delta int) {
	if delta == 0 {
		delta = 1
	}
	objects := d.textObjectsForCurrentItem("", "")
	if len(objects) == 0 {
		d.focusedObjectKind = ""
		d.focusedObjectIndex = 0
		return
	}
	current := d.focusedTextObjectIndex(objects)
	next := (current + delta) % len(objects)
	if next < 0 {
		next += len(objects)
	}
	d.focusedObjectKind = objects[next].Kind
	d.focusedObjectIndex = objects[next].Index
}

func (d *DetailsList) focusedTextObjectIndex(objects []detailsTextObject) int {
	for i := range objects {
		if objects[i].Kind == d.focusedObjectKind && objects[i].Index == d.focusedObjectIndex {
			return i
		}
	}
	for i := range objects {
		if objects[i].Kind == d.focusedObjectKind {
			return i
		}
	}
	return 0
}

func (d *DetailsList) currentFocusedTextObject(changeID, commitID string) *detailsTextObject {
	objects := d.textObjectsForCurrentItem(changeID, commitID)
	if len(objects) == 0 {
		return nil
	}
	idx := d.focusedTextObjectIndex(objects)
	return &objects[idx]
}

func (d *DetailsList) textObjectsForCurrentItem(changeID, commitID string) []detailsTextObject {
	current := d.current()
	if current == nil {
		return nil
	}
	fileX := 3
	return []detailsTextObject{{
		FocusedObject: common.FocusedObject{
			Kind:     detailsObjectFile,
			Value:    current.fileName,
			ChangeId: changeID,
			CommitId: commitID,
			Index:    0,
		},
		x:     fileX,
		width: max(render.StringWidth(current.name), 1),
	}}
}

func (d *DetailsList) hasFocusedObject() bool {
	return d.focusedObjectKind != ""
}

func (d *DetailsList) current() *item {
	if len(d.files) == 0 {
		return nil
	}
	return d.files[d.cursor]
}

// RenderFileList renders the file list to a DisplayContext
func (d *DetailsList) RenderFileList(dl *render.DisplayContext, viewRect layout.Box) {
	if len(d.files) == 0 {
		return
	}

	// Measure function - all items have height 1
	measure := func(index int) int {
		return 1
	}

	textStyle := common.DefaultPalette.Get("revisions details text")

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
		item := d.files[index]
		isSelected := index == d.cursor

		baseStyle := d.getStatusStyle(item.status, isSelected)
		if !isSelected {
			baseStyle = baseStyle.Background(textStyle.GetBackground())
		}
		background := lipgloss.NewStyle().Background(baseStyle.GetBackground())
		dl.AddFill(rect, ' ', background, 0)

		tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
		d.renderItemContent(tb, item, index, baseStyle, isSelected)
		tb.Done()
		if isSelected && d.hasFocusedObject() {
			if obj := d.currentFocusedTextObject("", ""); obj != nil {
				dl.SetCursorAt(detailsObjectCursor(), rect.Min.X+obj.x, rect.Min.Y)
			}
		}
	}

	clickMsg := func(index int, mouse tea.Mouse) render.ClickMessage {
		return FileClickedMsg{
			Index: index,
			Ctrl:  mouse.Mod&tea.ModCtrl != 0,
			Alt:   mouse.Mod&tea.ModAlt != 0,
		}
	}

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
func (d *DetailsList) renderItemContent(tb *render.TextBuilder, item *item, index int, style lipgloss.Style, selected bool) {
	// Build title with checkbox
	title := item.Title()
	if item.selected {
		title = "✓" + title
	} else {
		title = " " + title
	}

	tb.Styled(title, style.PaddingRight(1))

	dimmedStyle := common.DefaultPalette.Get("revisions details dimmed")
	conflictStyle := common.DefaultPalette.Get("revisions details conflict")
	selectedDimmedStyle := common.DefaultPalette.Get("revisions details selected dimmed")

	// Add conflict marker
	if item.conflict {
		conflictMarkerStyle := conflictStyle
		if selected {
			conflictMarkerStyle = common.DefaultPalette.Get("revisions details selected conflict")
		}
		tb.Styled("conflict ", conflictMarkerStyle)
	}

	// Add hint
	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.selected || (!d.hasSelectedItems() && index == d.cursor) {
			hint = d.selectedHint
		}
	}
	if hint != "" {
		hintStyle := dimmedStyle
		if selected {
			hintStyle = selectedDimmedStyle
		}
		tb.Styled(hint, hintStyle)
	}
}

func (d *DetailsList) getStatusStyle(s status, selected bool) lipgloss.Style {
	prefix := "revisions details"
	if selected {
		prefix += " selected"
	}
	switch s {
	case Added:
		return common.DefaultPalette.Get(prefix + " added")
	case Deleted:
		return common.DefaultPalette.Get(prefix + " deleted")
	case Modified:
		return common.DefaultPalette.Get(prefix + " modified")
	case Renamed:
		return common.DefaultPalette.Get(prefix + " renamed")
	case Copied:
		return common.DefaultPalette.Get(prefix + " copied")
	default:
		if selected {
			return common.DefaultPalette.Get("revisions details selected")
		}
		return common.DefaultPalette.Get("revisions details text")
	}
}

// Scroll handles mouse wheel scrolling
func (d *DetailsList) Scroll(delta int) {
	d.ensureCursorView = false
	d.listRenderer.SetScrollOffset(d.listRenderer.GetScrollOffset() + delta)
}

func (d *DetailsList) rangeSelect(from, to int) {
	lo := min(from, to)
	hi := max(from, to)
	for i := lo; i <= hi; i++ {
		if i >= 0 && i < len(d.files) {
			d.files[i].selected = !d.files[i].selected
		}
	}
}

func (d *DetailsList) Len() int {
	if d.files == nil {
		return 0
	}
	return len(d.files)
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}

func (d *DetailsList) hasSelectedItems() bool {
	for _, item := range d.files {
		if item.selected {
			return true
		}
	}
	return false
}
