package set_parents

import (
	"log"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*Model)(nil)
var _ common.Focusable = (*Model)(nil)
var _ dispatch.ScopeProvider = (*Model)(nil)

type Model struct {
	context  *context.MainContext
	target   *jj.Commit
	current  *jj.Commit
	toRemove map[string]bool
	toAdd    []string
	parents  []string
}

func (m *Model) IsFocused() bool {
	return true
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Scopes() []dispatch.Scope {
	return []dispatch.Scope{
		{
			Name:    actions.ScopeSetParents,
			Leak:    dispatch.LeakAll,
			Handler: m,
		},
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (m *Model) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (m *Model) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	m.current = commit
	return nil
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump(), true
	case intents.SetParentsToggleSelect:
		if m.current.GetChangeId() == m.target.GetChangeId() {
			return nil, true
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
		return nil, true
	case intents.Apply:
		if len(m.toAdd) == 0 && len(m.toRemove) == 0 {
			return common.Close, true
		}

		parentsToAdd := slices.Clone(m.toAdd)
		var parentsToRemove []string

		for changeId := range m.toRemove {
			parentsToRemove = append(parentsToRemove, changeId)
		}

		return m.context.RunCommand(jj.SetParents(m.target.GetChangeId(), parentsToAdd, parentsToRemove), common.RefreshAndSelect(m.target.GetChangeId()), common.CloseApplied), true
	case intents.Cancel:
		return common.Close, true
	}
	return nil, false
}

func (m *Model) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition != operations.RenderBeforeChangeId {
		return ""
	}
	sourceMarker := common.DefaultPalette.Get("set_parents source_marker")
	targetMarker := common.DefaultPalette.Get("set_parents target_marker")
	dimmedStyle := common.DefaultPalette.Get("set_parents dimmed")

	if slices.Contains(m.toAdd, commit.GetChangeId()) {
		return sourceMarker.Render("<< add >>")
	}
	if m.toRemove[commit.GetChangeId()] {
		return sourceMarker.Render("<< remove >>")
	}

	if slices.Contains(m.parents, commit.CommitId) {
		return dimmedStyle.Render("<< parent >>")
	}
	if commit.GetChangeId() == m.target.GetChangeId() {
		return targetMarker.Render("<< to >>")
	}
	return ""
}

func (m *Model) Name() string {
	return "set parents"
}

func NewModel(ctx *context.MainContext, to *jj.Commit) *Model {
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
	}
}
