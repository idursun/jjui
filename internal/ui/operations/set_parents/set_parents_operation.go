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
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context  *context.MainContext
	target   *jj.Commit
	current  *jj.Commit
	toRemove map[string]bool
	toAdd    map[string]bool
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
	return "set_parents"
}

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Id = "set_parents"
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
	current := o.context.Revisions.Current()
	if current == nil {
		return
	}
	o.current = current.Commit
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, o.keyMap.ToggleSelect):
		if o.current.GetChangeId() == o.target.GetChangeId() {
			return nil
		}

		if slices.Contains(o.parents, o.current.CommitId) {
			if o.toRemove[o.current.GetChangeId()] {
				delete(o.toRemove, o.current.GetChangeId())
			} else {
				o.toRemove[o.current.GetChangeId()] = true
			}
		} else {
			if o.toAdd[o.current.GetChangeId()] {
				delete(o.toAdd, o.current.GetChangeId())
			} else {
				o.toAdd[o.current.GetChangeId()] = true
			}
		}
	case key.Matches(msg, o.keyMap.Apply):
		if len(o.toAdd) == 0 && len(o.toRemove) == 0 {
			return common.Close
		}

		var parentsToAdd []string
		var parentsToRemove []string

		for changeId := range o.toAdd {
			parentsToAdd = append(parentsToAdd, changeId)
		}

		for changeId := range o.toRemove {
			parentsToRemove = append(parentsToRemove, changeId)
		}

		o.ViewManager.UnregisterView(o.GetId())
		return o.context.RunCommand(jj.SetParents(o.target.GetChangeId(), parentsToAdd, parentsToRemove), common.RefreshAndSelect(o.target.GetChangeId()))
	case key.Matches(msg, o.keyMap.Cancel):
		o.ViewManager.UnregisterView(o.GetId())
		return nil
	}
	return nil
}

func (o *Operation) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition != operations.RenderBeforeChangeId {
		return ""
	}
	if o.toAdd[commit.GetChangeId()] {
		return o.styles.sourceMarker.Render("<< add >>")
	}
	if o.toRemove[commit.GetChangeId()] {
		return o.styles.sourceMarker.Render("<< remove >>")
	}

	if slices.Contains(o.parents, commit.CommitId) {
		return o.styles.dimmed.Render("<< parent >>")
	}
	if commit.GetChangeId() == o.target.GetChangeId() {
		return o.styles.targetMarker.Render("<< to >>")
	}
	return ""
}

func NewOperation(ctx *context.MainContext, to *jj.Commit) view.IViewModel {
	styles := styles{
		sourceMarker: common.DefaultPalette.Get("set_parents source_marker"),
		targetMarker: common.DefaultPalette.Get("set_parents target_marker"),
		dimmed:       common.DefaultPalette.Get("set_parents dimmed"),
	}
	output, err := ctx.RunCommandImmediate(jj.GetParents(to.GetChangeId()))
	if err != nil {
		log.Println("Failed to get parents for commit", to.GetChangeId())
	}
	parents := strings.Fields(string(output))
	return &Operation{
		context:  ctx,
		keyMap:   config.Current.GetKeyMap(),
		parents:  parents,
		toRemove: make(map[string]bool),
		toAdd:    make(map[string]bool),
		target:   to,
		styles:   styles,
	}
}
