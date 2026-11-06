package bookmark

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
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

// RenameBookmarkOperation edits a bookmark name inline on its revision.
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

func (r *RenameBookmarkOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != r.revision {
		return ""
	}
	return r.viewContent() + r.name.Styles().Focused.Text.Render(" ")
}

func (r *RenameBookmarkOperation) InlineCursor(commit *jj.Commit, pos operations.RenderPosition) *tea.Cursor {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != r.revision {
		return nil
	}
	return r.name.Cursor()
}

func (r *RenameBookmarkOperation) Name() string {
	return "rename bookmark"
}

func (r *RenameBookmarkOperation) viewContent() string {
	dimmedStyle := theme.DefaultPalette.Get("revisions", "", "dimmed", false).Inline(true)
	textStyle := theme.DefaultPalette.Get("revisions", "", "text", false).Inline(true)
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
