package describe

import (
	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ operations.Operation         = (*Operation)(nil)
	_ operations.EmbeddedOperation = (*Operation)(nil)
	_ common.Editable              = (*Operation)(nil)
	_ dispatch.ScopeProvider       = (*Operation)(nil)
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

func (o *Operation) Scopes() []dispatch.Scope {
	return []dispatch.Scope{
		{
			Name:    actions.ScopeInlineDescribe,
			Leak:    dispatch.LeakNone,
			Handler: o,
		},
	}
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderOverDescription {
		return ""
	}
	return o.resizeInput(80, 0).View()
}

func (o *Operation) CanEmbed(_ *jj.Commit, pos operations.RenderPosition) bool {
	return pos == operations.RenderOverDescription
}

func (o *Operation) EmbeddedHeight(commit *jj.Commit, pos operations.RenderPosition, width int) int {
	if !o.CanEmbed(commit, pos) {
		return 0
	}
	return o.resizeInput(width, 0).Height()
}

func (o *Operation) Name() string {
	return "inline_describe"
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case cursor.BlinkMsg:
		// ignore cursor blink messages to prevent unnecessary rendering and height
		// recalculations
		o.input, cmd = o.input.Update(msg)
		return cmd
	case intents.Intent:
		cmd, _ := o.HandleIntent(msg)
		return cmd
	}

	o.input, cmd = o.input.Update(msg)

	return cmd
}

func (o *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		unsavedDescription := o.input.Value()
		if o.originalDesc == "" {
			stashed = &stashedDescription{
				revision:    o.revision,
				description: unsavedDescription,
			}
			return tea.Batch(common.Close, func() tea.Msg {
				return intents.AddMessage{Text: "Unsaved description is stashed. Edit again to restore."}
			}), true
		}
		return common.Close, true
	case intents.InlineDescribeEditor:
		return o.runInlineDescribeEditor(), true
	case intents.InlineDescribeNewLine:
		o.input.InsertString("\n")
		return nil, true
	case intents.InlineDescribeAccept:
		return o.runInlineDescribeAccept(intent.Force), true
	}
	return nil, false
}

func (o *Operation) runInlineDescribeEditor() tea.Cmd {
	selectedRevisions := jj.NewSelectedRevisions(o.revision)
	cmd := jj.SetDescription(o.revision.GetChangeId(), o.input.Value(), false)
	return o.context.RunCommandWithInput(
		cmd.Args, cmd.Input,
		common.CloseApplied,
		o.context.RunInteractiveCommand(jj.Describe(selectedRevisions), common.Refresh),
	)
}

func (o *Operation) runInlineDescribeAccept(force bool) tea.Cmd {
	cmd := jj.SetDescription(o.revision.GetChangeId(), o.input.Value(), force)
	return o.context.RunCommandWithInput(cmd.Args, cmd.Input, common.CloseApplied, common.Refresh)
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	input := o.resizeInput(box.R.Dx(), box.R.Dy())

	selectedStyle := common.DefaultPalette.Get("revisions selected")
	ds := input.Styles()
	ds.Focused.Base = selectedStyle.Underline(false).Strikethrough(false).Reverse(false).Blink(false)
	ds.Focused.CursorLine = ds.Focused.Base
	input.SetStyles(ds)

	rect := layout.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), input.Height())
	dl.AddDraw(rect, input.View(), 0)
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

	input := textarea.New()
	input.CharLimit = 0
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.DynamicHeight = true
	input.MinHeight = 1

	input.SetValue(desc)
	input.Focus()

	return &Operation{
		context:      context,
		input:        input,
		originalDesc: originalDesc,
		revision:     revision,
	}
}

func (o *Operation) resizeInput(width, maxHeight int) textarea.Model {
	input := o.input
	if width <= 0 {
		width = 80
	}
	input.MaxHeight = maxHeight
	if maxHeight <= 0 {
		input.MaxHeight = 0
	}
	input.SetWidth(width)
	return input
}
