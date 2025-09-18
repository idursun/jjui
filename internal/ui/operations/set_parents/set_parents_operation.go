package set_parents

import (
	"log"
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
	context  *context.MainContext
	target   *models.RevisionItem
	current  *models.RevisionItem
	toRemove map[string]models.RevisionItem
	toAdd    map[string]models.RevisionItem
	keyMap   config.KeyMappings[key.Binding]
	styles   styles
	parents  []string
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
	return "set parents"
}

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Id = o.GetId()
	v.NeedsRefresh = true
	keyDelegation := view.RevisionsViewId
	v.KeyDelegation = &keyDelegation
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keyMap.ToggleSelect,
		o.keyMap.Apply,
		o.keyMap.Cancel,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		o.ShortHelp(),
	}
}

type styles struct {
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
	dimmed       lipgloss.Style
}

func (o *Operation) setSelectedRevision() {
	o.current = o.context.Revisions.Current()
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, o.keyMap.ToggleSelect):
		if o.current.Commit.GetChangeId() == o.target.Commit.GetChangeId() {
			return nil
		}

		if slices.Contains(o.parents, o.current.Commit.CommitId) {
			if _, exists := o.toRemove[o.current.Commit.GetChangeId()]; exists {
				delete(o.toRemove, o.current.Commit.GetChangeId())
			} else {
				o.toRemove[o.current.Commit.GetChangeId()] = *o.current
			}
		} else {
			if _, exists := o.toAdd[o.current.Commit.GetChangeId()]; exists {
				delete(o.toAdd, o.current.Commit.GetChangeId())
			} else {
				o.toAdd[o.current.Commit.GetChangeId()] = *o.current
			}
		}
	case key.Matches(msg, o.keyMap.Apply):
		if len(o.toAdd) == 0 && len(o.toRemove) == 0 {
			o.ViewManager.UnregisterView(o.Id)
			return nil
		}
		var parentToAdd []models.RevisionItem
		for _, item := range o.toAdd {
			parentToAdd = append(parentToAdd, item)
		}
		var parentsToRemove []models.RevisionItem
		for _, item := range o.toRemove {
			parentsToRemove = append(parentsToRemove, item)
		}

		o.ViewManager.UnregisterView(o.GetId())
		return o.context.RunCommand(jj.Args(jj.RebaseSetParentsArgs{To: *o.target, ParentsToAdd: parentToAdd, ParentsToRemove: parentsToRemove}), common.RefreshAndSelect(o.target.Commit.GetChangeId()))
	case key.Matches(msg, o.keyMap.Cancel):
		o.ViewManager.UnregisterView(o.GetId())
		return nil
	}
	return nil
}

func (o *Operation) Render(commit *models.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition != operations.RenderBeforeChangeId {
		return ""
	}
	if _, exists := o.toAdd[commit.GetChangeId()]; exists {
		return o.styles.sourceMarker.Render("<< add >>")
	}
	if _, exists := o.toRemove[commit.GetChangeId()]; exists {
		return o.styles.sourceMarker.Render("<< remove >>")
	}

	if slices.Contains(o.parents, commit.CommitId) {
		return o.styles.dimmed.Render("<< parent >>")
	}
	if commit.GetChangeId() == o.target.Commit.GetChangeId() {
		return o.styles.targetMarker.Render("<< to >>")
	}
	return ""
}

func NewOperation(ctx *context.MainContext, to *models.RevisionItem) view.IViewModel {
	styles := styles{
		sourceMarker: common.DefaultPalette.Get("set_parents source_marker"),
		targetMarker: common.DefaultPalette.Get("set_parents target_marker"),
		dimmed:       common.DefaultPalette.Get("set_parents dimmed"),
	}
	output, err := ctx.RunCommandImmediate(jj.GetParents(to.Commit.GetChangeId()).GetArgs())
	if err != nil {
		log.Println("Failed to get parents for commit", to.Commit.GetChangeId())
	}
	parents := strings.Fields(string(output))
	return &Operation{
		context:  ctx,
		keyMap:   config.Current.GetKeyMap(),
		parents:  parents,
		toRemove: make(map[string]models.RevisionItem),
		toAdd:    make(map[string]models.RevisionItem),
		target:   to,
		styles:   styles,
	}
}
