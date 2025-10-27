package details

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IList = (*DetailsList)(nil)

type DetailsList struct {
	*common.Sizeable
	files          []*item
	cursor         int
	renderer       *list.ListRenderer
	selectedHint   string
	unselectedHint string
	styles         styles
}

func NewDetailsList(styles styles, size *common.Sizeable) *DetailsList {
	d := &DetailsList{
		Sizeable:       size,
		files:          []*item{},
		cursor:         -1,
		selectedHint:   "",
		unselectedHint: "",
		styles:         styles,
	}
	d.renderer = list.NewRenderer(d, size)
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

func (i itemRenderer) Render(w io.Writer, _ int) {
	title := i.item.Title()
	if i.item.selected {
		title = "✓" + title
	} else {
		title = " " + title
	}

	_, _ = fmt.Fprint(w, i.style.PaddingRight(1).Render(title))
	if i.item.conflict {
		_, _ = fmt.Fprint(w, i.styles.Conflict.Render("conflict "))
	}
	if i.hint != "" {
		_, _ = fmt.Fprint(w, i.styles.Dimmed.Render(i.hint))
	}
	_, _ = fmt.Fprintln(w)
}

func (i itemRenderer) Height() int {
	return 1
}
