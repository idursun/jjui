package bookmark

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/operations"
)

var (
	applyMove = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "move bookmark"))
)

type MoveBookmarkOperation struct {
	Overlay tea.Model
}

func (m MoveBookmarkOperation) ShortHelp() []key.Binding {
	return []key.Binding{
		cancel,
		applyMove,
	}
}

func (m MoveBookmarkOperation) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m MoveBookmarkOperation) Update(msg tea.Msg) (operations.Operation, tea.Cmd) {
	var cmd tea.Cmd
	m.Overlay, cmd = m.Overlay.Update(msg)
	return MoveBookmarkOperation{Overlay: m.Overlay}, cmd
}

func (m MoveBookmarkOperation) Render() string {
	return m.Overlay.View()
}

func (m MoveBookmarkOperation) RenderPosition() operations.RenderPosition {
	return operations.RenderPositionAfter
}

func NewMoveBookmarkOperation(commands common.UICommands, selected *jj.Commit, width int) (operations.Operation, tea.Cmd) {
	op := MoveBookmarkOperation{
		Overlay: New(commands, selected.GetChangeId(), width),
	}
	return op, tea.Batch(commands.FetchBookmarks(selected.GetChangeId()), op.Overlay.Init())
}
