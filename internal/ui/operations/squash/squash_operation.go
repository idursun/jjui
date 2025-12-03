package squash

import (
	"slices"

	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var (
	_ operations.Operation = (*Operation)(nil)
	_ common.Focusable     = (*Operation)(nil)
)

type Operation struct {
	context               *context.MainContext
	from                  jj.SelectedRevisions
	files                 []string
	current               *jj.Commit
	keyMap                config.KeyMappings[key.Binding]
	keepEmptied           bool
	useDestinationMessage bool
	interactive           bool
	styles                styles
}

func (s *Operation) IsFocused() bool {
	return true
}

type styles struct {
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
}

func (s *Operation) Init() tea.Cmd {
	return nil
}

func (s *Operation) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return s.HandleKey(msg)
	}
	return nil
}

func (s *Operation) View() string {
	return ""
}

func (s *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, s.keyMap.Apply, s.keyMap.ForceApply):
		ignoreImmutable := key.Matches(msg, s.keyMap.ForceApply)
		args := jj.Squash(s.from, s.current.GetChangeId(), s.files, s.keepEmptied, s.useDestinationMessage, s.interactive, ignoreImmutable)
		continuation := common.RefreshAndSelect(s.current.GetChangeId())
		if s.interactive || !s.useDestinationMessage {
			return tea.Batch(common.Close, s.context.RunInteractiveCommand(args, continuation))
		} else {
			return tea.Batch(common.Close, s.context.RunCommand(args, continuation))
		}
	case key.Matches(msg, s.keyMap.Cancel):
		return common.Close
	case key.Matches(msg, s.keyMap.Squash.KeepEmptied):
		s.keepEmptied = !s.keepEmptied
	case key.Matches(msg, s.keyMap.Squash.UseDestinationMessage):
		s.useDestinationMessage = !s.useDestinationMessage
	case key.Matches(msg, s.keyMap.Squash.Interactive):
		s.interactive = !s.interactive
	}
	return nil
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	s.current = commit
	return nil
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}

	isSelected := s.current != nil && s.current.GetChangeId() == commit.GetChangeId()
	if isSelected {
		marker := "<< into >>"
		if s.useDestinationMessage {
			marker = "<< use this message >>"
		}
		return s.styles.targetMarker.Render(marker)
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

func (s *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		s.keyMap.Apply,
		s.keyMap.ForceApply,
		s.keyMap.Cancel,
		s.keyMap.Squash.KeepEmptied,
		s.keyMap.Squash.UseDestinationMessage,
		s.keyMap.Squash.Interactive,
	}
}

func (s *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
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
		keyMap:  config.Current.GetKeyMap(),
		from:    from,
		styles:  styles,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
