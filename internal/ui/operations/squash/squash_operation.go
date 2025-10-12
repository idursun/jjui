package squash

import (
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ operations.Operation = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)

type Operation struct {
	context     *context.MainContext
	from        jj.SelectedRevisions
	files       []string
	current     *jj.Commit
	keepEmptied bool
	interactive bool
	styles      styles
}

func (s *Operation) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings("squash")
}

type styles struct {
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
}

func (s *Operation) Init() tea.Cmd {
	return nil
}

func (s *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "squash.apply", "squash.force_apply":
			ignoreImmutable := msg.Action.Id == "squash.force_apply"
			return s, tea.Batch(s.context.RunInteractiveCommand(jj.Squash(s.from, s.current.GetChangeId(), s.files, s.keepEmptied, s.interactive, ignoreImmutable), common.RefreshAndSelect(s.current.GetChangeId())))
		case "squash.keep_emptied":
			s.keepEmptied = !s.keepEmptied
		case "squash.interactive":
			s.interactive = !s.interactive
		}
	}
	return s, nil
}

func (s *Operation) View() string {
	return ""
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

func (s *Operation) Name() string {
	return "squash"
}

type Option func(*Operation)

func WithFiles(files []string) Option {
	return func(op *Operation) {
		op.files = files
	}
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions, opts ...Option) *Operation {
	styles := styles{
		dimmed:       common.DefaultPalette.Get("squash dimmed"),
		sourceMarker: common.DefaultPalette.Get("squash source_marker"),
		targetMarker: common.DefaultPalette.Get("squash target_marker"),
	}
	o := &Operation{
		context: context,
		from:    from,
		styles:  styles,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
