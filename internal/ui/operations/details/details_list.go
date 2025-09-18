package details

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type DetailsList struct {
	*list.CheckableList[*models.RevisionFileItem]
	renderer            *list.ListRenderer[*models.RevisionFileItem]
	selectedHint        string
	unselectedHint      string
	isVirtuallySelected bool
	styles              styles
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}

func (d *DetailsList) RenderItem(w io.Writer, index int) {
	item := d.Items[index]
	var style lipgloss.Style
	switch item.Status {
	case models.Added:
		style = d.styles.Added
	case models.Deleted:
		style = d.styles.Deleted
	case models.Modified:
		style = d.styles.Modified
	case models.Renamed:
		style = d.styles.Renamed
	}

	if index == d.Cursor {
		style = style.Bold(true).Background(d.styles.Selected.GetBackground())
	} else {
		style = style.Background(d.styles.Text.GetBackground())
	}

	title := item.Title()
	if item.IsChecked() {
		title = "✓" + title
	} else {
		title = " " + title
	}

	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.IsChecked() || (d.isVirtuallySelected && index == d.Cursor) {
			hint = d.selectedHint
			title = "✓" + item.Title()
		}
	}

	_, _ = fmt.Fprint(w, style.PaddingRight(1).Render(title))
	if item.Conflict {
		_, _ = fmt.Fprint(w, d.styles.Conflict.Render("conflict "))
	}
	if hint != "" {
		_, _ = fmt.Fprint(w, d.styles.Dimmed.Render(hint))
	}
	_, _ = fmt.Fprintln(w)
}

func (d *DetailsList) GetItemHeight(int) int {
	return 1
}
