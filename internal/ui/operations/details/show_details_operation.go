package details

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

type Operation struct {
	*view.BaseView
	context           *context.DetailsContext
	Current           *jj.Commit
	keyMap            config.KeyMappings[key.Binding]
	targetMarkerStyle lipgloss.Style
	selected          *jj.Commit
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) {
	s.Current = commit
}

//
//func (s *Operation) ShortHelp() []key.Binding {
//	return s.Model.ShortHelp()
//}
//
//func (s *Operation) FullHelp() [][]key.Binding {
//	return [][]key.Binding{s.ShortHelp()}
//}

func (s *Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	var cmd tea.Cmd
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return s.Model.View()
}

func (s *Operation) Name() string {
	return "details"
}

func NewOperation(ctx *context.DetailsContext, selected *jj.Commit) *Operation {
	m := New(ctx.Main, selected)
	op := &Operation{
		BaseView:          &view.BaseView{Id: "details", Visible: true, Focused: true, Model: m},
		context:           ctx,
		selected:          selected,
		keyMap:            config.Current.GetKeyMap(),
		targetMarkerStyle: common.DefaultPalette.Get("details target_marker"),
	}
	return op
}
