package squash

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/routing"
)

var (
	_ operations.Operation  = (*Operation)(nil)
	_ common.Focusable      = (*Operation)(nil)
	_ common.Overlay        = (*Operation)(nil)
	_ common.Editable       = (*Operation)(nil)
	_ routing.ScopeProvider = (*Operation)(nil)
)

type Operation struct {
	context               *context.MainContext
	from                  jj.SelectedRevisions
	files                 []string
	current               *jj.Commit
	targetName            string
	targetPicker          *target_picker.Model
	keepEmptied           bool
	useDestinationMessage bool
	interactive           bool
	styles                styles
}

func (s *Operation) IsFocused() bool {
	return true
}

func (s *Operation) IsEditing() bool {
	return s.targetPicker != nil
}

func (s *Operation) IsOverlay() bool {
	return s.targetPicker != nil
}

func (s *Operation) Scopes() []routing.Scope {
	var ret []routing.Scope
	if s.targetPicker != nil {
		ret = append(ret, routing.Scope{
			Name:      actions.ScopeTargetPicker,
			AllowLeak: false,
			Handler:   s.targetPicker,
		})
	}
	ret = append(ret, routing.Scope{
		Name:      actions.ScopeSquash,
		AllowLeak: true,
		Handler:   s,
	})
	return ret
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
	case target_picker.TargetSelectedMsg:
		s.targetPicker = nil
		s.targetName = strings.TrimSpace(msg.Target)
		cmd, _ := s.HandleIntent(intents.Apply{Force: msg.Force})
		return cmd
	case target_picker.TargetPickerCancelMsg:
		s.targetPicker = nil
		return nil
	case intents.Intent:
		if s.targetPicker != nil {
			switch msg.(type) {
			case intents.TargetPickerNavigate, intents.TargetPickerApply, intents.TargetPickerCancel:
				return s.targetPicker.Update(msg)
			}
		}
		cmd, _ := s.HandleIntent(msg)
		return cmd
	case tea.KeyMsg:
		if s.targetPicker != nil {
			return s.targetPicker.Update(msg)
		}
		return nil
	default:
		if s.targetPicker != nil {
			return s.targetPicker.Update(msg)
		}
		return nil
	}
}

func (s *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump(), true
	case intents.Apply:
		args := jj.Squash(s.from, s.targetArg(), s.files, s.keepEmptied, s.useDestinationMessage, s.interactive, intent.Force)
		continuation := common.RefreshAndSelect(s.current.GetChangeId())
		if s.interactive || !s.useDestinationMessage {
			return tea.Batch(common.CloseApplied, s.context.RunInteractiveCommand(args, continuation)), true
		}
		return tea.Batch(common.CloseApplied, s.context.RunCommand(args, continuation)), true
	case intents.SquashOpenTargetPicker:
		s.targetPicker = target_picker.NewModel(s.context)
		return s.targetPicker.Init(), true
	case intents.Cancel:
		return common.Close, true
	case intents.SquashToggleOption:
		switch intent.Option {
		case intents.SquashOptionKeepEmptied:
			s.keepEmptied = !s.keepEmptied
		case intents.SquashOptionUseDestinationMessage:
			s.useDestinationMessage = !s.useDestinationMessage
		case intents.SquashOptionInteractive:
			s.interactive = !s.interactive
		}
		return nil, true
	}
	return nil, false
}

func (s *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if s.targetPicker != nil {
		s.targetPicker.ViewRect(dl, box)
	}
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

func (s *Operation) targetArg() string {
	if strings.TrimSpace(s.targetName) != "" {
		return s.targetName
	}
	if s.current != nil {
		return s.current.GetChangeId()
	}
	return ""
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
