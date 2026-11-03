package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/workspacepane"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workspaceTestRepo(t *testing.T) (string, func(string, ...string) string) {
	t.Helper()
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj is not installed")
	}
	base := t.TempDir()
	configPath := filepath.Join(base, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[user]\nname = 'Workspace Test'\nemail = 'workspace@example.test'\n"), 0600))
	t.Setenv("JJ_CONFIG", configPath)
	t.Setenv("JJ_CONFIG_TOML", "")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("jj", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%v: %s", args, out)
		return strings.TrimSpace(string(out))
	}
	root := filepath.Join(base, "main")
	run(base, "git", "init", "--no-colocate", root)
	return root, run
}

func newWorkspaceTestUI(t *testing.T, root string, streaming ...bool) *Model {
	t.Helper()
	previous := config.Current.Revisions.LogBatching
	config.Current.Revisions.LogBatching = false
	if len(streaming) > 0 {
		config.Current.Revisions.LogBatching = streaming[0]
	}
	t.Cleanup(func() { config.Current.Revisions.LogBatching = previous })
	ctx := appContext.NewAppContext(root, askpass.NewUnstartedServer("JJUI_TEST"))
	ctx.DefaultRevset, ctx.CurrentRevset = "all()", "all()"
	m := NewUI(ctx)
	test.SimulateModel(m, m.revisions.Init())
	return m
}

func TestWorkspaceSwitchPreservesRevisionAndNewRunsInActiveWorkspace(t *testing.T) {
	t.Run("batch", func(t *testing.T) { testWorkspaceSwitchAndNew(t, false) })
	t.Run("streaming", func(t *testing.T) { testWorkspaceSwitchAndNew(t, true) })
}

func testWorkspaceSwitchAndNew(t *testing.T, streaming bool) {
	t.Helper()
	root, run := workspaceTestRepo(t)
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	model := newWorkspaceTestUI(t, root, streaming)
	ctx := model.context
	launchID := run(root, "log", "-r", "@", "--no-graph", "-T", "change_id")
	test.SimulateModel(model, model.Update(intents.Navigate{ChangeID: launchID}))
	require.NotNil(t, model.revisions.SelectedRevision())
	selected := model.revisions.SelectedRevision().ChangeId
	beforeGraph := model.revisions.GetCommitIds()
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 'w', Text: "w"}))
	require.Equal(t, workspaceContentID, model.splitContainer.ActiveID())
	require.True(t, model.splitContainer.ContentFocused())
	// No separate workspace cursor: moving to the next row activates it.
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}))
	require.Equal(t, "feature", ctx.WorkspaceName)
	assert.Equal(t, feature, ctx.Location)
	assert.Equal(t, selected, model.revisions.SelectedRevision().ChangeId)
	assert.Equal(t, beforeGraph, model.revisions.GetCommitIds())
	assert.Equal(t, "all()", ctx.CurrentRevset)
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyTab}))
	require.True(t, model.revisions.IsFocused())
	model.width, model.height = 120, 25
	assert.Contains(t, ansi.Strip(model.View()), "workspace: feature")
	featureBefore := run(feature, "log", "-r", "@", "--no-graph", "-T", "change_id")
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 'n', Text: "n"}))
	assert.Equal(t, launchID, run(root, "log", "-r", "@", "--no-graph", "-T", "change_id"), "launch working copy must not move")
	assert.NotEqual(t, featureBefore, run(feature, "log", "-r", "@", "--no-graph", "-T", "change_id"))
	assert.Equal(t, launchID, run(feature, "log", "-r", "@-", "--no-graph", "-T", "change_id"))
}

func TestWorkspaceActivationFailurePreservesContextAndSelection(t *testing.T) {
	root, _ := workspaceTestRepo(t)
	model := newWorkspaceTestUI(t, root)
	selected := model.revisions.SelectedRevision().ChangeId
	oldRunner := model.context.CommandRunner
	test.SimulateModel(model, model.Update(workspacepane.ActivateMsg{Name: "missing", Path: filepath.Join(root, "missing")}))
	assert.Equal(t, root, model.context.Location)
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Same(t, oldRunner, model.context.CommandRunner)
	assert.Equal(t, selected, model.revisions.SelectedRevision().ChangeId)
}

