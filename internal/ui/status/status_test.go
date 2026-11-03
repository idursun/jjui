package status

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

type statusTestState map[string]any

func (s statusTestState) QueryState(name string) (any, bool) {
	value, ok := s[name]
	return value, ok
}

func (statusTestState) Selection() common.SelectionSnapshot { return common.SelectionSnapshot{} }

func TestSync_KeepsUnavailableBindingsCrossedOutInExpandedHelp(t *testing.T) {
	previous := config.Current
	t.Cleanup(func() { config.Current = previous })
	config.Current = &config.Config{Bindings: []config.BindingConfig{
		{Action: "revisions.diff", Scope: "revisions", Key: config.StringList{"d"}},
		{Action: "revisions.refresh", Scope: "revisions", Key: config.StringList{"r"}},
		{Action: "custom.selection", Scope: "revisions", Key: config.StringList{"x"}, When: "revisions.has_selection"},
	}}

	state := statusTestState{
		"revisions.has_revision":  false,
		"revisions.is_empty":      true,
		"revisions.has_selection": false,
	}
	ctx := &context.MainContext{Histories: config.NewHistories()}
	ctx.SetStateProvider(state)
	m := New(ctx)
	m.Sync([]common.Scope{{Name: actions.ScopeRevisions}}, nil)

	entries := m.Help()[0].Entries
	assert.True(t, entries[0].Disabled, "diff should remain visible but unavailable")
	assert.False(t, entries[1].Disabled, "refresh is always available")
	assert.True(t, entries[2].Disabled, "selection-only binding should remain visible but unavailable")

	state["revisions.has_revision"] = true
	state["revisions.is_empty"] = false
	state["revisions.has_selection"] = true
	m.Sync([]common.Scope{{Name: actions.ScopeRevisions}}, nil)
	for _, entry := range m.Help()[0].Entries {
		assert.False(t, entry.Disabled, "%s became available", entry.Label)
	}
}

func TestModel_BuildHelpGrid_ColumnMajorOrder(t *testing.T) {
	m := &Model{}
	entries := []string{"A", "B", "C", "D", "E", "F"}

	lines := m.buildHelpGrid(entries, 1, 9)

	assert.Equal(t, []string{
		"A  C  E",
		"B  D  F",
	}, lines)
}

func TestModel_CollectGroupEntriesStrikesDisabledAndOverriddenEntries(t *testing.T) {
	m := &Model{}
	shortcutStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	dimmedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	entries := []help.Entry{
		{Label: "x", Desc: "unavailable", Disabled: true},
		{Label: "y", Desc: "overridden", Overridden: true},
	}

	rendered, _ := m.collectGroupEntries(entries, shortcutStyle, dimmedStyle)
	assert.Equal(t, dimmedStyle.Strikethrough(true).Render("x unavailable"), rendered[0])
	assert.Equal(t, dimmedStyle.Strikethrough(true).Render("y overridden"), rendered[1])
}

func TestModel_ViewRectShowsMode(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.mode = "diff"

	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
	rendered := dl.RenderToString(40, 1)

	assert.Contains(t, rendered, "diff")
}

func TestWorkspaceIndicatorDoesNotGetClippedByModeLabel(t *testing.T) {
	ctx := &context.MainContext{WorkspaceName: "feature", Histories: config.NewHistories()}
	m := New(ctx)
	m.mode = "revisions"
	for _, width := range []int{25, 80} {
		dl := render.NewDisplayContext()
		m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, width, 1)))
		assert.Contains(t, dl.RenderToString(width, 1), "workspace: feature")
	}
}

func TestModel_PromptScopesHideMode(t *testing.T) {
	for _, scope := range []keybindings.ScopeName{actions.ScopeStatusInput, actions.ScopeQuickSearchInput} {
		m := New(&context.MainContext{Histories: config.NewHistories()})
		m.mode = "revisions"
		m.Sync([]common.Scope{{Name: scope, Leak: common.LeakNone}}, nil)
		assert.Empty(t, m.Mode(), scope)

		m.setHelp([]help.Entry{{Label: "esc", Desc: "cancel"}})
		dl := render.NewDisplayContext()
		m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
		assert.True(t, strings.HasPrefix(dl.RenderToString(40, 1), " esc cancel"), scope)
	}
}

func TestModel_ViewRectExpandedHelpLeavesMainViewVisible(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	entries := make([]help.Entry, 20)
	for i := range entries {
		entries[i] = help.Entry{Label: "k", Desc: "entry"}
	}
	m.setHelp(entries)
	m.SetStatusExpanded(true)

	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 4, 20, 1)))
	rendered := dl.RenderToString(20, 5)
	lines := strings.Split(rendered, "\n")

	assert.NotContains(t, lines[0], "entry")
	assert.Contains(t, rendered, "full help")
}

func TestModel_HintKeysResolveByAction(t *testing.T) {
	groups := []help.ScopeGroup{{Name: "ui", Entries: []help.Entry{
		{Action: "ui.expand_status", Label: "ctrl+h", Desc: "expand status help"},
		{Action: "ui.open_help", Label: "f2", Desc: "help"},
	}}}
	assert.Equal(t, "ctrl+h", expandStatusKey(groups))
	assert.Equal(t, "f2", fullHelpKey(groups))

	groups[0].Entries[0].Overridden = true
	groups[0].Entries[1].Disabled = true
	assert.Equal(t, "?", expandStatusKey(groups))
	assert.Equal(t, "f1", fullHelpKey(groups))
}

func TestModel_HintKeysFollowReboundBindings(t *testing.T) {
	previous := config.Current
	t.Cleanup(func() { config.Current = previous })
	config.Current = &config.Config{Bindings: []config.BindingConfig{
		{Action: "ui.expand_status", Scope: "ui", Key: config.StringList{"ctrl+h"}, Desc: "expand status help"},
		{Action: "ui.open_help", Scope: "ui", Key: config.StringList{"f2"}, Desc: "help"},
	}}
	groups := help.BuildGroupedFromBindings([]keybindings.ScopeName{"ui"}, config.Current.Bindings)
	assert.Equal(t, "ctrl+h", expandStatusKey(groups))
	assert.Equal(t, "f2", fullHelpKey(groups))
}
