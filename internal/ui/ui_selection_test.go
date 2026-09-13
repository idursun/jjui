package ui

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type selectionReplyMsg struct{ item common.SelectedItem }
type selectionActionResultMsg struct{}

type changingSelectionScope struct {
	scopeOnlyStackedModel
	snapshot common.SelectionSnapshot
	next     common.SelectedItem
	cmd      tea.Cmd
}

func (m *changingSelectionScope) Selection() common.SelectionSnapshot { return m.snapshot }

func (m *changingSelectionScope) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.snapshot.Highlighted = m.next
	case selectionReplyMsg:
		m.snapshot.Highlighted = msg.item
	}
	return nil
}

func (m *changingSelectionScope) HandleIntent(intents.Intent) (tea.Cmd, bool) {
	m.snapshot.Highlighted = m.next
	return m.cmd, true
}

// Scopes must use this handler, rather than the embedded scope-only fixture.
func (m *changingSelectionScope) Scopes() []common.Scope {
	scopes := m.scopeOnlyStackedModel.Scopes()
	scopes[0].Handler = m
	return scopes
}

func TestUpdateDetectsHighlightChangesAcrossReturnPaths(t *testing.T) {
	oldBindings := config.Current.Bindings
	t.Cleanup(func() { config.Current.Bindings = oldBindings })
	config.Current.Bindings = []config.BindingConfig{
		{Action: "choose.move_down", Scope: "choose", Key: config.StringList{"j"}},
	}
	first := common.SelectedRevision{ChangeId: "first", CommitId: "commit1"}
	for _, tc := range []struct {
		name string
		msg  tea.Msg
	}{
		{"routed intent", tea.KeyPressMsg{Code: 'j', Text: "j"}},
		{"blocking scope helper", tea.KeyPressMsg{Code: 'x', Text: "x"}},
		{"async reply", selectionReplyMsg{item: first}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))
			model := NewUI(ctx)
			model.stacked = &changingSelectionScope{
				scopeOnlyStackedModel: scopeOnlyStackedModel{scope: actions.ScopeChoose},
				next:                  first,
			}
			cmd := model.Update(tc.msg)
			assert.Equal(t, first, ctx.Selection().Highlighted)
			var changes []common.SelectionChangedMsg
			test.SimulateModel(model, cmd, func(msg tea.Msg) {
				if msg, ok := msg.(common.SelectionChangedMsg); ok {
					changes = append(changes, msg)
				}
			})
			assert.Equal(t, []common.SelectionChangedMsg{{Item: first}}, changes,
				"nested handling and delivery of the notification must not duplicate it")
		})
	}
}

func TestUpdateNotifiesWhenEarlyReturnRemovesHighlight(t *testing.T) {
	for _, msg := range []tea.Msg{common.CloseViewMsg{}, common.ShowInputMsg{}} {
		t.Run(reflect.TypeOf(msg).Name(), func(t *testing.T) {
			model := NewUI(test.NewTestContext(test.NewTestCommandRunner(t)))
			model.stacked = &changingSelectionScope{snapshot: common.SelectionSnapshot{
				Highlighted: common.SelectedRevision{ChangeId: "first"},
			}}
			require.NotNil(t, model.Update(tea.ModeReportMsg{}))
			var changes []common.SelectionChangedMsg
			test.SimulateModel(model, model.Update(msg), func(msg tea.Msg) {
				if msg, ok := msg.(common.SelectionChangedMsg); ok {
					changes = append(changes, msg)
				}
			})
			assert.Equal(t, []common.SelectionChangedMsg{{}}, changes)
		})
	}
}

func TestDirectHandleIntentLeavesNotificationToUpdate(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := NewUI(ctx)
	model.stacked = &changingSelectionScope{snapshot: common.SelectionSnapshot{
		Highlighted: common.SelectedRevision{ChangeId: "first"},
	}}
	require.NotNil(t, model.Update(tea.ModeReportMsg{}))
	// Opening history replaces the selection provider synchronously.
	_, handled := model.HandleIntent(intents.CommandHistoryToggle{})
	require.True(t, handled)
	assert.Nil(t, ctx.Selection().Highlighted, "live reads do not depend on notification")
	cmd := model.Update(tea.ModeReportMsg{})
	require.NotNil(t, cmd)
	assert.Equal(t, common.SelectionChangedMsg{}, cmd())
	assert.Nil(t, model.Update(tea.ModeReportMsg{}))
}

func TestDispatchActionCompletesAfterHighlightNotification(t *testing.T) {
	for _, withCommand := range []bool{false, true} {
		t.Run(map[bool]string{false: "nil command", true: "action command"}[withCommand], func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))
			model := NewUI(ctx)
			first := common.SelectedRevision{ChangeId: "first"}
			scope := &changingSelectionScope{
				scopeOnlyStackedModel: scopeOnlyStackedModel{scope: actions.ScopeChoose},
				next:                  first,
			}
			if withCommand {
				scope.cmd = func() tea.Msg { return selectionActionResultMsg{} }
			}
			model.stacked = scope
			cmd := model.Update(common.DispatchActionMsg{Action: "choose.move_down", CompletionID: "action"})
			assert.Equal(t, first, ctx.Selection().Highlighted)
			require.NotNil(t, cmd)
			msg := cmd()
			// Bubble Tea's sequence message is private. Inspect its command stages;
			// SimulateModel flattens batches and sequences and cannot prove this ordering.
			_, isBatch := msg.(tea.BatchMsg)
			require.False(t, isBatch, "completion must be sequenced after the notification")
			sequence := reflect.ValueOf(msg)
			require.Equal(t, reflect.Slice, sequence.Kind())
			require.Equal(t, 2, sequence.Len())
			beforeCompletion := sequence.Index(0).Interface().(tea.Cmd)()
			if withCommand {
				batch, ok := beforeCompletion.(tea.BatchMsg)
				require.True(t, ok)
				require.Len(t, batch, 2)
				assert.Equal(t, selectionActionResultMsg{}, batch[0]())
				beforeCompletion = batch[1]()
			}
			assert.Equal(t, common.SelectionChangedMsg{Item: first}, beforeCompletion)
			assert.Equal(t, common.ActionCompletedMsg{ID: "action"}, sequence.Index(1).Interface().(tea.Cmd)())
		})
	}
}

func TestUpdateHighlightNotificationRefreshesPreview(t *testing.T) {
	oldCommand := config.Current.Preview.RevisionCommand
	t.Cleanup(func() { config.Current.Preview.RevisionCommand = oldCommand })
	config.Current.Preview.RevisionCommand = []string{"diff", "-r", "$change_id"}
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect([]string{"diff", "-r", "first"}).SetOutput([]byte("updated preview"))
	defer commandRunner.Verify()
	model := NewUI(test.NewTestContext(commandRunner))
	showPreview(t, model, "original preview")
	model.stacked = &changingSelectionScope{snapshot: common.SelectionSnapshot{
		Highlighted: common.SelectedRevision{ChangeId: "first"},
	}}

	cmd := model.Update(tea.ModeReportMsg{})
	require.NotNil(t, cmd)
	assert.Contains(t, renderSplitView(model, 100, 50), "original preview",
		"detecting a change must leave preview refresh to the emitted message")
	test.SimulateModel(model, cmd)
	assert.Contains(t, renderSplitView(model, 100, 50), "updated preview")
}
