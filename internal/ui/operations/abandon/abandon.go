package abandon

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ common.Focusable           = (*Operation)(nil)
)

type selectionType int

const (
	selectionTypeRevision selectionType = iota
	selectionTypeDescendants
)

type selections struct {
	items map[string]selectionType
}

type Operation struct {
	context           *context.MainContext
	selectedRevisions jj.SelectedRevisions
	selections        selections
	current           *jj.Commit
	styles            styles
}

type styles struct {
	sourceMarker lipgloss.Style
}

type addSelectionMsg struct {
	jj.SelectedRevisions
}

func (a *Operation) IsFocused() bool {
	return true
}

func (a *Operation) Init() tea.Cmd {
	return nil
}

func (a *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return a.handleIntent(msg)
	case addSelectionMsg:
		a.selectedRevisions = msg.SelectedRevisions
	case tea.KeyMsg:
		return nil
	}
	return nil
}

func (a *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (a *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump()
	case intents.Apply:
		force := intent.(intents.Apply).Force
		if len(a.selectedRevisions.Revisions) == 0 {
			return nil
		}
		return a.context.RunCommand(jj.Abandon(a.selectedRevisions, force), common.Refresh, common.Close)
	case intents.AbandonToggleSelect:
		if a.current == nil {
			return nil
		}
		a.selections.toggle(a.current.GetChangeId(), selectionTypeRevision)
		return a.refreshSelectedRevisionsCmd()
	case intents.AbandonSelectDescendants:
		if a.current == nil {
			return nil
		}
		a.selections.toggle(a.current.GetChangeId(), selectionTypeDescendants)
		return a.refreshSelectedRevisionsCmd()
	case intents.Cancel:
		return common.Close
	}
	return nil
}

func (a *Operation) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	return actions.ResolveByScopeStrict(a.Scope(), action, args)
}

func (a *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	a.current = commit
	return nil
}

func (a *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}
	if a.selections.has(commit.GetChangeId(), selectionTypeDescendants) {
		return a.styles.sourceMarker.Render("<< abandon descendants of >>")
	}
	if a.selections.has(commit.GetChangeId(), selectionTypeRevision) {
		return a.styles.sourceMarker.Render("<< abandon >>")
	}
	return ""
}

func (a *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (a *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (a *Operation) RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	if row.Commit == nil || !a.selectedRevisions.Contains(row.Commit) {
		return ""
	}
	return currentStyle.Strikethrough(true).Render(segment.Text)
}

func (a *Operation) refreshSelectedRevisionsCmd() tea.Cmd {
	revset := a.selections.revset()
	if revset == "" {
		a.selectedRevisions = jj.NewSelectedRevisions()
		return nil
	}
	return func() tea.Msg {
		bytes, err := a.context.RunCommandImmediate(jj.GetIdsFromRevset(revset))
		if err != nil {
			return common.CommandCompletedMsg{Err: err}
		}
		return addSelectionMsg{selectedRevisionsFromOutput(bytes)}
	}
}

func (a *Operation) Name() string {
	return "abandon"
}

func (a *Operation) Scope() keybindings.Scope {
	return actions.OwnerAbandon
}

func NewOperation(context *context.MainContext, selectedRevisions jj.SelectedRevisions) *Operation {
	styles := styles{
		sourceMarker: common.DefaultPalette.Get("abandon source_marker"),
	}
	selectionItems := make(map[string]selectionType, len(selectedRevisions.Revisions))
	for _, revision := range selectedRevisions.Revisions {
		if revision == nil {
			continue
		}
		changeId := revision.GetChangeId()
		if changeId == "" {
			continue
		}
		selectionItems[changeId] = selectionTypeRevision
	}
	return &Operation{
		context:           context,
		selectedRevisions: selectedRevisions,
		selections:        selections{items: selectionItems},
		styles:            styles,
	}
}

func (s *selections) toggle(changeId string, t selectionType) {
	if changeId == "" {
		return
	}
	if s.items == nil {
		s.items = make(map[string]selectionType)
	}
	if existing, ok := s.items[changeId]; ok && existing == t {
		delete(s.items, changeId)
		return
	}
	s.items[changeId] = t
}

func (s *selections) revset() string {
	parts := make([]string, 0, len(s.items))
	for changeId, t := range s.items {
		if changeId == "" {
			continue
		}
		if t == selectionTypeDescendants {
			parts = append(parts, changeId+"::")
			continue
		}
		parts = append(parts, changeId)
	}
	return strings.Join(parts, " | ")
}

func (s *selections) has(changeId string, t selectionType) bool {
	if s.items == nil {
		return false
	}
	selectedType, ok := s.items[changeId]
	return ok && selectedType == t
}

func selectedRevisionsFromOutput(output []byte) jj.SelectedRevisions {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return jj.NewSelectedRevisions()
	}
	ids := strings.Split(trimmed, "\n")
	revisions := make([]*jj.Commit, 0, len(ids))
	for _, id := range ids {
		revisions = append(revisions, &jj.Commit{ChangeId: id})
	}
	return jj.NewSelectedRevisions(revisions...)
}
