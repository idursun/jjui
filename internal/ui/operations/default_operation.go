package operations

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
)

var _ Operation = (*Default)(nil)

type Default struct {
	keyMap config.KeyMappings[key.Binding]
}

func (n *Default) Init() tea.Cmd {
	return nil
}

func (n *Default) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (n *Default) View() string {
	return ""
}

func (n *Default) ShortHelp() []key.Binding {
	return []key.Binding{
		n.keyMap.Up,
		n.keyMap.Down,
		n.keyMap.Quit,
		n.keyMap.Help,
		n.keyMap.Refresh,
		n.keyMap.Preview.Mode,
		n.keyMap.Revset,
		n.keyMap.Details.Mode,
		n.keyMap.Evolog.Mode,
		n.keyMap.Rebase.Mode,
		n.keyMap.Squash.Mode,
		n.keyMap.Bookmark.Mode,
		n.keyMap.Git.Mode,
		n.keyMap.OpLog.Mode,
	}
}

func (n *Default) FullHelp() [][]key.Binding {
	return [][]key.Binding{n.ShortHelp()}
}

func (n *Default) Render(*jj.Commit, RenderPosition) string {
	return ""
}

func (n *Default) Name() string {
	return "normal"
}

func NewDefault() *Default {
	return &Default{
		keyMap: config.Current.GetKeyMap(),
	}
}
