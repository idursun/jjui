package describe

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ operations.Operation         = (*Operation)(nil)
	_ operations.EmbeddedOperation = (*Operation)(nil)
	_ common.Editable              = (*Operation)(nil)
	_ common.ScopeProvider         = (*Operation)(nil)
	_ common.StateProvider         = (*Operation)(nil)
)

type Operation struct {
	context      *context.MainContext
	input        textarea.Model
	revision     *jj.Commit
	originalDesc string
	killedText   string
	confirmation *confirmation.Model
}

func (o *Operation) IsEditing() bool {
	return true
}

func (o *Operation) QueryState(name string) (any, bool) {
	if name == "content" {
		return o.input.Value(), true
	}
	return nil, false
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Scopes() []common.Scope {
	var scopes []common.Scope
	if o.confirmation != nil {
		scopes = append(scopes, common.Scope{
			Name:    actions.ScopeInlineDescribeConfirmation,
			Leak:    common.LeakNone,
			Handler: o,
		})
	}
	return append(scopes, common.Scope{
		Name:    actions.ScopeInlineDescribe,
		Leak:    common.LeakNone,
		Handler: o,
	})
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderOverDescription {
		return ""
	}
	view := o.resizeInput(80, 0).View()
	if o.confirmation != nil {
		view = lipgloss.JoinVertical(lipgloss.Left, view, o.confirmation.View())
	}
	return view
}

func (o *Operation) CanEmbed(_ *jj.Commit, pos operations.RenderPosition) bool {
	return pos == operations.RenderOverDescription
}

func (o *Operation) EmbeddedHeight(commit *jj.Commit, pos operations.RenderPosition, width int) int {
	if !o.CanEmbed(commit, pos) {
		return 0
	}
	return o.resizeInput(width, 0).Height() + o.confirmationHeight()
}

func (o *Operation) Name() string {
	return "inline_describe"
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case confirmation.CloseMsg:
		o.confirmation = nil
		return nil
	case confirmation.SelectOptionMsg:
		if o.confirmation != nil {
			return o.confirmation.Update(msg)
		}
		return nil
	case intents.Intent:
		cmd, _ := o.HandleIntent(msg)
		return cmd
	case tea.KeyPressMsg:
		if o.confirmation != nil {
			return o.confirmation.Update(msg)
		}
		isCtrlKill := msg.Mod&tea.ModCtrl != 0 && (msg.Code == 'u' || msg.Code == 'k')
		if isCtrlKill {
			o.captureKilledText(msg.Code == 'k')
		} else if msg.Code == 'y' && msg.Mod&tea.ModCtrl != 0 {
			if o.killedText != "" {
				o.input.InsertString(o.killedText)
			}
			return nil
		}
	case tea.PasteMsg:
		if o.confirmation != nil {
			return nil
		}
	}

	o.input, cmd = o.input.Update(msg)

	return cmd
}

func (o *Operation) captureKilledText(forward bool) {
	lines := strings.Split(o.input.Value(), "\n")
	row := o.input.Line()
	col := o.input.Column()
	if row < 0 || row >= len(lines) {
		return
	}

	line := []rune(lines[row])
	if col > len(line) {
		col = len(line)
	}
	if forward && col < len(line) {
		o.killedText = string(line[col:])
	} else if forward && row < len(lines)-1 {
		// Bubbles joins the current line with the next one when Ctrl+K is
		// pressed at the end of a line.
		o.killedText = "\n"
	} else if !forward && col > 0 {
		o.killedText = string(line[:col])
	} else if !forward && row > 0 {
		// Bubbles joins the current line with the previous one when Ctrl+U is
		// pressed at the beginning of a line.
		o.killedText = "\n"
	}
}

func (o *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		if o.confirmation != nil {
			return o.confirmation.Update(intent), true
		}
		if o.input.Value() != o.originalDesc {
			return o.openCancelConfirmation(), true
		}
		return common.Close, true
	case intents.Apply:
		if o.confirmation != nil {
			return o.confirmation.Update(intent), true
		}
	case intents.OptionSelect:
		if o.confirmation != nil {
			return o.confirmation.Update(intent), true
		}
	case intents.InlineDescribeEditor:
		return o.runInlineDescribeEditor(), true
	case intents.InlineDescribeNewLine:
		o.input.InsertString("\n")
		return nil, true
	case intents.InlineDescribeClear:
		if value := o.input.Value(); value != "" {
			o.killedText = value
		}
		o.input.Reset()
		return nil, true
	case intents.InlineDescribeAccept:
		return o.runInlineDescribeAccept(intent.Force), true
	}
	return nil, false
}

func (o *Operation) openCancelConfirmation() tea.Cmd {
	o.confirmation = confirmation.New(
		[]string{"You have unsaved changes. Discard them?"},
		confirmation.WithStyleScope("revisions"),
		confirmation.WithOption("Discard",
			common.Close,
			key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "discard"))),
		confirmation.WithOption("Keep editing",
			confirmation.Close,
			key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "keep editing"))),
	)
	background := common.DefaultPalette.GetBlended("revisions", "", "", true).GetBackground()
	o.confirmation.Styles.Border = o.confirmation.Styles.Border.
		Background(background).
		BorderBackground(background)
	o.confirmation.Styles.Text = o.confirmation.Styles.Text.Background(background)
	o.confirmation.Styles.Dimmed = o.confirmation.Styles.Dimmed.Background(background)
	return o.confirmation.Init()
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
	confirmationHeight := o.confirmationHeight()
	o.input = o.resizeInput(box.R.Dx(), max(box.R.Dy()-confirmationHeight, 0))
	input := o.input

	selectedStyle := common.DefaultPalette.GetBlended("revisions", "", "", true)
	ds := input.Styles()
	ds.Focused.Base = selectedStyle.Underline(false).Strikethrough(false).Reverse(false).Blink(false)
	ds.Focused.CursorLine = ds.Focused.Base
	input.SetStyles(ds)

	rect := layout.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), input.Height())
	dl.AddDraw(rect, input.View(), 0)
	if o.confirmation == nil {
		dl.SetCursorInRect(input.Cursor(), rect, 0, 0)
	} else if confirmationHeight > 0 && input.Height() < box.R.Dy() {
		confirmationRect := layout.Rect(box.R.Min.X, box.R.Min.Y+input.Height(), box.R.Dx(), confirmationHeight)
		o.confirmation.ViewRect(dl, layout.Box{R: confirmationRect})
	}
}

func (o *Operation) confirmationHeight() int {
	if o.confirmation == nil {
		return 0
	}
	return lipgloss.Height(o.confirmation.View())
}

func NewOperation(context *context.MainContext, revision *jj.Commit) *Operation {
	descOutput, _ := context.RunCommandImmediate(jj.GetDescription(revision.GetChangeId()))
	originalDesc := string(descOutput)

	input := textarea.New()
	input.CharLimit = 0
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.DynamicHeight = true
	input.MinHeight = 1
	input.SetVirtualCursor(false)

	input.SetValue(originalDesc)
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
	input.MaxHeight = max(maxHeight, 0)
	input.SetWidth(width)
	return input
}
