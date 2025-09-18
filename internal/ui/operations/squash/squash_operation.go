package squash

import (
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ view.IViewModel = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context     *context.MainContext
	from        jj.SelectedRevisions
	files       []*models.RevisionFileItem
	current     *models.RevisionItem
	keyMap      config.KeyMappings[key.Binding]
	keepEmptied bool
	interactive bool
	styles      styles
}

func (s *Operation) Init() tea.Cmd {
	return nil
}

func (s *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if cmd := s.HandleKey(msg); cmd != nil {
			return s, cmd
		}
	case common.RefreshMsg:
		s.setSelectedRevision()
		return s, nil
	}
	return s, nil
}

func (s *Operation) View() string {
	return ""
}

func (s *Operation) GetId() view.ViewId {
	return "squash"
}

func (s *Operation) Mount(v *view.ViewNode) {
	s.ViewNode = v
	v.Id = s.GetId()
	v.NeedsRefresh = true
	revisionsViewId := view.RevisionsViewId
	v.KeyDelegation = &revisionsViewId
}

type styles struct {
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
}

func (s *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, s.keyMap.Apply):
		s.ViewManager.UnregisterView(s.GetId())
		var args jj.IGetArgs
		if len(s.files) > 0 {
			args = jj.SquashFilesArgs{
				From:        *s.from[0],
				Into:        *s.current,
				Files:       jj.Convert(s.files),
				Interactive: false,
				KeepEmptied: false,
			}
		} else {
			args = jj.SquashRevisionArgs{
				From:        s.from,
				Into:        *s.current,
				Interactive: s.interactive,
				KeepEmptied: s.keepEmptied,
			}
		}
		return s.context.RunInteractiveCommand(jj.Squash(args), common.RefreshAndSelect(s.current.Commit.GetChangeId()))
	case key.Matches(msg, s.keyMap.Cancel):
		s.ViewManager.UnregisterView(s.GetId())
		return nil
	case key.Matches(msg, s.keyMap.Squash.KeepEmptied):
		s.keepEmptied = !s.keepEmptied
	case key.Matches(msg, s.keyMap.Squash.Interactive):
		s.interactive = !s.interactive
	}
	return nil
}

func (s *Operation) setSelectedRevision() {
	if current := s.context.Revisions.Current(); current != nil {
		s.current = current
	}
}

func (s *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}

	isSelected := s.current != nil && s.current.Commit.GetChangeId() == commit.GetChangeId()
	if isSelected {
		return s.styles.targetMarker.Render("<< into >>")
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
		return s.styles.sourceMarker.Render(marker)
	}
	return ""
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

type SquashOperationOpts struct {
	KeepEmptied bool
	Interactive bool
	From        jj.SelectedRevisions
	Files       []*models.RevisionFileItem
}

func NewSquashRevisionsOpts(from jj.SelectedRevisions) SquashOperationOpts {
	return SquashOperationOpts{
		From: from,
	}
}

func NewSquashFilesOpts(from jj.SelectedRevisions, files []*models.RevisionFileItem) SquashOperationOpts {
	return SquashOperationOpts{
		From:  from,
		Files: files,
	}
}

func NewOperation(context *context.MainContext, opts SquashOperationOpts) view.IViewModel {
	styles := styles{
		dimmed:       common.DefaultPalette.Get("squash dimmed"),
		sourceMarker: common.DefaultPalette.Get("squash source_marker"),
		targetMarker: common.DefaultPalette.Get("squash target_marker"),
	}
	return &Operation{
		context:     context,
		keyMap:      config.Current.GetKeyMap(),
		from:        opts.From,
		files:       opts.Files,
		interactive: opts.Interactive,
		keepEmptied: opts.KeepEmptied,
		styles:      styles,
	}
}
