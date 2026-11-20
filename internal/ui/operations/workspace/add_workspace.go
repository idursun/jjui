package workspace

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

var _ operations.Operation = (*AddWorkspaceOperation)(nil)
var _ common.Editable = (*AddWorkspaceOperation)(nil)
var _ common.ScopeProvider = (*AddWorkspaceOperation)(nil)

type AddWorkspaceOperation struct {
	context  *context.MainContext
	revision string
	input    textinput.Model
}

func NewAddWorkspaceOperation(context *context.MainContext, changeId string, initialValue string) *AddWorkspaceOperation {
	t := textinput.New()
	t.CharLimit = 255
	t.Prompt = ""
	t.Placeholder = "workspace path"
	t.SetVirtualCursor(false)
	t.SetValue(initialValue)
	t.CursorEnd()
	t.Focus()

	return &AddWorkspaceOperation{
		context:  context,
		revision: changeId,
		input:    t,
	}
}

func (o *AddWorkspaceOperation) IsEditing() bool {
	return true
}

func (o *AddWorkspaceOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeAddWorkspace,
			Leak:    common.LeakNone,
			Handler: o,
		},
	}
}

func (o *AddWorkspaceOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		value := strings.TrimSpace(o.input.Value())
		if value == "" {
			return common.Close, true
		}
		return o.context.RunCommand(jj.WorkspaceAdd(o.revision, value), common.CloseApplied, common.Refresh), true
	}
	return nil, false
}

func (o *AddWorkspaceOperation) Update(msg tea.Msg) tea.Cmd {
	if intent, ok := msg.(intents.Intent); ok {
		cmd, _ := o.HandleIntent(intent)
		return cmd
	}
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return cmd
}

func (o *AddWorkspaceOperation) Init() tea.Cmd {
	return textinput.Blink
}

func (o *AddWorkspaceOperation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := o.viewContent()
	w, h := lipgloss.Size(content)
	rect := layout.Rect(box.R.Min.X, box.R.Min.Y, w, h)
	dl.AddDraw(rect, content, 0)
	dl.SetCursorInRect(o.input.Cursor(), rect, 0, 0)
}

func (o *AddWorkspaceOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != o.revision {
		return ""
	}
	return o.viewContent() + o.input.Styles().Focused.Text.Render(" ")
}

func (o *AddWorkspaceOperation) InlineCursor(commit *jj.Commit, pos operations.RenderPosition) *tea.Cursor {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != o.revision {
		return nil
	}
	return o.input.Cursor()
}

func (o *AddWorkspaceOperation) Name() string {
	return "add workspace"
}

func (o *AddWorkspaceOperation) viewContent() string {
	dimmedStyle := theme.DefaultPalette.Get("revisions", "", "dimmed", false).Inline(true)
	textStyle := theme.DefaultPalette.Get("revisions", "", "text", false).Inline(true)
	styles := o.input.Styles()
	styles.Focused.Text = textStyle
	styles.Focused.Prompt = textStyle
	styles.Focused.Placeholder = dimmedStyle
	styles.Blurred.Text = textStyle
	styles.Blurred.Prompt = textStyle
	styles.Blurred.Placeholder = dimmedStyle
	o.input.SetStyles(styles)

	// textinput shows only the first placeholder rune when it has no width.
	if o.input.Value() == "" {
		return dimmedStyle.Render(o.input.Placeholder)
	}
	return o.input.View()
}
