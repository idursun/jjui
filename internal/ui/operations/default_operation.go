package operations

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
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

func (n *Default) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (n *Default) ShortHelp() []key.Binding {
	return []key.Binding{
		n.keyMap.Up,
		n.keyMap.Down,
		n.keyMap.Details.Mode,
		n.keyMap.Abandon,
		n.keyMap.New,
		n.keyMap.Split,
		n.keyMap.Diff,
		n.keyMap.AceJump,
		n.keyMap.Preview.Mode,
		n.keyMap.Revset,
		n.keyMap.ToggleSelect,
		n.keyMap.InlineDescribe.Mode,
		n.keyMap.Describe,
		n.keyMap.Edit,
		n.keyMap.Rebase.Mode,
		n.keyMap.Squash.Mode,
		n.keyMap.Bookmark.Set,
		n.keyMap.Bookmark.Mode,
		n.keyMap.Git.Mode,
		n.keyMap.Revert.Mode,
		n.keyMap.JumpToParent,
		n.keyMap.JumpToChildren,
		n.keyMap.JumpToWorkingCopy,
		n.keyMap.Commit,
		n.keyMap.Diffedit,
		n.keyMap.Absorb,
		n.keyMap.Duplicate.Mode,
		n.keyMap.Evolog.Mode,
		n.keyMap.Refresh,
		n.keyMap.SetParents,
		n.keyMap.OpLog.Mode,
		n.keyMap.CustomCommands,
		n.keyMap.Leader,
		n.keyMap.Quit,
	}
}

func (n *Default) FullHelp() [][]key.Binding {
	return [][]key.Binding{n.ShortHelp()}
}

func (n *Default) Render(*jj.Commit, RenderPosition) string {
	return ""
}

func (n *Default) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (n *Default) DesiredHeight(_ *jj.Commit, _ RenderPosition) int {
	return 0
}

func (n *Default) Name() string {
	return "normal"
}

func NewDefault() *Default {
	return &Default{
		keyMap: config.Current.GetKeyMap(),
	}
}
