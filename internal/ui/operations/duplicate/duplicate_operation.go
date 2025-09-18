package duplicate

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context        *appContext.MainContext
	From           jj.SelectedRevisions
	InsertStart    *models.RevisionItem
	To             *models.RevisionItem
	Target         jj.Target
	keyMap         config.KeyMappings[key.Binding]
	styles         styles
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if cmd := r.HandleKey(msg); cmd != nil {
			return r, cmd
		}
	case common.RefreshMsg:
		r.setSelectedRevision()
		return r, nil
	}
	return r, nil
}

func (r *Operation) View() string {
	return ""
}

func (r *Operation) GetId() view.ViewId {
	return "duplicate"
}

func (r *Operation) Mount(v *view.ViewNode) {
	r.ViewNode = v
	v.Id = "duplicate"
	delegatedViewId := view.RevisionsViewId
	v.KeyDelegation = &delegatedViewId
	v.NeedsRefresh = true
}

type styles struct {
	changeId     lipgloss.Style
	dimmed       lipgloss.Style
	shortcut     lipgloss.Style
	targetMarker lipgloss.Style
	sourceMarker lipgloss.Style
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.Duplicate.Onto):
		r.Target = jj.TargetDestination
	case key.Matches(msg, r.keyMap.Duplicate.After):
		r.Target = jj.TargetAfter
	case key.Matches(msg, r.keyMap.Duplicate.Before):
		r.Target = jj.TargetBefore
	case key.Matches(msg, r.keyMap.Apply):
		r.ViewManager.UnregisterView(r.GetId())
		return r.context.RunCommand(jj.Args(jj.DuplicateArgs{From: r.From, To: *r.To, Target: r.Target}), common.RefreshAndSelect(r.From.Last()))
	case key.Matches(msg, r.keyMap.Cancel):
		r.ViewManager.UnregisterView(r.GetId())
		return nil
	}
	return nil
}

func (r *Operation) setSelectedRevision() {
	current := r.context.Revisions.Current()
	if current == nil {
		r.To = nil
		return
	}
	r.To = current
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Cancel,
		r.keyMap.Duplicate.After,
		r.keyMap.Duplicate.Before,
		r.keyMap.Duplicate.Onto,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
}

func (r *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		if r.From.Contains(commit) {
			return r.styles.sourceMarker.Render("<< duplicate >>")
		}

		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if r.Target == jj.TargetBefore {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := r.To != nil && r.To.Commit.GetChangeId() == commit.GetChangeId()
	if !isSelected {
		return ""
	}

	var ret string
	if r.Target == jj.TargetDestination {
		ret = "onto"
	}
	if r.Target == jj.TargetAfter {
		ret = "after"
	}
	if r.Target == jj.TargetBefore {
		ret = "before"
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		r.styles.targetMarker.Render("<< "+ret+" >>"),
		r.styles.dimmed.Render(" duplicate "),
		r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
		r.styles.dimmed.Render("", ret, ""),
		r.styles.changeId.Render(r.To.Commit.GetChangeId()),
	)
}

func NewOperation(context *appContext.MainContext, from jj.SelectedRevisions) view.IViewModel {
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
		Target:  jj.TargetDestination,
		styles:  styles,
	}
}
