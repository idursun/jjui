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

var _ view.IViewModel = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *models.RevisionItem
	To             *models.RevisionItem
	Target         jj.Target
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
		o.Target = jj.TargetDestination
	case key.Matches(msg, o.keyMap.Revert.After):
		o.Target = jj.TargetAfter
	case key.Matches(msg, o.keyMap.Revert.Before):
		o.Target = jj.TargetBefore
	case key.Matches(msg, o.keyMap.Revert.Insert):
		o.Target = jj.TargetInsert
		o.InsertStart = o.To
	case key.Matches(msg, o.keyMap.Apply):
		o.ViewManager.UnregisterView(o.GetId())
		if o.Target == jj.TargetInsert {
			//return o.context.RunCommand(jj.RevertInsert(o.From, o.InsertStart.GetChangeId(), o.To.GetChangeId()), common.RefreshAndSelect(o.From.Last()))
			return o.context.RunCommand(jj.Args(jj.RevertInsertArgs{From: o.From, InsertAfter: *o.InsertStart, InsertBefore: *o.To}), common.RefreshAndSelect(o.From.Last()))
		} else {
			return o.context.RunCommand(jj.Args(jj.RevertArgs{
				From:            o.From,
				To:              *o.To,
				Target:          o.Target,
				GlobalArguments: jj.GlobalArguments{IgnoreImmutable: false},
			}), common.RefreshAndSelect(o.From.Last()))
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
	o.To = current
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
		if o.Target == jj.TargetInsert && o.InsertStart.Commit.GetChangeId() == commit.GetChangeId() {
			return o.styles.sourceMarker.Render("<< after this >>")
		}
		if o.Target == jj.TargetInsert && o.To.Commit.GetChangeId() == commit.GetChangeId() {
			return o.styles.sourceMarker.Render("<< before this >>")
		}
		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if o.Target == jj.TargetBefore || o.Target == jj.TargetInsert {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := o.To != nil && o.To.Commit.GetChangeId() == commit.GetChangeId()
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
	if o.Target == jj.TargetDestination {
		ret = "onto"
	}
	if o.Target == jj.TargetAfter {
		ret = "after"
	}
	if o.Target == jj.TargetBefore {
		ret = "before"
	}
	if o.Target == jj.TargetInsert {
		ret = "insert"
	}

	if o.Target == jj.TargetInsert {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			o.styles.targetMarker.Render("<< insert >>"),
			" ",
			o.styles.dimmed.Render(source),
			o.styles.changeId.Render(strings.Join(o.From.GetIds(), " ")),
			o.styles.dimmed.Render(" between "),
			o.styles.changeId.Render(o.InsertStart.Commit.GetChangeId()),
			o.styles.dimmed.Render(" and "),
			o.styles.changeId.Render(o.To.Commit.GetChangeId()),
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
		o.styles.changeId.Render(o.To.Commit.GetChangeId()),
	)
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions) view.IViewModel {
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
		Target:  jj.TargetDestination,
		styles:  styles,
	}
}
