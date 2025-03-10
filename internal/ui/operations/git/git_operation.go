package git

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	context context.AppContext
	keyMap  config.KeyMappings[key.Binding]
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) RenderPosition() operations.RenderPosition {
	return operations.RenderPositionNil
}

func (o *Operation) Render() string {
	return ""
}

func (o *Operation) Name() string {
	return "git"
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, o.keyMap.Git.Fetch):
		return o.context.RunCommand(jj.GitFetch(), common.Refresh, common.Close)
	case key.Matches(msg, o.keyMap.Git.Push):
		return o.context.RunCommand(jj.GitPush(), common.Refresh, common.Close)
	case key.Matches(msg, o.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Git.Fetch,
		o.keyMap.Git.Push,
		o.keyMap.Cancel,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func NewOperation(context context.AppContext) *Operation {
	return &Operation{
		context: context,
		keyMap:  context.KeyMap(),
	}
}
