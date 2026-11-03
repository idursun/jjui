package status

import (
	"errors"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

func TestStatus_Update_ExecProcessCompletedMsg(t *testing.T) {
	cases := []struct {
		name           string
		msg            common.ExecProcessCompletedMsg
		expectedMode   string
		expectedPrompt string
		expectedInput  string
		shouldFocus    bool
		expectCmd      bool
	}{
		{
			name: "Execution failed, should restore input",
			msg: common.ExecProcessCompletedMsg{
				Err: errors.New("exit status 1"),
				Msg: common.ExecMsg{
					Line: "invalid command",
					Mode: common.ExecShell,
				},
			},
			expectedMode:   "exec sh",
			expectedPrompt: "$ ",
			expectedInput:  "invalid command",
			shouldFocus:    true,
			expectCmd:      true,
		},
		{
			name: "Execution succeeded",
			msg: common.ExecProcessCompletedMsg{
				Err: nil,
			},
			expectCmd: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &context.MainContext{
				Histories: config.NewHistories(),
			}
			m := New(ctx)
			cmd := m.Update(tt.msg)

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}

			if tt.shouldFocus {
				assert.True(t, m.IsFocused())
				assert.Equal(t, tt.expectedMode, m.mode)
				assert.Equal(t, tt.expectedPrompt, m.input.Prompt)
				assert.Equal(t, tt.expectedInput, m.input.Value())
			} else {
				assert.False(t, m.IsFocused())
			}
		})
	}
}

type statusTestState map[string]any

func (s statusTestState) QueryState(name string) (any, bool) {
	value, ok := s[name]
	return value, ok
}

func (statusTestState) Selection() common.SelectionSnapshot { return common.SelectionSnapshot{} }

func TestSetScopes_KeepsUnavailableBindingsCrossedOutInExpandedHelp(t *testing.T) {
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
	m.SetScopes([]common.Scope{{Name: actions.ScopeRevisions}})

	entries := m.Help()[0].Entries
	assert.True(t, entries[0].Disabled, "diff should remain visible but unavailable")
	assert.False(t, entries[1].Disabled, "refresh is always available")
	assert.True(t, entries[2].Disabled, "selection-only binding should remain visible but unavailable")

	state["revisions.has_revision"] = true
	state["revisions.is_empty"] = false
	state["revisions.has_selection"] = true
	m.SetScopes([]common.Scope{{Name: actions.ScopeRevisions}})
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

func TestModel_ViewStatusRectShowsHelpWhileInputIsFocused(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.SetHelp([]help.Entry{{Label: "enter", Desc: "apply"}})
	m.StartExec(common.ExecShell)

	dl := render.NewDisplayContext()
	m.ViewStatusRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
	rendered := dl.RenderToString(40, 1)

	assert.NotContains(t, rendered, "exec sh")
	assert.Contains(t, rendered, "enter")
	assert.Contains(t, rendered, "apply")
}

func TestModel_ViewStatusRectShowsMode(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.SetMode("diff")

	dl := render.NewDisplayContext()
	m.ViewStatusRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
	rendered := dl.RenderToString(40, 1)

	assert.Contains(t, rendered, "diff")
}

func TestModel_ViewStatusRectExpandedHelpLeavesMainViewVisible(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	entries := make([]help.Entry, 20)
	for i := range entries {
		entries[i] = help.Entry{Label: "k", Desc: "entry"}
	}
	m.SetHelp(entries)
	m.SetStatusExpanded(true)

	dl := render.NewDisplayContext()
	m.ViewStatusRect(dl, layout.NewBox(layout.Rect(0, 4, 20, 1)))
	rendered := dl.RenderToString(20, 5)
	lines := strings.Split(rendered, "\n")

	assert.NotContains(t, lines[0], "entry")
	assert.Contains(t, rendered, "full help")
}

func TestModel_ViewInputRectUsesFullWidth(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.StartExec(common.ExecShell)

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 40, 1))
	m.ViewInputRect(dl, box)

	assert.Equal(t, 27, m.input.Width())
	assert.NotNil(t, dl.Cursor())
}