func TestWorkspaceActivationRejectsUnrelatedRepository(t *testing.T) {
	root, run := workspaceTestRepo(t)
	other := filepath.Join(filepath.Dir(root), "unrelated")
	run(filepath.Dir(root), "git", "init", "--no-colocate", other)
	model := newWorkspaceTestUI(t, root)
	err := model.context.ActivateWorkspace("default", other)
	require.ErrorContains(t, err, "different repository")
	assert.Equal(t, root, model.context.Location)
}

func TestWorkspacePanelRenamesAndForgetsActiveWorkspace(t *testing.T) {
	root, run := workspaceTestRepo(t)
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	model := newWorkspaceTestUI(t, root)
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
	test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: "feature"}))
	require.Equal(t, "feature", model.context.WorkspaceName)
	test.SimulateModel(model, model.Update(intents.WorkspaceRename{}))
	test.SimulateModel(model, model.Update(input.SelectedMsg{Value: "renamed"}))
	require.Equal(t, "renamed", model.context.WorkspaceName)
	assert.Contains(t, run(root, "workspace", "list"), "renamed")
	test.SimulateModel(model, model.Update(intents.WorkspaceForget{}))
	test.SimulateModel(model, model.Update(intents.OptionSelect{Delta: 1}))
	test.SimulateModel(model, model.Update(intents.Apply{}))
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Equal(t, root, model.context.Location)
	assert.DirExists(t, feature)
	assert.NotContains(t, run(root, "workspace", "list"), "renamed")
	assert.NotNil(t, model.revisions.SelectedRevision())
	assert.Equal(t, common.SelectedRevision{ChangeId: model.revisions.SelectedRevision().ChangeId, CommitId: model.revisions.SelectedRevision().CommitId}, model.context.Selection().Highlighted)
}

func TestWorkspacePanelRemovesActiveWorkspaceAndReturnsToLaunch(t *testing.T) {
	root, run := workspaceTestRepo(t)
	if err := exec.Command("jj", "workspace", "remove", "--help").Run(); err != nil {
		t.Skip("installed jj does not support workspace remove")
	}
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	model := newWorkspaceTestUI(t, root)
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
	test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: "feature"}))
	require.Equal(t, "feature", model.context.WorkspaceName)
	selected := run(feature, "log", "-r", model.revisions.SelectedRevision().CommitId, "--no-graph", "-T", "change_id")
	test.SimulateModel(model, model.Update(intents.WorkspaceRemove{}))
	test.SimulateModel(model, model.Update(intents.OptionSelect{Delta: 1}))
	test.SimulateModel(model, model.Update(intents.Apply{}))
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Equal(t, root, model.context.Location)
	assert.NoDirExists(t, feature)
	assert.Equal(t, selected, run(root, "log", "-r", model.revisions.SelectedRevision().CommitId, "--no-graph", "-T", "change_id"))
}

func TestWorkspacePanelAddsWorkspaceWithoutActivatingIt(t *testing.T) {
	root, run := workspaceTestRepo(t)
	model := newWorkspaceTestUI(t, root)
	feature := filepath.Join(filepath.Dir(root), "feature")
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
	test.SimulateModel(model, model.Update(intents.WorkspaceAdd{}))
	test.SimulateModel(model, model.Update(input.SelectedMsg{Value: feature}))
	test.SimulateModel(model, model.Update(input.SelectedMsg{Value: "feature"}))
	assert.DirExists(t, feature)
	assert.Contains(t, run(root, "workspace", "list"), "feature")
	assert.Equal(t, "default", model.context.WorkspaceName)
	test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: "feature"}))
	assert.Equal(t, "feature", model.context.WorkspaceName)
	test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: "default"}))
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Nil(t, model.dialog(), "returning to the launch workspace must not prompt for its known path")
}

func TestQueuedRevisionCommandStaysInOriginalWorkspace(t *testing.T) {
	root, run := workspaceTestRepo(t)
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	model := newWorkspaceTestUI(t, root)
	selected := model.revisions.SelectedRevision().ChangeId
	featureBefore := run(feature, "log", "-r", "@", "--no-graph", "-T", "change_id")
	// Create the command, but switch workspace before its subprocess starts.
	queued := model.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	require.NotNil(t, queued)
	test.SimulateModel(model, model.Update(workspacepane.ActivateMsg{Name: "feature", Path: feature}))
	require.Equal(t, "feature", model.context.WorkspaceName)
	test.SimulateModel(model, queued)
	assert.Equal(t, featureBefore, run(feature, "log", "-r", "@", "--no-graph", "-T", "change_id"))
	assert.Equal(t, selected, model.revisions.SelectedRevision().ChangeId, "the old command must not jump selection to the new workspace's @")
}
