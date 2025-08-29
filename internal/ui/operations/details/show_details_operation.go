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
)

type mode int

const (
	viewMode mode = iota
	squashTargetMode
)

type Operation struct {
	context           *context.MainContext
	Overlay           Model
	Current           *jj.Commit
	keyMap            config.KeyMappings[key.Binding]
	targetMarkerStyle lipgloss.Style
	selected          *jj.Commit
}

func (s *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if s.Overlay.mode == squashTargetMode {
		switch {
		case key.Matches(msg, s.keyMap.Cancel):
			return common.Close
		case key.Matches(msg, s.keyMap.Apply):
			selectedFiles, _ := s.Overlay.getSelectedFiles()
			return tea.Batch(s.context.RunCommand(jj.SquashFiles(s.selected.GetChangeId(), s.Current.GetChangeId(), selectedFiles), common.Refresh), common.Close)
		}
	}
	return nil
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) {
	s.Current = commit
}

func (s *Operation) ShortHelp() []key.Binding {
	return s.Overlay.ShortHelp()
}

func (s *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

func (s *Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	switch s.Overlay.mode {
	case viewMode:
		var cmd tea.Cmd
		s.Overlay, cmd = s.Overlay.Update(msg)
		return s, cmd
	case squashTargetMode:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			cmd := s.HandleKey(msg)
			return s, cmd
		}
	}
	return s, nil
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if s.Overlay.mode == squashTargetMode {
		if pos == operations.RenderBeforeChangeId && s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId() {
			return s.targetMarkerStyle.Render("<< squash >>")
		}
		return ""
	}

	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return s.Overlay.View()
}

func (s *Operation) Name() string {
	if s.Overlay.mode == squashTargetMode {
		return "target"
	}
	return "details"
}

func NewOperation(context *context.MainContext, selected *jj.Commit) (operations.Operation, tea.Cmd) {
	op := &Operation{
		Overlay:           New(context, selected),
		context:           context,
		selected:          selected,
		keyMap:            config.Current.GetKeyMap(),
		targetMarkerStyle: common.DefaultPalette.Get("details target_marker"),
	}
	return op, op.Overlay.Init()
}
