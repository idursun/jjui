package operations

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/context"
)

type Default struct {
	keyMap config.KeyMappings[key.Binding]
}

func (n *Default) ShortHelp() []key.Binding {
	return []key.Binding{n.keyMap.Up, n.keyMap.Down, n.keyMap.Quit, n.keyMap.Help, n.keyMap.Refresh, n.keyMap.Preview.Mode, n.keyMap.Revset, n.keyMap.Details.Mode, n.keyMap.Evolog, n.keyMap.Rebase.Mode, n.keyMap.Squash, n.keyMap.Bookmark.Mode, n.keyMap.Git.Mode, n.keyMap.OpLog.Mode}
}

func (n *Default) FullHelp() [][]key.Binding {
	return [][]key.Binding{n.ShortHelp()}
}

func (n *Default) RenderPosition() RenderPosition {
	return RenderPositionNil
}

func (n *Default) Render() string {
	return ""
}

func (n *Default) Name() string {
	return "normal"
}

func NewDefault(c context.AppContext) *Default {
	return &Default{
		keyMap: c.KeyMap(),
	}
}
