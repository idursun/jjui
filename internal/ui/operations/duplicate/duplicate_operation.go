package duplicate

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
)

var (
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

type styles struct {
	changeId     lipgloss.Style
	dimmed       lipgloss.Style
	shortcut     lipgloss.Style
	targetMarker lipgloss.Style
	sourceMarker lipgloss.Style
}

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)

type Operation struct {
	context     *appContext.MainContext
	From        jj.SelectedRevisions
	InsertStart *jj.Commit
	To          *jj.Commit
	Target      Target
	keyMap      config.KeyMappings[key.Binding]
	styles      styles
}

func (r *Operation) IsFocused() bool {
	return true
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return r.HandleKey(msg)
	}
	return nil
}

func (r *Operation) View() string {
	return ""
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.AceJump):
		return common.StartAceJump()
	case key.Matches(msg, r.keyMap.Duplicate.Onto):
		r.Target = TargetDestination
	case key.Matches(msg, r.keyMap.Duplicate.After):
		r.Target = TargetAfter
	case key.Matches(msg, r.keyMap.Duplicate.Before):
		r.Target = TargetBefore
	case key.Matches(msg, r.keyMap.Apply):
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Duplicate(r.From, r.To.GetChangeId(), target), common.RefreshAndSelect(r.From.Last()), common.Close)
	case key.Matches(msg, r.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	r.To = commit
	return nil
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Cancel,
		r.keyMap.Duplicate.After,
		r.keyMap.Duplicate.Before,
		r.keyMap.Duplicate.Onto,
		r.keyMap.AceJump,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		if r.From.Contains(commit) {
			return r.styles.sourceMarker.Render("<< duplicate >>")
		}

		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if r.Target == TargetBefore {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := r.To != nil && r.To.GetChangeId() == commit.GetChangeId()
	if !isSelected {
		return ""
	}

	var ret string
	if r.Target == TargetDestination {
		ret = "onto"
	}
	if r.Target == TargetAfter {
		ret = "after"
	}
	if r.Target == TargetBefore {
		ret = "before"
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		r.styles.targetMarker.Render("<< "+ret+" >>"),
		r.styles.dimmed.Render(" duplicate "),
		r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
		r.styles.dimmed.Render("", ret, ""),
		r.styles.changeId.Render(r.To.GetChangeId()),
	)
}

func (r *Operation) Name() string {
	return "duplicate"
}

func NewOperation(context *appContext.MainContext, from jj.SelectedRevisions, target Target) *Operation {
	styles := styles{
		changeId:     common.DefaultPalette.Get("duplicate change_id"),
		dimmed:       common.DefaultPalette.Get("duplicate dimmed"),
		sourceMarker: common.DefaultPalette.Get("duplicate source_marker"),
		targetMarker: common.DefaultPalette.Get("duplicate target_marker"),
	}
	return &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		From:    from,
		Target:  target,
		styles:  styles,
	}
}
