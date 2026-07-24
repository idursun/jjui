package details

import (
	"strings"
	"unicode/utf8"

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

type fileMatch struct {
	index int
	start int
	end   int
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
	filtering        bool
	filterQuery      string
	matches          []fileMatch
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
	currentFile := ""
	if current := d.current(); current != nil {
		currentFile = current.fileName
	}
	d.files = files
	d.rebuildMatches(currentFile)
	d.listRenderer.SetScrollOffset(0)
	d.ensureCursorView = true
}

func (d *DetailsList) setFilter(query string, enabled bool) {
	currentFile := ""
	if current := d.current(); current != nil {
		currentFile = current.fileName
	}
	d.filtering = enabled
	d.filterQuery = query
	d.rebuildMatches(currentFile)
	d.listRenderer.SetScrollOffset(0)
	d.ensureCursorView = true
}

func (d *DetailsList) rebuildMatches(preferredFile string) {
	if d.filtering {
		d.matches = nil
		query := strings.TrimSpace(d.filterQuery)
		for index, item := range d.files {
			start, end, matched := findSubstringFold(item.name, query)
			if matched {
				d.matches = append(d.matches, fileMatch{index: index, start: start, end: end})
			}
		}
	} else {
		d.matches = nil
	}

	visibleLen := d.VisibleLen()
	if visibleLen == 0 {
		d.cursor = -1
		return
	}
	if preferredFile != "" {
		for index := range visibleLen {
			if candidate := d.itemAt(index); candidate != nil && candidate.fileName == preferredFile {
				d.cursor = index
				return
			}
		}
		d.cursor = 0
		return
	}
	if d.cursor < 0 {
		d.cursor = 0
	} else if d.cursor >= visibleLen {
		d.cursor = visibleLen - 1
	}
}

func (d *DetailsList) navigate(delta int, page bool) {
	if d.VisibleLen() == 0 {
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
	totalItems := d.VisibleLen()
	newCursor := d.cursor + step
	if newCursor < 0 {
		newCursor = 0
	} else if newCursor >= totalItems {
		newCursor = totalItems - 1
	}

	d.setCursor(newCursor)
}

func (d *DetailsList) setCursor(index int) {
	if index >= 0 && index < d.VisibleLen() {
		d.cursor = index
		d.ensureCursorView = true
	}
}

func (d *DetailsList) current() *item {
	return d.itemAt(d.cursor)
}

func (d *DetailsList) itemAt(index int) *item {
	sourceIndex, ok := d.sourceIndex(index)
	if !ok {
		return nil
	}
	return d.files[sourceIndex]
}

func (d *DetailsList) sourceIndex(index int) (int, bool) {
	if index < 0 || index >= d.VisibleLen() {
		return 0, false
	}
	if d.filtering {
		return d.matches[index].index, true
	}
	return index, true
}

// RenderFileList renders the file list to a DisplayContext
func (d *DetailsList) RenderFileList(dl *render.DisplayContext, viewRect layout.Box) {
	if d.VisibleLen() == 0 {
		return
	}

	// Measure function - all items have height 1
	measure := func(index int) int {
		return 1
	}

	textStyle := common.DefaultPalette.Get("revisions", "details", "text", false)

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
		item := d.itemAt(index)
		if item == nil {
			return
		}
		isSelected := index == d.cursor

		baseStyle := d.getStatusStyle(item.status, isSelected)
		if !isSelected {
			baseStyle = baseStyle.Background(textStyle.GetBackground())
		}
		background := lipgloss.NewStyle().Background(baseStyle.GetBackground())
		dl.AddFill(rect, ' ', background, 0)

		tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
		var match *fileMatch
		if d.filtering {
			match = &d.matches[index]
		}
		d.renderItemContent(tb, item, index, match, baseStyle, isSelected)
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
		d.VisibleLen(),
		d.cursor,
		d.ensureCursorView,
		measure,
		renderItem,
		clickMsg,
	)
	d.listRenderer.RegisterScroll(dl, viewRect)
}

// renderItemContent renders a single item to a string
func (d *DetailsList) renderItemContent(tb *render.TextBuilder, item *item, index int, match *fileMatch, style lipgloss.Style, selected bool) {
	// Build title with checkbox
	title := item.Title()
	if item.selected {
		tb.Styled("✓", style)
	} else {
		tb.Styled(" ", style)
	}
	if match == nil || match.start == match.end {
		tb.Styled(title, style)
	} else {
		offset := len(title) - len(item.name)
		start := offset + match.start
		end := offset + match.end
		matchStyle := common.DefaultPalette.Get("revisions", "details", "matched", selected)
		tb.Styled(title[:start], style)
		tb.Styled(title[start:end], matchStyle)
		tb.Styled(title[end:], style)
	}
	tb.Styled(" ", style)

	// Add conflict marker
	if item.conflict {
		conflictMarkerStyle := common.DefaultPalette.Get("revisions", "details", "conflict", selected)
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
		hintStyle := common.DefaultPalette.Get("revisions", "details", "dimmed", selected)
		tb.Styled(hint, hintStyle)
	}
}

func (d *DetailsList) getStatusStyle(s status, selected bool) lipgloss.Style {
	role := "text"
	switch s {
	case Added:
		role = "added"
	case Deleted:
		role = "deleted"
	case Modified:
		role = "modified"
	case Renamed:
		role = "renamed"
	case Copied:
		role = "copied"
	}
	return common.DefaultPalette.Get("revisions", "details", role, selected)
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
		if item := d.itemAt(i); item != nil {
			item.selected = !item.selected
		}
	}
}

func (d *DetailsList) Len() int {
	if d.files == nil {
		return 0
	}
	return len(d.files)
}

func (d *DetailsList) VisibleLen() int {
	if d.filtering {
		return len(d.matches)
	}
	return d.Len()
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}

func findSubstringFold(candidate, query string) (int, int, bool) {
	if query == "" {
		return 0, 0, true
	}

	queryRunes := utf8.RuneCountInString(query)
	offsets := make([]int, 0, utf8.RuneCountInString(candidate)+1)
	for offset := range candidate {
		offsets = append(offsets, offset)
	}
	offsets = append(offsets, len(candidate))
	for start := 0; start+queryRunes < len(offsets); start++ {
		startByte := offsets[start]
		endByte := offsets[start+queryRunes]
		if strings.EqualFold(candidate[startByte:endByte], query) {
			return startByte, endByte, true
		}
	}
	return 0, 0, false
}

func (d *DetailsList) hasSelectedItems() bool {
	for _, item := range d.files {
		if item.selected {
			return true
		}
	}
	return false
}
