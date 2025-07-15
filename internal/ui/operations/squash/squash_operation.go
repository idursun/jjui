package squash

import (
	"github.com/charmbracelet/lipgloss"
	"slices"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Operation struct {
	context     *context.MainContext
	from        jj.SelectedRevisions
	current     *jj.Commit
	keyMap      config.KeyMappings[key.Binding]
	keepEmptied bool
	interactive bool
	styles      styles
}

type styles struct {
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
}

func (s *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, s.keyMap.Apply):
		return tea.Batch(common.Close, s.context.RunInteractiveCommand(jj.Squash(s.from, s.current.ChangeId, s.keepEmptied, s.interactive), common.Refresh))
	case key.Matches(msg, s.keyMap.Cancel):
		return common.Close
	case key.Matches(msg, s.keyMap.Squash.KeepEmptied):
		s.keepEmptied = !s.keepEmptied
	case key.Matches(msg, s.keyMap.Squash.Interactive):
		s.interactive = !s.interactive
	}
	return nil
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) {
	s.current = commit
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}

	isSelected := s.current != nil && s.current.GetChangeId() == commit.GetChangeId()
	if isSelected {
		return s.styles.targetMarker.Render("<< into >>") + " "
	}
	sourceIds := s.from.GetIds()
	if slices.Contains(sourceIds, commit.ChangeId) {
		marker := "<< from >>"
		if s.keepEmptied {
			marker = "<< keep empty >>"
		}
		if s.interactive {
			marker += " (interactive)"
		}
		return s.styles.sourceMarker.Render(marker) + " "
	}
	return ""
}

func (s *Operation) Name() string {
	return "squash"
}

func (s *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		s.keyMap.Apply,
		s.keyMap.Cancel,
		s.keyMap.Squash.KeepEmptied,
		s.keyMap.Squash.Interactive,
	}
}

func (s *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions) *Operation {
	styles := styles{
		dimmed:       common.DefaultPalette.Get("squash dimmed"),
		sourceMarker: common.DefaultPalette.Get("squash source_marker"),
		targetMarker: common.DefaultPalette.Get("squash target_marker"),
	}
	return &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		from:    from,
		styles:  styles,
	}
}
