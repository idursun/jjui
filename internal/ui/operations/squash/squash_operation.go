package squash

import (
	"slices"

	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
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
	switch msg := msg.(type) {
	case intents.Intent:
		return s.handleIntent(msg)
	case tea.KeyMsg:
		return s.HandleKey(msg)
	default:
		return nil
	}
}

func (s *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump()
	case intents.Apply:
		args := jj.Squash(s.from, s.current.GetChangeId(), s.files, s.keepEmptied, s.useDestinationMessage, s.interactive, intent.Force)
		continuation := common.RefreshAndSelect(s.current.GetChangeId())
		if s.interactive || !s.useDestinationMessage {
			return tea.Batch(common.Close, s.context.RunInteractiveCommand(args, continuation))
		}
		return tea.Batch(common.Close, s.context.RunCommand(args, continuation))
	case intents.Cancel:
		return common.Close
	case intents.SquashToggleKeepEmptied:
		s.keepEmptied = !s.keepEmptied
	case intents.SquashToggleUseDestinationMessage:
		s.useDestinationMessage = !s.useDestinationMessage
	case intents.SquashToggleInteractive:
		s.interactive = !s.interactive
	default:
		return nil
	}
	return nil
}

func (s *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (s *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, s.keyMap.AceJump):
		return s.handleIntent(intents.StartAceJump{})
	case key.Matches(msg, s.keyMap.Apply, s.keyMap.ForceApply):
		return s.handleIntent(intents.Apply{Force: key.Matches(msg, s.keyMap.ForceApply)})
	case key.Matches(msg, s.keyMap.Cancel):
		return s.handleIntent(intents.Cancel{})
	case key.Matches(msg, s.keyMap.Squash.KeepEmptied):
		return s.handleIntent(intents.SquashToggleKeepEmptied{})
	case key.Matches(msg, s.keyMap.Squash.UseDestinationMessage):
		return s.handleIntent(intents.SquashToggleUseDestinationMessage{})
	case key.Matches(msg, s.keyMap.Squash.Interactive):
		return s.handleIntent(intents.SquashToggleInteractive{})
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

func (s *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (s *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
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
		s.keyMap.AceJump,
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
