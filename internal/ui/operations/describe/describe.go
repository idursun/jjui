package describe

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations"
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
	*common.ViewNode
	context      *context.MainContext
	keyMap       config.KeyMappings[key.Binding]
	input        textarea.Model
	revision     *jj.Commit
	originalDesc string
}

func (o *Operation) IsEditing() bool {
	return true
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Cancel,
		o.keyMap.InlineDescribe.Editor,
		o.keyMap.InlineDescribe.Accept,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderOverDescription {
		return ""
	}
	return o.View()
}

func (o *Operation) Name() string {
	return "desc"
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
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			unsavedDescription := o.input.Value()
			if o.originalDesc != unsavedDescription {
				stashed = &stashedDescription{
					revision:    o.revision,
					description: unsavedDescription,
				}
				return tea.Batch(common.Close, intents.Invoke(intents.AddMessage{Text: "Unsaved description is stashed. Edit again to restore."}))
			}
			return common.Close
		case key.Matches(msg, o.keyMap.InlineDescribe.Editor):
			selectedRevisions := jj.NewSelectedRevisions(o.revision)
			return o.context.RunCommand(
				jj.SetDescription(o.revision.GetChangeId(), o.input.Value()),
				common.CloseApplied,
				o.context.RunInteractiveCommand(jj.Describe(selectedRevisions), common.Refresh),
			)
		case key.Matches(msg, o.keyMap.InlineDescribe.Accept):
			return o.context.RunCommand(jj.SetDescription(o.revision.GetChangeId(), o.input.Value()), common.CloseApplied, common.Refresh)
		}
	}

	o.input, cmd = o.input.Update(msg)

	newValue := o.input.Value()
	h := lipgloss.Height(newValue)
	if h >= o.input.Height() {
		o.SetHeight(h + 1)
	}

	return cmd
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) View() string {
	o.SetWidth(o.Parent.Width)
	o.input.SetWidth(o.Width)
	o.input.SetHeight(o.Height)
	return o.input.View()
}

func NewOperation(context *context.MainContext, revision *jj.Commit) *Operation {
	descOutput, _ := context.RunCommandImmediate(jj.GetDescription(revision.GetChangeId()))
	originalDesc := string(descOutput)
	desc := originalDesc
	if stashed != nil && stashed.revision.CommitId == revision.CommitId && stashed.description != originalDesc {
		desc = stashed.description
	}

	// clear the stashed description regardless
	stashed = nil

	h := lipgloss.Height(desc)

	selectedStyle := common.DefaultPalette.Get("revisions selected")

	input := textarea.New()
	input.CharLimit = 0
	input.MaxHeight = 10
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.FocusedStyle.Base = selectedStyle.Underline(false).Strikethrough(false).Reverse(false).Blink(false)
	input.FocusedStyle.CursorLine = input.FocusedStyle.Base
	input.SetValue(desc)
	input.Focus()

	return &Operation{
		ViewNode:     common.NewViewNode(0, h+1),
		context:      context,
		keyMap:       config.Current.GetKeyMap(),
		input:        input,
		originalDesc: originalDesc,
		revision:     revision,
	}
}
