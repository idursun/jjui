package workspacepane

import (
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*Model, *test.CommandRunner) {
	t.Helper()
	runner := test.NewTestCommandRunner(t)
	t.Cleanup(runner.Verify)
	ctx := test.NewTestContext(runner)
	ctx.Location, ctx.WorkspaceName = "/repo", "default"
	m := New(ctx)
	m.SetFocused(true)
	m.workspaces = []jj.Workspace{{Name: "default", Current: true}, {Name: "feature"}}
	return m, runner
}

func TestNavigationRequestsActivationWithoutSeparateSelection(t *testing.T) {
	m, runner := setup(t)
	runner.Expect([]string{"workspace", "root", "--name", "feature", "--ignore-working-copy"}).SetOutput([]byte("/feature"))
	cmd, handled := m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	require.True(t, handled)
	require.NotNil(t, cmd)
	assert.Equal(t, ActivateMsg{Name: "feature", Path: "/feature"}, cmd())
	assert.Equal(t, "default", m.context.WorkspaceName, "only the root model applies validated activations")
	// A successful activation becomes the only list position; there is no cursor.
	m.context.WorkspaceName = "feature"
	cmd, handled = m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	assert.True(t, handled)
	assert.Nil(t, cmd, "navigation clamps at the last workspace")
}

func TestMissingRootPromptsForDirectory(t *testing.T) {
	m, runner := setup(t)
	runner.Expect([]string{"workspace", "root", "--name", "feature", "--ignore-working-copy"}).SetError(errors.New("no recorded path"))
	cmd := m.Update(ItemClickedMsg{Name: "feature"})
	require.NotNil(t, cmd)
	assert.IsType(t, common.ShowInputMsg{}, cmd())
	cmd = m.Update(input.SelectedMsg{Value: "../feature"})
	assert.Equal(t, ActivateMsg{Name: "feature", Path: "/feature"}, cmd())
}

func TestRenameUsesActiveWorkspaceAndUpdatesOnlyAfterSuccess(t *testing.T) {
	m, runner := setup(t)
	cmd, handled := m.HandleIntent(intents.WorkspaceRename{})
	require.True(t, handled)
	assert.IsType(t, common.ShowInputMsg{}, cmd())
	runner.Expect([]string{"workspace", "rename", "--", "renamed"})
	cmd = m.Update(input.SelectedMsg{Value: "renamed"})
	assert.Equal(t, "default", m.context.WorkspaceName)
	result := cmd()
	m.Update(result)
	assert.Equal(t, "renamed", m.context.WorkspaceName)
	assert.Equal(t, "renamed", m.launchName)
}

func TestAddPromptsForPathAndNameAndKeepsActiveWorkspace(t *testing.T) {
	m, runner := setup(t)
	cmd, _ := m.HandleIntent(intents.WorkspaceAdd{})
	assert.IsType(t, common.ShowInputMsg{}, cmd())
	cmd = m.Update(input.SelectedMsg{Value: "../feature"})
	assert.Equal(t, common.ShowInputMsg{Title: "Add workspace", Prompt: "Name: ", Value: "feature"}, cmd())
	runner.Expect([]string{"workspace", "add", "--name", "feature", "--", "/feature"})
	cmd = m.Update(input.SelectedMsg{Value: "feature"})
	m.Update(cmd())
	assert.Equal(t, "default", m.context.WorkspaceName)
}

func TestForgetAndRemoveRequireConfirmationAndReturnToLaunchWorkspace(t *testing.T) {
	for _, operation := range []string{"forget", "remove"} {
		t.Run(operation, func(t *testing.T) {
			m, runner := setup(t)
			m.context.Location, m.context.WorkspaceName = "/feature", "feature"
			var intent intents.Intent = intents.WorkspaceForget{}
			if operation == "remove" {
				intent = intents.WorkspaceRemove{}
			}
			cmd, handled := m.HandleIntent(intent)
			require.True(t, handled)
			assert.Nil(t, cmd)
			require.NotNil(t, m.confirmation)
			assert.Contains(t, m.confirmation.View(), "default")
			if operation == "remove" {
				assert.Contains(t, m.confirmation.View(), "ignored and untracked")
			}
			// Enter defaults to cancellation; deletion is never the default choice.
			cmd, _ = m.HandleIntent(intents.Apply{})
			assert.IsType(t, confirmation.CloseMsg{}, cmd())
			m.HandleIntent(intents.OptionSelect{Delta: 1})
			cmd, _ = m.HandleIntent(intents.Apply{})
			runner.Expect([]string{"-R", "/repo", "workspace", operation, "--", "feature"})
			cmd = m.Update(cmd())
			cmd = m.Update(cmd())
			assert.Equal(t, ActivateMsg{Name: "default", Path: "/repo"}, cmd())
		})
	}
}

func TestLaunchWorkspaceIsProtected(t *testing.T) {
	m, _ := setup(t)
	for _, intent := range []intents.Intent{intents.WorkspaceForget{}, intents.WorkspaceRemove{}} {
		cmd, handled := m.HandleIntent(intent)
		require.True(t, handled)
		assert.IsType(t, intents.AddMessage{}, cmd())
		assert.Nil(t, m.confirmation)
	}
}

func TestFailedOperationDoesNotChangeActiveWorkspace(t *testing.T) {
	m, runner := setup(t)
	m.pending = "rename"
	runner.Expect([]string{"workspace", "rename", "--", "duplicate"}).SetError(errors.New("already exists"))
	cmd := m.Update(input.SelectedMsg{Value: "duplicate"})
	cmd = m.Update(cmd())
	assert.IsType(t, intents.AddMessage{}, cmd())
	assert.Equal(t, "default", m.context.WorkspaceName)
	assert.False(t, m.busy)
}

func TestStaleLoadsAndCancelledInputDoNotChangePanel(t *testing.T) {
	m, _ := setup(t)
	m.Update(loadedMsg{location: "/old", workspaces: []jj.Workspace{{Name: "wrong"}}})
	assert.Len(t, m.workspaces, 2)
	m.pending, m.pendingName = "activate", "feature"
	m.Update(input.CancelledMsg{})
	assert.Nil(t, m.Update(input.SelectedMsg{Value: "/wrong"}))
	assert.Equal(t, "default", m.context.WorkspaceName)
}

func TestBusyPanelConsumesNavigation(t *testing.T) {
	m, _ := setup(t)
	m.busy = true
	for _, intent := range []intents.Intent{intents.WorkspaceNavigate{Delta: 1}, intents.Cancel{}, intents.OpenWorkspaces{}} {
		cmd, handled := m.HandleIntent(intent)
		assert.True(t, handled)
		assert.Nil(t, cmd)
	}
	m.busy = false
	cmd, _ := m.HandleIntent(intents.Apply{})
	assert.IsType(t, ReturnToRevisionsMsg{}, cmd())
	m.SetFocused(false)
	_, handled := m.HandleIntent(intents.WorkspaceAdd{})
	assert.False(t, handled)
}
