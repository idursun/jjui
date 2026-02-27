package set_parents

import (
	"log"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*Model)(nil)
var _ common.Focusable = (*Model)(nil)

type Model struct {
	context  *context.MainContext
	target   *jj.Commit
	current  *jj.Commit
	toRemove map[string]bool
	toAdd    []string
	styles   styles
	parents  []string
}

func (m *Model) IsFocused() bool {
	return true
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case tea.KeyMsg:
		return nil
	}
	return nil
}

func (m *Model) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

type styles struct {
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
	dimmed       lipgloss.Style
}

func (m *Model) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	m.current = commit
	return nil
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump()
	case intents.SetParentsToggleSelect:
		if m.current.GetChangeId() == m.target.GetChangeId() {
			return nil
		}

		if slices.Contains(m.parents, m.current.CommitId) {
			if m.toRemove[m.current.GetChangeId()] {
				delete(m.toRemove, m.current.GetChangeId())
			} else {
				m.toRemove[m.current.GetChangeId()] = true
			}
		} else {
			changeId := m.current.GetChangeId()
			if idx := slices.Index(m.toAdd, changeId); idx >= 0 {
				m.toAdd = append(m.toAdd[:idx], m.toAdd[idx+1:]...)
			} else {
				m.toAdd = append(m.toAdd, changeId)
			}
		}
		return nil
	case intents.Apply:
		if len(m.toAdd) == 0 && len(m.toRemove) == 0 {
			return common.Close
		}

		parentsToAdd := slices.Clone(m.toAdd)
		var parentsToRemove []string

		for changeId := range m.toRemove {
			parentsToRemove = append(parentsToRemove, changeId)
		}

		return m.context.RunCommand(jj.SetParents(m.target.GetChangeId(), parentsToAdd, parentsToRemove), common.RefreshAndSelect(m.target.GetChangeId()), common.Close)
	case intents.Cancel:
		return common.Close
	}
	return nil
}

func (m *Model) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	return actions.ResolveByScopeStrict(m.Scope(), action, args)
}

func (m *Model) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition != operations.RenderBeforeChangeId {
		return ""
	}
	if slices.Contains(m.toAdd, commit.GetChangeId()) {
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

func (m *Model) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (m *Model) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (m *Model) Name() string {
	return "set parents"
}

func (m *Model) Scope() keybindings.Scope {
	return keybindings.Scope(actions.OwnerSetParents)
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
		parents:  parents,
		toRemove: make(map[string]bool),
		toAdd:    []string{},
		target:   to,
		styles:   styles,
	}
}
