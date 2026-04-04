package operations

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ Operation = (*Default)(nil)

type Default struct {
}

func (n *Default) Init() tea.Cmd {
	return nil
}

func (n *Default) HandleIntent(_ intents.Intent) (tea.Cmd, bool) {
	return nil, false
}

func (n *Default) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (n *Default) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (n *Default) Render(*jj.Commit, RenderPosition) string {
	return ""
}

func (n *Default) Name() string {
	return "normal"
}

func NewDefault() *Default {
	return &Default{}
}
