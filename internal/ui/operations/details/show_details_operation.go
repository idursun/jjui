package details

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	Overlay tea.Model
	Current *jj.Commit
	keyMap  config.KeyMappings[key.Binding]
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) {
	s.Current = commit
}

func (s *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		s.keyMap.Up,
		s.keyMap.Down,
		s.keyMap.Cancel,
		s.keyMap.Details.Diff,
		s.keyMap.Details.ToggleSelect,
		s.keyMap.Details.Split,
		s.keyMap.Details.Restore,
		s.keyMap.Details.Absorb,
		s.keyMap.Details.RevisionsChangingFile,
	}
}

func (s *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

func (s *Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	var cmd tea.Cmd
	s.Overlay, cmd = s.Overlay.Update(msg)
	return s, cmd
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return s.Overlay.View()
}

func (s *Operation) Name() string {
	return "details"
}

func NewOperation(context *context.MainContext, selected *jj.Commit) (operations.Operation, tea.Cmd) {
	op := &Operation{
		Overlay: New(context, selected),
		keyMap:  config.Current.GetKeyMap(),
	}
	return op, op.Overlay.Init()
}
