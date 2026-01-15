package details

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// FileClickedMsg is sent when a file item is clicked
type FileClickedMsg struct {
	Index int
}

// FileListScrollMsg is sent when the file list is scrolled via mouse wheel
type FileListScrollMsg struct {
	Delta int
}

// SetDelta implements render.ScrollDeltaCarrier
func (f FileListScrollMsg) SetDelta(delta int) tea.Msg {
	return FileListScrollMsg{Delta: delta}
}

type DetailsList struct {
	files          []*item
	cursor         int
	listRenderer   *render.ListRenderer
	selectedHint   string
	unselectedHint string
	styles         styles
}

func NewDetailsList(styles styles) *DetailsList {
	d := &DetailsList{
		files:          []*item{},
		cursor:         -1,
		selectedHint:   "",
		unselectedHint: "",
		styles:         styles,
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
}

func (d *DetailsList) cursorUp() {
	if d.cursor > 0 {
		d.cursor--
	}
}

func (d *DetailsList) cursorDown() {
	if d.cursor < len(d.files)-1 {
		d.cursor++
	}
}

func (d *DetailsList) setCursor(index int) {
	if index >= 0 && index < len(d.files) {
		d.cursor = index
	}
}

func (d *DetailsList) current() *item {
	if len(d.files) == 0 {
		return nil
	}
	return d.files[d.cursor]
}

// RenderFileList renders the file list to a DisplayContext
func (d *DetailsList) RenderFileList(dl *render.DisplayContext, viewRect layout.Box, screenOffset cellbuf.Position) {
	if len(d.files) == 0 {
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

		// Build the content string
		content := d.renderItemContent(item, index, rect.Dx())

		// Add draw for the item
		dl.AddDraw(rect, content, 0)

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

	// Use the generic list renderer with screen offset for interactions
	d.listRenderer.RenderWithOffset(
		dl,
		viewRect,
		len(d.files),
		d.cursor,
		true, // ensureCursorVisible
		measure,
		renderItem,
		clickMsg,
		screenOffset,
	)
}

// renderItemContent renders a single item to a string
func (d *DetailsList) renderItemContent(item *item, index int, width int) string {
	var result strings.Builder

	// Get style based on status
	style := d.getStatusStyle(item.status)
	if index == d.cursor {
		style = style.Bold(true).Background(d.styles.Selected.GetBackground())
	} else {
		style = style.Background(d.styles.Text.GetBackground())
	}

	// Build title with checkbox
	title := item.Title()
	if item.selected {
		title = "✓" + title
	} else {
		title = " " + title
	}

	result.WriteString(style.PaddingRight(1).Render(title))

	// Add conflict marker
	if item.conflict {
		result.WriteString(d.styles.Conflict.Render("conflict "))
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
		result.WriteString(d.styles.Dimmed.Render(hint))
	}

	// Pad to width
	content := result.String()
	if lipgloss.Width(content) < width {
		content = lipgloss.PlaceHorizontal(width, lipgloss.Left, content)
	}

	return content
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
	d.listRenderer.SetScrollOffset(d.listRenderer.GetScrollOffset() + delta)
}

func (d *DetailsList) Len() int {
	return len(d.files)
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}
