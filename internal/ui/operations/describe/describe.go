package describe

import (
	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ operations.Operation = (*Operation)(nil)
	_ common.Editable      = (*Operation)(nil)
)

var stashed *stashedDescription = nil

type stashedDescription struct {
	revision    *jj.Commit
	description string
}

type Operation struct {
	context      *context.MainContext
	input        textarea.Model
	revision     *jj.Commit
	originalDesc string
}

func (o *Operation) IsEditing() bool {
	return true
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderOverDescription {
		return ""
	}
	return o.viewContent(80)
}

func (o *Operation) RenderToDisplayContext(dl *render.DisplayContext, commit *jj.Commit, pos operations.RenderPosition, rect cellbuf.Rectangle, screenOffset cellbuf.Position) int {
	if pos != operations.RenderOverDescription {
		return 0
	}
	width := rect.Dx()
	height := o.DesiredHeight(commit, pos)

	o.input.SetWidth(width)
	o.input.SetHeight(height)
	content := o.input.View()

	drawRect := cellbuf.Rect(rect.Min.X, rect.Min.Y, width, height)
	dl.AddDraw(drawRect, content, 0)
	return height
}

func (o *Operation) DesiredHeight(_ *jj.Commit, pos operations.RenderPosition) int {
	if pos != operations.RenderOverDescription {
		return 0
	}
	h := lipgloss.Height(o.input.Value())
	if h <= 0 {
		h = 1
	}
	return h + 1
}

func (o *Operation) Name() string {
	return "inline_describe"
}

func (o *Operation) Scope() keybindings.Scope {
	return keybindings.Scope(actions.OwnerInlineDescribe)
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case cursor.BlinkMsg:
		// ignore cursor blink messages to prevent unnecessary rendering and height
		// recalculations
		o.input, cmd = o.input.Update(msg)
		return cmd
	case tea.KeyMsg:
	case intents.Intent:
		return o.handleIntent(msg)
	}

	o.input, cmd = o.input.Update(msg)

	return cmd
}

func (o *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent.(type) {
	case intents.Cancel:
		unsavedDescription := o.input.Value()
		if o.originalDesc == "" {
			stashed = &stashedDescription{
				revision:    o.revision,
				description: unsavedDescription,
			}
			return tea.Batch(common.Close, func() tea.Msg {
				return intents.AddMessage{Text: "Unsaved description is stashed. Edit again to restore."}
			})
		}
		return common.Close
	case intents.InlineDescribeEditor:
		return o.runInlineDescribeEditor()
	case intents.InlineDescribeAccept:
		return o.runInlineDescribeAccept()
	default:
		return nil
	}
}

func (o *Operation) runInlineDescribeEditor() tea.Cmd {
	selectedRevisions := jj.NewSelectedRevisions(o.revision)
	cmd := jj.SetDescription(o.revision.GetChangeId(), o.input.Value())
	return o.context.RunCommandWithInput(
		cmd.Args, cmd.Input,
		common.CloseApplied,
		o.context.RunInteractiveCommand(jj.Describe(selectedRevisions), common.Refresh),
	)
}

func (o *Operation) runInlineDescribeAccept() tea.Cmd {
	cmd := jj.SetDescription(o.revision.GetChangeId(), o.input.Value())
	return o.context.RunCommandWithInput(cmd.Args, cmd.Input, common.CloseApplied, common.Refresh)
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := o.viewContent(box.R.Dx())
	w, h := lipgloss.Size(content)
	rect := cellbuf.Rect(box.R.Min.X, box.R.Min.Y, w, h)
	dl.AddDraw(rect, content, 0)
}

func NewOperation(context *context.MainContext, revision *jj.Commit) *Operation {
	descOutput, _ := context.RunCommandImmediate(jj.GetDescription(revision.GetChangeId()))
	originalDesc := string(descOutput)
	desc := originalDesc
	if stashed != nil && stashed.revision.CommitId == revision.CommitId && originalDesc == "" {
		desc = stashed.description
	}

	// clear the stashed description regardless
	stashed = nil

	selectedStyle := common.DefaultPalette.Get("revisions selected")

	input := textarea.New()
	input.CharLimit = 0
	input.Prompt = ""
	input.ShowLineNumbers = false
	ds := input.Styles()
	ds.Focused.Base = selectedStyle.Underline(false).Strikethrough(false).Reverse(false).Blink(false)
	ds.Focused.CursorLine = ds.Focused.Base
	input.SetStyles(ds)
	input.SetValue(desc)
	input.Focus()

	return &Operation{
		context:      context,
		input:        input,
		originalDesc: originalDesc,
		revision:     revision,
	}
}

func (o *Operation) viewContent(width int) string {
	if width <= 0 {
		width = 80
	}
	h := lipgloss.Height(o.input.Value())
	if h <= 0 {
		h = 1
	}
	height := h + 1

	o.input.SetWidth(width)
	o.input.SetHeight(height)
	return o.input.View()
}

func (o *Operation) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	return actions.ResolveByScopeStrict(o.Scope(), action, args)
}
