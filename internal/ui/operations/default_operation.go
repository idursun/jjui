package operations

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ Operation = (*Default)(nil)

type Default struct {
}

func (n *Default) Init() tea.Cmd {
	return nil
}

func (n *Default) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (n *Default) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

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

func (n *Default) Scope() keybindings.Scope {
	return actions.OwnerRevisions
}

func NewDefault() *Default {
	return &Default{}
}
