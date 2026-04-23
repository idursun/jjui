package details

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

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
	files            []*item
	cursor           int
	listRenderer     *render.ListRenderer
	selectedHint     string
	unselectedHint   string
	ensureCursorView bool
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

	selectedStyle := common.DefaultPalette.Get("revisions details selected")
	textStyle := common.DefaultPalette.Get("revisions details text")

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
		item := d.files[index]
		isSelected := index == d.cursor

		baseStyle := d.getStatusStyle(item.status)
		if isSelected {
			baseStyle = selectedStyle.Inherit(baseStyle)
		} else {
			baseStyle = baseStyle.Background(textStyle.GetBackground())
		}
		background := lipgloss.NewStyle().Background(baseStyle.GetBackground())
		dl.AddFill(rect, ' ', background, 0)

		tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
		d.renderItemContent(tb, item, index, baseStyle, isSelected)
		tb.Done()
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
	selectedStyle := common.DefaultPalette.Get("revisions details selected")

	// Add conflict marker
	if item.conflict {
		conflictStyle := conflictStyle
		if selected {
			conflictStyle = selectedStyle.Inherit(conflictStyle)
		}
		tb.Styled("conflict ", conflictStyle)
	}

	// Add hint
	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.selected || (index == d.cursor) {
			hint = d.selectedHint
		}
	}
	if hint != "" {
		hintStyle := dimmedStyle
		if selected {
			hintStyle = selectedStyle.Inherit(hintStyle)
		}
		tb.Styled(hint, hintStyle)
	}
}

func (d *DetailsList) getStatusStyle(s status) lipgloss.Style {
	addedStyle := common.DefaultPalette.Get("revisions details added")
	deletedStyle := common.DefaultPalette.Get("revisions details deleted")
	modifiedStyle := common.DefaultPalette.Get("revisions details modified")
	renamedStyle := common.DefaultPalette.Get("revisions details renamed")
	copiedStyle := common.DefaultPalette.Get("revisions details copied")
	textStyle := common.DefaultPalette.Get("revisions details text")

	switch s {
	case Added:
		return addedStyle
	case Deleted:
		return deletedStyle
	case Modified:
		return modifiedStyle
	case Renamed:
		return renamedStyle
	case Copied:
		return copiedStyle
	default:
		return textStyle
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
