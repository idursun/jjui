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
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ operations.Operation = (*Model)(nil)
var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	context  *context.MainContext
	target   *jj.Commit
	current  *jj.Commit
	toRemove map[string]bool
	toAdd    map[string]bool
	keyMap   config.KeyMappings[key.Binding]
	styles   styles
	parents  []string
}

func (m *Model) GetActionMap() map[string]actions.Action {
	return map[string]actions.Action{
		"j":      {Id: "revisions.down"},
		"k":      {Id: "revisions.up"},
		" ":      {Id: "set_parents.toggle_select"},
		"enter":  {Id: "set_parents.apply", Next: []actions.Action{{Id: "close set_parents"}}},
		"esc":    {Id: "close set_parents"},
		"ctrl+c": {Id: "close set_parents"},
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "set_parents.toggle_select":
			if m.current.GetChangeId() == m.target.GetChangeId() {
				return m, nil
			}

			if slices.Contains(m.parents, m.current.CommitId) {
				if m.toRemove[m.current.GetChangeId()] {
					delete(m.toRemove, m.current.GetChangeId())
				} else {
					m.toRemove[m.current.GetChangeId()] = true
				}
			} else {
				if m.toAdd[m.current.GetChangeId()] {
					delete(m.toAdd, m.current.GetChangeId())
				} else {
					m.toAdd[m.current.GetChangeId()] = true
				}
			}
			return m, nil
		case "set_parents.apply":
			if len(m.toAdd) == 0 && len(m.toRemove) == 0 {
				return m, nil
			}

			var parentsToAdd []string
			var parentsToRemove []string

			for changeId := range m.toAdd {
				parentsToAdd = append(parentsToAdd, changeId)
			}

			for changeId := range m.toRemove {
				parentsToRemove = append(parentsToRemove, changeId)
			}

			return m, m.context.RunCommand(jj.SetParents(m.target.GetChangeId(), parentsToAdd, parentsToRemove), common.RefreshAndSelect(m.target.GetChangeId()))
		}
	}
	return m, nil
}

func (m *Model) View() string {
	return ""
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keyMap.ToggleSelect,
		m.keyMap.Apply,
		m.keyMap.Cancel,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		m.ShortHelp(),
	}
}

type styles struct {
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
	dimmed       lipgloss.Style
}

func (m *Model) SetSelectedRevision(commit *jj.Commit) {
	m.current = commit
}

func (m *Model) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition != operations.RenderBeforeChangeId {
		return ""
	}
	if m.toAdd[commit.GetChangeId()] {
		return m.styles.sourceMarker.Render("<< add >>")
	}
	if m.toRemove[commit.GetChangeId()] {
		return m.styles.sourceMarker.Render("<< remove >>")
	}

	if slices.Contains(m.parents, commit.CommitId) {
		return m.styles.dimmed.Render("<< parent >>")
	}
	if commit.GetChangeId() == m.target.GetChangeId() {
		return m.styles.targetMarker.Render("<< to >>")
	}
	return ""
}

func (m *Model) Name() string {
	return "set parents"
}

func NewModel(ctx *context.MainContext, to *jj.Commit) *Model {
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
	return &Model{
		context:  ctx,
		keyMap:   config.Current.GetKeyMap(),
		parents:  parents,
		toRemove: make(map[string]bool),
		toAdd:    make(map[string]bool),
		target:   to,
		styles:   styles,
	}
}
