package bookmark

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

var _ operations.Operation = (*RenameBookmarkOperation)(nil)
var _ common.Editable = (*RenameBookmarkOperation)(nil)
var _ common.ScopeProvider = (*RenameBookmarkOperation)(nil)
var _ operations.SegmentRenderer = (*RenameBookmarkOperation)(nil)

// RenameBookmarkOperation edits a bookmark name in place: the local bookmark
// name on its revision is replaced by the input.
type RenameBookmarkOperation struct {
	context  *context.MainContext
	revision string
	oldName  string
	name     textinput.Model
}

func NewRenameBookmarkOperation(context *context.MainContext, changeId string, oldName string) *RenameBookmarkOperation {
	t := textinput.New()
	t.CharLimit = 120
	t.Prompt = ""
	t.SetVirtualCursor(false)
	t.SetValue(oldName)
	t.CursorEnd()
	t.Focus()

	return &RenameBookmarkOperation{
		context:  context,
		revision: changeId,
		oldName:  oldName,
		name:     t,
	}
}

func (r *RenameBookmarkOperation) IsEditing() bool {
	return true
}

func (r *RenameBookmarkOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeSetBookmark,
			Leak:    common.LeakNone,
			Handler: r,
		},
	}
}

func (r *RenameBookmarkOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		newName := strings.TrimSpace(r.name.Value())
		if newName == "" || newName == r.oldName {
			return common.Close, true
		}
		return r.context.RunCommand(jj.BookmarkRename(r.oldName, newName), common.CloseApplied, common.Refresh), true
	case intents.AutocompleteCycle:
		return nil, true
	}
	return nil, false
}

func (r *RenameBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	if intent, ok := msg.(intents.Intent); ok {
		cmd, _ := r.HandleIntent(intent)
		return cmd
	}
	var cmd tea.Cmd
	r.name, cmd = r.name.Update(msg)
	r.name.SetValue(strings.ReplaceAll(r.name.Value(), " ", "-"))
	return cmd
}

func (r *RenameBookmarkOperation) Init() tea.Cmd {
	return textinput.Blink
}

func (r *RenameBookmarkOperation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := r.viewContent()
	w, h := lipgloss.Size(content)
	rect := layout.Rect(box.R.Min.X, box.R.Min.Y, w, h)
	dl.AddDraw(rect, content, 0)
	dl.SetCursorInRect(r.name.Cursor(), rect, 0, 0)
}

func (r *RenameBookmarkOperation) IsFocused() bool {
	return true
}

func (r *RenameBookmarkOperation) Render(*jj.Commit, operations.RenderPosition) string {
	return ""
}

func (r *RenameBookmarkOperation) RenderSegment(style lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	if row.Commit == nil || row.Commit.GetChangeId() != r.revision || segment != localBookmarkSegment(row, r.oldName) {
		return ""
	}
	content := r.styledContent(style)
	if content == "" {
		// An emptied name still needs a cell for the cursor.
		return style.Render(" ")
	}
	return content
}

func (r *RenameBookmarkOperation) InlineCursor(commit *jj.Commit, pos operations.RenderPosition) *tea.Cursor {
	if pos != operations.RenderOverSegment || commit.GetChangeId() != r.revision {
		return nil
	}
	return r.name.Cursor()
}

// localBookmarkSegment finds the segment naming the local bookmark. A remote
// bookmark has the same name followed by "@".
func localBookmarkSegment(row parser.Row, name string) *screen.Segment {
	for _, line := range row.Lines {
		for i, segment := range line.Segments {
			if jj.UnquoteRefName(segment.Text) != name || !segment.HasLabel("name") || segment.HasLabel("remote_bookmarks") {
				continue
			}
			if !segment.HasLabel("bookmarks") && !segment.HasLabel("local_bookmarks") {
				continue
			}
			if i+1 < len(line.Segments) && line.Segments[i+1].Text == "@" {
				continue
			}
			return segment
		}
	}
	return nil
}

func (r *RenameBookmarkOperation) Name() string {
	return "rename bookmark"
}

func (r *RenameBookmarkOperation) viewContent() string {
	return r.styledContent(theme.DefaultPalette.Get("revisions", "", "text", false))
}

// styledContent renders the input with the style of the text it replaces.
func (r *RenameBookmarkOperation) styledContent(style lipgloss.Style) string {
	dimmedStyle := theme.DefaultPalette.Get("revisions", "", "dimmed", false).Inline(true)
	textStyle := style.Inline(true)
	styles := r.name.Styles()
	styles.Focused.Text = textStyle
	styles.Focused.Prompt = textStyle
	styles.Focused.Suggestion = dimmedStyle
	styles.Focused.Placeholder = dimmedStyle
	styles.Blurred.Text = textStyle
	styles.Blurred.Prompt = textStyle
	styles.Blurred.Suggestion = dimmedStyle
	styles.Blurred.Placeholder = dimmedStyle
	r.name.SetStyles(styles)

	return r.name.View()
}
