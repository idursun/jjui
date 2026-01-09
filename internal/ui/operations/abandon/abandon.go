package abandon

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var (
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ common.Focusable           = (*Operation)(nil)
)

type Operation struct {
	context           *context.MainContext
	selectedRevisions jj.SelectedRevisions
	current           *jj.Commit
	keyMap            config.KeyMappings[key.Binding]
	styles            styles
}

type styles struct {
	sourceMarker lipgloss.Style
}

func (a *Operation) IsFocused() bool {
	return true
}

func (a *Operation) Init() tea.Cmd {
	return nil
}

func (a *Operation) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return a.HandleKey(msg)
	}
	return nil
}

func (a *Operation) View() string {
	return ""
}

func (a *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, a.keyMap.AceJump):
		return common.StartAceJump()
	case key.Matches(msg, a.keyMap.Apply):
		if len(a.selectedRevisions.Revisions) == 0 {
			return nil
		}
		return a.context.RunCommand(jj.Abandon(a.selectedRevisions, false), common.Refresh, common.Close)
	case key.Matches(msg, a.keyMap.ForceApply):
		if len(a.selectedRevisions.Revisions) == 0 {
			return nil
		}
		return a.context.RunCommand(jj.Abandon(a.selectedRevisions, true), common.Refresh, common.Close)
	case key.Matches(msg, a.keyMap.ToggleSelect):
		if a.current == nil {
			return nil
		}
		item := context.SelectedRevision{
			ChangeId: a.current.GetChangeId(),
			CommitId: a.current.CommitId,
		}
		a.context.ToggleCheckedItem(item)
		a.toggleSelectedRevision(a.current)
		return nil
	case key.Matches(msg, a.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (a *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		a.keyMap.Apply,
		a.keyMap.ForceApply,
		a.keyMap.ToggleSelect,
		a.keyMap.Cancel,
		a.keyMap.AceJump,
	}
}

func (a *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{a.ShortHelp()}
}

func (a *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	a.current = commit
	return nil
}

func (a *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}
	if !a.selectedRevisions.Contains(commit) {
		return ""
	}
	return a.styles.sourceMarker.Render("<< abandon >>")
}

func (a *Operation) RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	if row.Commit == nil || !a.selectedRevisions.Contains(row.Commit) {
		return ""
	}
	return currentStyle.Strikethrough(true).Render(segment.Text)
}

func (a *Operation) toggleSelectedRevision(commit *jj.Commit) {
	if commit == nil {
		return
	}
	if a.selectedRevisions.Contains(commit) {
		var kept []*jj.Commit
		for _, revision := range a.selectedRevisions.Revisions {
			if revision.GetChangeId() != commit.GetChangeId() {
				kept = append(kept, revision)
			}
		}
		a.selectedRevisions = jj.NewSelectedRevisions(kept...)
		return
	}
	a.selectedRevisions = jj.NewSelectedRevisions(append(a.selectedRevisions.Revisions, commit)...)
}

func (a *Operation) Name() string {
	return "abandon"
}

func NewOperation(context *context.MainContext, selectedRevisions jj.SelectedRevisions) *Operation {
	styles := styles{
		sourceMarker: common.DefaultPalette.Get("abandon source_marker"),
	}
	return &Operation{
		context:           context,
		selectedRevisions: selectedRevisions,
		keyMap:            config.Current.GetKeyMap(),
		styles:            styles,
	}
}
