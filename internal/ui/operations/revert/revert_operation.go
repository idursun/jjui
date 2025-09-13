package revert

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
	TargetInsert
)

var (
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

var _ view.IViewModel = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *models.Commit
	To             *models.Commit
	Target         Target
	keyMap         config.KeyMappings[key.Binding]
	highlightedIds []string
	styles         styles
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if cmd := o.HandleKey(msg); cmd != nil {
			return o, cmd
		}
	case common.RefreshMsg:
		o.setSelectedRevision()
		return o, nil
	}
	return o, nil
}

func (o *Operation) View() string {
	return ""
}

func (o *Operation) GetId() view.ViewId {
	return "revert"
}

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Id = o.GetId()
	delegatedViewId := view.RevisionsViewId
	v.KeyDelegation = &delegatedViewId
	v.NeedsRefresh = true
}

type styles struct {
	shortcut     lipgloss.Style
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
	changeId     lipgloss.Style
	text         lipgloss.Style
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, o.keyMap.Revert.Onto):
		o.Target = TargetDestination
	case key.Matches(msg, o.keyMap.Revert.After):
		o.Target = TargetAfter
	case key.Matches(msg, o.keyMap.Revert.Before):
		o.Target = TargetBefore
	case key.Matches(msg, o.keyMap.Revert.Insert):
		o.Target = TargetInsert
		o.InsertStart = o.To
	case key.Matches(msg, o.keyMap.Apply):
		o.ViewManager.UnregisterView(o.GetId())
		if o.Target == TargetInsert {
			return o.context.RunCommand(jj.RevertInsert(o.From, o.InsertStart.GetChangeId(), o.To.GetChangeId()), common.RefreshAndSelect(o.From.Last()))
		} else {
			source := "--revisions"
			target := targetToFlags[o.Target]
			return o.context.RunCommand(jj.Revert(o.From, o.To.GetChangeId(), source, target), common.RefreshAndSelect(o.From.Last()))
		}
	case key.Matches(msg, o.keyMap.Cancel):
		o.ViewManager.UnregisterView(o.GetId())
		return nil
	}
	return nil
}

func (o *Operation) setSelectedRevision() {
	current := o.context.Revisions.Current()
	if current == nil {
		return
	}
	o.highlightedIds = nil
	o.To = current.Commit
	o.highlightedIds = o.From.GetIds()
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.Revert.Before,
		o.keyMap.Revert.After,
		o.keyMap.Revert.Onto,
		o.keyMap.Revert.Insert,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		if slices.Contains(o.highlightedIds, changeId) {
			return o.styles.sourceMarker.Render("<< revert >>")
		}
		if o.Target == TargetInsert && o.InsertStart.GetChangeId() == commit.GetChangeId() {
			return o.styles.sourceMarker.Render("<< after this >>")
		}
		if o.Target == TargetInsert && o.To.GetChangeId() == commit.GetChangeId() {
			return o.styles.sourceMarker.Render("<< before this >>")
		}
		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if o.Target == TargetBefore || o.Target == TargetInsert {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := o.To != nil && o.To.GetChangeId() == commit.GetChangeId()
	if !isSelected {
		return ""
	}

	var source string
	isMany := len(o.From) > 0
	switch {
	case isMany:
		source = "revisions "
	default:
		source = "revision "
	}
	var ret string
	if o.Target == TargetDestination {
		ret = "onto"
	}
	if o.Target == TargetAfter {
		ret = "after"
	}
	if o.Target == TargetBefore {
		ret = "before"
	}
	if o.Target == TargetInsert {
		ret = "insert"
	}

	if o.Target == TargetInsert {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			o.styles.targetMarker.Render("<< insert >>"),
			" ",
			o.styles.dimmed.Render(source),
			o.styles.changeId.Render(strings.Join(o.From.GetIds(), " ")),
			o.styles.dimmed.Render(" between "),
			o.styles.changeId.Render(o.InsertStart.GetChangeId()),
			o.styles.dimmed.Render(" and "),
			o.styles.changeId.Render(o.To.GetChangeId()),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		o.styles.targetMarker.Render("<< "+ret+" >>"),
		o.styles.dimmed.Render(" revert "),
		o.styles.dimmed.Render(source),
		o.styles.changeId.Render(strings.Join(o.From.GetIds(), " ")),
		o.styles.dimmed.Render(" "),
		o.styles.dimmed.Render(ret),
		o.styles.dimmed.Render(" "),
		o.styles.changeId.Render(o.To.GetChangeId()),
	)
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions, target Target) view.IViewModel {
	styles := styles{
		changeId:     common.DefaultPalette.Get("revert change_id"),
		shortcut:     common.DefaultPalette.Get("revert shortcut"),
		dimmed:       common.DefaultPalette.Get("revert dimmed"),
		sourceMarker: common.DefaultPalette.Get("revert source_marker"),
		targetMarker: common.DefaultPalette.Get("revert target_marker"),
	}
	return &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		From:    from,
		Target:  target,
		styles:  styles,
	}
}
