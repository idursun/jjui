package workspace

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

var _ operations.Operation = (*RenameWorkspaceOperation)(nil)
var _ common.Editable = (*RenameWorkspaceOperation)(nil)
var _ common.ScopeProvider = (*RenameWorkspaceOperation)(nil)
var _ operations.SegmentRenderer = (*RenameWorkspaceOperation)(nil)

// RenameWorkspaceOperation edits the current workspace's name in place on its
// working-copy commit.
type RenameWorkspaceOperation struct {
	context  *context.MainContext
	revision string
	oldName  string
	name     textinput.Model
	// cursorOffset is the width of the workspace names rendered before the
	// input in the same segment.
	cursorOffset int
}

func NewRenameWorkspaceOperation(context *context.MainContext, changeId string, oldName string) *RenameWorkspaceOperation {
	t := textinput.New()
	t.CharLimit = 120
	t.Prompt = ""
	t.SetVirtualCursor(false)
	t.SetValue(oldName)
	t.CursorEnd()
	t.Focus()

	return &RenameWorkspaceOperation{
		context:  context,
		revision: changeId,
		oldName:  oldName,
		name:     t,
	}
}

func (r *RenameWorkspaceOperation) IsEditing() bool {
	return true
}

func (r *RenameWorkspaceOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeRenameWorkspace,
			Leak:    common.LeakNone,
			Handler: r,
		},
	}
}

func (r *RenameWorkspaceOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		newName := strings.TrimSpace(r.name.Value())
		if newName == "" || newName == r.oldName {
			return common.Close, true
		}
		return r.context.RunCommand(jj.WorkspaceRename(newName), common.CloseApplied, common.Refresh), true
	}
	return nil, false
}

func (r *RenameWorkspaceOperation) Update(msg tea.Msg) tea.Cmd {
	if intent, ok := msg.(intents.Intent); ok {
		cmd, _ := r.HandleIntent(intent)
		return cmd
	}
	var cmd tea.Cmd
	r.name, cmd = r.name.Update(msg)
	r.name.SetValue(strings.ReplaceAll(r.name.Value(), " ", "-"))
	return cmd
}

func (r *RenameWorkspaceOperation) Init() tea.Cmd {
	return textinput.Blink
}

func (r *RenameWorkspaceOperation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := r.styledContent(theme.DefaultPalette.Get("revisions", "", "text", false))
	w, h := lipgloss.Size(content)
	rect := layout.Rect(box.R.Min.X, box.R.Min.Y, w, h)
	dl.AddDraw(rect, content, 0)
	dl.SetCursorInRect(r.name.Cursor(), rect, 0, 0)
}

func (r *RenameWorkspaceOperation) IsFocused() bool {
	return true
}

func (r *RenameWorkspaceOperation) Render(*jj.Commit, operations.RenderPosition) string {
	return ""
}

// RenderSegment replaces the old name in the segment listing the workspaces
// of the working-copy commit, which reads "a@ b@".
func (r *RenameWorkspaceOperation) RenderSegment(style lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	if row.Commit == nil || row.Commit.GetChangeId() != r.revision || !segment.HasLabel("working_copies") {
		return ""
	}
	names := strings.Split(segment.Text, " ")
	i := 0
	for i < len(names) && names[i] != r.oldName+"@" {
		i++
	}
	if i == len(names) {
		return ""
	}
	before := strings.Join(names[:i], " ")
	after := strings.Join(names[i+1:], " ")
	if before != "" {
		before += " "
	}
	if after != "" {
		after = " " + after
	}
	r.cursorOffset = render.StringWidth(before)
	return style.Render(before) + r.styledContent(style) + style.Render("@"+after)
}

func (r *RenameWorkspaceOperation) InlineCursor(commit *jj.Commit, pos operations.RenderPosition) *tea.Cursor {
	if pos != operations.RenderOverSegment || commit.GetChangeId() != r.revision {
		return nil
	}
	cursor := r.name.Cursor()
	if cursor != nil {
		cursor.X += r.cursorOffset
	}
	return cursor
}

func (r *RenameWorkspaceOperation) Name() string {
	return "rename workspace"
}

// styledContent renders the name with the style of the text it replaces. The
// terminal cursor marks the edit position, so only the value is drawn and the
// @ that follows stays next to it.
func (r *RenameWorkspaceOperation) styledContent(style lipgloss.Style) string {
	value := r.name.Value()
	if value == "" {
		value = " "
	}
	return style.Inline(true).Render(value)
}
