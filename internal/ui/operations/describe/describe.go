package describe

import (
	"log"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ tea.Model = (*Operation)(nil)
var _ view.IViewModel = (*Operation)(nil)
var _ operations.Operation = (*Operation)(nil)
var _ view.IView = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context  *context.MainContext
	keyMap   config.KeyMappings[key.Binding]
	input    textarea.Model
	revision *models.RevisionItem
}

func (o *Operation) Mount(v *view.ViewNode) {
	value := o.input.Value()
	_, h := lipgloss.Size(value)

	o.input.SetWidth(v.Parent.Width - 4)
	o.input.SetHeight(h + 1)

	v.Id = o.GetId()
	v.Sizeable.SetWidth(v.Parent.Width - 4)
	v.Sizeable.SetHeight(h + 1)
	o.ViewNode = v
}

func (o *Operation) GetId() view.ViewId {
	return "describe"
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Cancel,
		o.keyMap.InlineDescribe.Accept,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) Init() tea.Cmd {
	return o.input.Focus()
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Describe operation update %T", msg)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, o.keyMap.Cancel):
			o.ViewManager.StopEditing()
			o.ViewManager.UnregisterView(o.GetId())
			return o, nil
		case key.Matches(keyMsg, o.keyMap.InlineDescribe.Accept):
			value := o.input.Value()
			o.ViewManager.StopEditing()
			o.ViewManager.UnregisterView(o.GetId())
			return o, o.context.RunCommand(jj.Args(jj.SetDescriptionArgs{Revision: *o.revision, Message: value}), common.Refresh)
		}
	}
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)

	newValue := o.input.Value()
	h := lipgloss.Height(newValue)
	if h >= o.input.Height() {
		o.input.SetHeight(h + 1)
	}

	return o, cmd
}

func (o *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderOverDescription {
		return ""
	}
	return o.View()
}

func (o *Operation) View() string {
	return o.input.View()
}

func NewOperation(context *context.MainContext, revision *models.RevisionItem) *Operation {
	descOutput, _ := context.RunCommandImmediate(jj.Args(jj.GetDescriptionArgs{Revision: *revision}))
	desc := string(descOutput)

	selectedStyle := common.DefaultPalette.Get("revisions selected")

	input := textarea.New()
	input.Cursor.SetMode(cursor.CursorStatic)
	input.CharLimit = 0
	input.MaxHeight = 10
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.FocusedStyle.Base = selectedStyle.Underline(false).Strikethrough(false).Reverse(false).Blink(false)
	input.FocusedStyle.CursorLine = input.FocusedStyle.Base
	input.SetValue(desc)

	m := &Operation{
		context:  context,
		keyMap:   config.Current.GetKeyMap(),
		input:    input,
		revision: revision,
	}
	return m
}
