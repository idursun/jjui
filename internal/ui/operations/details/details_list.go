package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/ops"
)

var _ list.IList = (*DetailsList)(nil)

type DetailsList struct {
	files          []*item
	cursor         int
	renderer       *list.ListRenderer
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
	d.renderer = list.NewRenderer(d)
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
	d.renderer.Reset()
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

func (d *DetailsList) current() *item {
	if len(d.files) == 0 {
		return nil
	}
	return d.files[d.cursor]
}

func (d *DetailsList) GetItemRenderer(index int) list.IItemRenderer {
	item := d.files[index]
	var style lipgloss.Style
	switch item.status {
	case Added:
		style = d.styles.Added
	case Deleted:
		style = d.styles.Deleted
	case Modified:
		style = d.styles.Modified
	case Renamed:
		style = d.styles.Renamed
	case Copied:
		style = d.styles.Copied
	}

	if index == d.cursor {
		style = style.Bold(true).Background(d.styles.Selected.GetBackground())
	} else {
		style = style.Background(d.styles.Text.GetBackground())
	}

	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.selected || (index == d.cursor) {
			hint = d.selectedHint
		}
	}
	r := itemRenderer{
		item:   item,
		styles: d.styles,
		style:  style,
		hint:   hint,
	}
	return r
}

func (d *DetailsList) Len() int {
	return len(d.files)
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	item           *item
	styles         styles
	style          lipgloss.Style
	selectedHint   string
	unselectedHint string
	isChecked      bool
	hint           string
}

func (i itemRenderer) showHint() bool {
	return i.selectedHint != "" || i.unselectedHint != ""
}

func (i itemRenderer) Render(dl *ops.DisplayList, rect cellbuf.Rectangle, _ int) {
	var sb strings.Builder
	title := i.item.Title()
	if i.item.selected {
		title = "✓" + title
	} else {
		title = " " + title
	}

	fmt.Fprint(&sb, i.style.PaddingRight(1).Render(title))
	if i.item.conflict {
		fmt.Fprint(&sb, i.styles.Conflict.Render("conflict "))
	}
	if i.hint != "" {
		fmt.Fprint(&sb, i.styles.Dimmed.Render(i.hint))
	}

	content := sb.String()
	if content != "" {
		dl.AddDraw(rect, content, 0)
	}
}

func (i itemRenderer) Height() int {
	return 1
}
