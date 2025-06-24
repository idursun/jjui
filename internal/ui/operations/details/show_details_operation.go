package details

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	Overlay Model
	keyMap  config.KeyMappings[key.Binding]
}

func (s Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		s.keyMap.Up,
		s.keyMap.Down,
		s.keyMap.Cancel,
		s.keyMap.Details.Diff,
		s.keyMap.Details.ToggleSelect,
		s.keyMap.Details.Split,
		s.keyMap.Details.Restore,
		s.keyMap.Details.RevisionsChangingFile,
	}
}

func (s Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

func (s Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	var cmd tea.Cmd
	s.Overlay, cmd = s.Overlay.Update(msg)
	return s, cmd
}

func (s Operation) Render() string {
	return s.Overlay.View()
}

func (s Operation) RenderPosition() operations.RenderPosition {
	return operations.RenderPositionAfter
}

func (s Operation) Name() string {
	return "details"
}

func NewOperation(context context.AppContext, selected *jj.Commit) (operations.Operation, tea.Cmd) {
	op := Operation{
		Overlay: New(context, selected.GetChangeId()),
		keyMap:  context.KeyMap(),
	}
	return op, op.Overlay.Init()
}
