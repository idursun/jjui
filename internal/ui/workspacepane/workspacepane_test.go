package workspacepane

import (
	"errors"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
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

func TestNavigationOnlyBrowsesAndApplyExplicitlyActivates(t *testing.T) {
	m, _ := setup(t)
	cmd, handled := m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	require.True(t, handled)
	assert.Nil(t, cmd)
	assert.Equal(t, "feature", m.selectedName)
	assert.Equal(t, "default", m.context.WorkspaceName)
	m.paths["feature"] = "/feature"
	cmd, handled = m.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	assert.Equal(t, ActivateMsg{Name: "feature", Path: "/feature"}, cmd())
	assert.Equal(t, "default", m.context.WorkspaceName, "only the root model applies validated activations")
	m.context.WorkspaceName = "feature"
	cmd, handled = m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	assert.True(t, handled)
	assert.Nil(t, cmd, "navigation clamps at the last workspace")
	assert.Nil(t, m.Update(ItemClickedMsg{Name: "default"}))
	assert.Equal(t, "default", m.selectedName)
	assert.Equal(t, "feature", m.context.WorkspaceName, "a click only browses")
}

func TestMissingRootPromptsForDirectory(t *testing.T) {
	m, _ := setup(t)
	assert.Nil(t, m.Update(ItemClickedMsg{Name: "feature"}))
	cmd, _ := m.HandleIntent(intents.Apply{})
	require.NotNil(t, cmd)
	assert.IsType(t, common.ShowInputMsg{}, cmd())
	cmd = m.Update(input.SelectedMsg{Value: "../feature"})
	assert.Equal(t, ActivateMsg{Name: "feature", Path: "/feature"}, cmd())
}

func TestKnownSessionPathAvoidsDirectoryPrompt(t *testing.T) {
	m, _ := setup(t)
	m.context.Location, m.context.WorkspaceName = "/feature", "feature"
	m.Update(ItemClickedMsg{Name: "default"})
	cmd, handled := m.HandleIntent(intents.Apply{})
	require.True(t, handled)
	assert.Equal(t, ActivateMsg{Name: "default", Path: "/repo"}, cmd())
}

func TestOlderJJLoadsNamesAndPromptsForUnknownPaths(t *testing.T) {
	m, runner := setup(t)
	runner.Expect(jj.WorkspaceListWithPaths()).SetError(errors.New("unknown template keyword: root"))
	runner.Expect(jj.WorkspaceList()).SetOutput([]byte("default\ttrue\nfeature\tfalse\n"))
	m.Update(m.OnShow()())
	assert.Equal(t, "/repo", m.paths["default"])
	assert.Empty(t, m.paths["feature"])
	m.Update(ItemClickedMsg{Name: "feature"})
	cmd, handled := m.HandleIntent(intents.Apply{})
	require.True(t, handled)
	assert.IsType(t, common.ShowInputMsg{}, cmd())
}

func TestRecordedPathsIdentifyCurrentWorkspaceWithoutRevisionHint(t *testing.T) {
	m, runner := setup(t)
	m.context.WorkspaceName = ""
	runner.Expect(jj.WorkspaceListWithPaths()).SetOutput([]byte("\"default\"\ttrue\t\"/repo\"\n\"feature\"\ttrue\t\"/feature\"\n"))
	m.Update(m.OnShow()())
	assert.Equal(t, "default", m.context.WorkspaceName)
	assert.Equal(t, "default", m.selectedName)
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
			m.selectedName = "feature"
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
	// Selecting the launch workspace must remain protected even when another
	// workspace is active; protection belongs to the target, not the context.
	m.context.Location, m.context.WorkspaceName = "/feature", "feature"
	for _, intent := range []intents.Intent{intents.WorkspaceForget{}, intents.WorkspaceRemove{}} {
		cmd, handled := m.HandleIntent(intent)
		require.True(t, handled)
		assert.IsType(t, intents.AddMessage{}, cmd())
		assert.Nil(t, m.confirmation)
	}
}

func TestRemoveAndForgetTargetBrowsedWorkspaceWhileLaunchRemainsActive(t *testing.T) {
	for _, operation := range []string{"remove", "forget"} {
		t.Run(operation, func(t *testing.T) {
			m, runner := setup(t)
			m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
			var intent intents.Intent = intents.WorkspaceRemove{}
			if operation == "forget" {
				intent = intents.WorkspaceForget{}
			}
			cmd, handled := m.HandleIntent(intent)
			require.True(t, handled)
			require.Nil(t, cmd, "the selected non-launch workspace should reach confirmation")
			require.NotNil(t, m.confirmation)
			assert.Contains(t, m.confirmation.View(), strings.ToUpper(operation[:1])+operation[1:]+" workspace feature")
			assert.Contains(t, m.confirmation.View(), "will remain default")
			runner.Expect([]string{"-R", "/repo", "workspace", operation, "--", "feature"})
			m.HandleIntent(intents.OptionSelect{Delta: 1})
			cmd, _ = m.HandleIntent(intents.Apply{})
			cmd = m.Update(cmd())
			cmd = m.Update(cmd())
			assert.IsType(t, common.RefreshMsg{}, cmd(), "removing an inactive workspace must not activate another workspace")
			assert.Equal(t, "default", m.context.WorkspaceName)
			assert.Equal(t, "/repo", m.context.Location)
		})
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
	assert.Nil(t, cmd, "applying the already active workspace is a no-op")
	m.SetFocused(false)
	_, handled := m.HandleIntent(intents.WorkspaceAdd{})
	assert.False(t, handled)
}

func TestRevealWorkingCopyDoesNotActivateWorkspace(t *testing.T) {
	m, runner := setup(t)
	m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	runner.Expect(jj.ResolveRevisionID(`"feature"@`)).SetOutput([]byte("change;commit"))
	cmd, handled := m.HandleIntent(intents.WorkspaceReveal{})
	require.True(t, handled)
	assert.Equal(t, RevealRevisionMsg{Name: "feature", CommitID: "commit"}, cmd())
	assert.Equal(t, "default", m.context.WorkspaceName)
}

func TestUpdateStaleTargetsBrowsedWorkspaceWithoutActivation(t *testing.T) {
	m, runner := setup(t)
	m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	m.paths["feature"] = "/feature"
	runner.Expect([]string{"-R", "/feature", "workspace", "update-stale"})
	cmd, handled := m.HandleIntent(intents.WorkspaceUpdateStale{})
	require.True(t, handled)
	m.Update(cmd())
	assert.Equal(t, "feature", m.selectedName)
	assert.Equal(t, "default", m.context.WorkspaceName)
	assert.Equal(t, "/repo", m.context.Location)
}

func TestRefreshPreservesBrowsedWorkspace(t *testing.T) {
	m, _ := setup(t)
	m.HandleIntent(intents.WorkspaceNavigate{Delta: 1})
	m.Update(loadedMsg{location: "/repo", workspaces: []jj.Workspace{{Name: "default", Current: true}, {Name: "feature"}}})
	assert.Equal(t, "feature", m.selectedName)
	assert.Equal(t, "default", m.context.WorkspaceName)
}

func TestWorkspacePathsAreLoadedAndRenderedBesideNames(t *testing.T) {
	m, runner := setup(t)
	runner.Expect(jj.WorkspaceListWithPaths()).SetOutput([]byte("\"default\"\ttrue\tnull\n\"feature\"\tfalse\t\"/feature\"\n\"missing\"\tfalse\tnull\n"))
	m.Update(m.OnShow()())
	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 80, 8)))
	lines := strings.Split(dl.RenderToString(80, 8), "\n")
	assert.Contains(t, lines[2], "default")
	assert.Contains(t, lines[2], "/repo")
	assert.Contains(t, lines[3], "feature")
	assert.Contains(t, lines[3], "/feature")
	assert.Contains(t, lines[4], "missing")
	assert.Contains(t, lines[4], "(path unavailable)")
	assert.Equal(t, "default", m.context.WorkspaceName)
}

func TestLongWorkspacePathsStayOnOneClippedRow(t *testing.T) {
	m, _ := setup(t)
	m.paths["feature"] = "/" + strings.Repeat("long-directory/", 20)
	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 30, 6)))
	lines := strings.Split(dl.RenderToString(30, 6), "\n")
	assert.Contains(t, lines[3], "feature")
	assert.NotContains(t, lines[4], "long-directory")
	for _, line := range lines {
		assert.LessOrEqual(t, render.StringWidth(line), 30)
	}
}
