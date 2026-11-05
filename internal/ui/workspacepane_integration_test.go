package ui

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
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
		if os.Getenv("JJUI_REQUIRE_JJ") != "" {
			t.Fatal("jj is required for workspace integration tests")
		}
		t.Skip("jj is not installed")
	}
	// jj reports canonical workspace roots, including macOS /var -> /private/var.
	base, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
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
	previousWorkspace := config.Current.Workspace
	config.Current.Workspace.Enabled = true
	config.Current.Revisions.LogBatching = false
	if len(streaming) > 0 {
		config.Current.Revisions.LogBatching = streaming[0]
	}
	t.Cleanup(func() {
		config.Current.Revisions.LogBatching = previous
		config.Current.Workspace = previousWorkspace
	})
	ctx := appContext.NewAppContext(root, askpass.NewUnstartedServer("JJUI_TEST"))
	ctx.DefaultRevset, ctx.CurrentRevset = "all()", "all()"
	m := NewUI(ctx)
	test.SimulateModel(m, m.revisions.Init())
	return m
}

func TestWorkspaceDiscoveryIsOptInAndLazy(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			root, _ := workspaceTestRepo(t)
			realJJ, err := exec.LookPath("jj")
			require.NoError(t, err)
			bin := t.TempDir()
			logPath := filepath.Join(bin, "workspace-queries")
			wrapper := "#!/bin/sh\nif [ \"$1\" = workspace ]; then\n  printf 'workspace\\n' >> \"$JJUI_QUERY_LOG\"\nfi\nexec \"$JJUI_REAL_JJ\" \"$@\"\n"
			require.NoError(t, os.WriteFile(filepath.Join(bin, "jj"), []byte(wrapper), 0700))
			t.Setenv("JJUI_REAL_JJ", realJJ)
			t.Setenv("JJUI_QUERY_LOG", logPath)
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			previous := config.Current.Workspace
			config.Current.Workspace.Enabled = enabled
			t.Cleanup(func() { config.Current.Workspace = previous })
			ctx := appContext.NewAppContext(root, askpass.NewUnstartedServer("JJUI_TEST"))
			ctx.DefaultRevset, ctx.CurrentRevset = "all()", "all()"
			model := NewUI(ctx)
			test.SimulateModel(model, model.revisions.Init())
			require.Empty(t, ctx.WorkspaceName)
			_, err = os.Stat(logPath)
			require.True(t, os.IsNotExist(err), "startup must not discover workspaces")
			test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 'w', Text: "w"}))
			if enabled {
				assert.Equal(t, workspaceContentID, model.splitContainer.ActiveID())
				assert.Equal(t, "default", ctx.WorkspaceName)
				queries, err := os.ReadFile(logPath)
				require.NoError(t, err)
				assert.Equal(t, "workspace\n", string(queries), "one structured list query discovers names and paths")
			} else {
				test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
				assert.NotEqual(t, workspaceContentID, model.splitContainer.ActiveID())
				_, err := os.Stat(logPath)
				assert.True(t, os.IsNotExist(err), "disabled opening must not discover workspaces")
				assert.Empty(t, ctx.WorkspaceName)
			}
		})
	}
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
	// Leave room for full paths even when the test runner uses a long TMPDIR.
	panelWidth := 2 * (max(len(root), len(feature)) + 32)
	panel := renderSplitView(model, panelWidth, 20)
	assert.Contains(t, panel, root)
	assert.Contains(t, panel, feature)
	// Browsing must not change execution context or revision navigation.
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}))
	assert.Equal(t, "default", ctx.WorkspaceName)
	assert.Equal(t, root, ctx.Location)
	assert.Equal(t, selected, model.revisions.SelectedRevision().ChangeId)
	assert.Equal(t, beforeGraph, model.revisions.GetCommitIds())
	key := tea.KeyPressMsg{Code: tea.KeyEnter}
	if streaming {
		key = tea.KeyPressMsg{Code: ' ', Text: " "}
	}
	test.SimulateModel(model, model.Update(key))
	require.Equal(t, "feature", ctx.WorkspaceName)
	assert.Equal(t, feature, ctx.Location)
	assert.Equal(t, selected, model.revisions.SelectedRevision().ChangeId)
	assert.ElementsMatch(t, beforeGraph, model.revisions.GetCommitIds(), "jj may reorder the graph after activation")
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
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
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
	activateWorkspaceFromPanel(t, model, "feature")
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
	activateWorkspaceFromPanel(t, model, "feature")
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

func TestWorkspacePanelRemovesOrForgetsSelectedInactiveWorkspace(t *testing.T) {
	for _, operation := range []string{"remove", "forget"} {
		for _, active := range []string{"default", "other"} {
			t.Run(operation+"/active-"+active, func(t *testing.T) {
				root, run := workspaceTestRepo(t)
				if operation == "remove" {
					if err := exec.Command("jj", "workspace", "remove", "--help").Run(); err != nil {
						t.Skip("installed jj does not support workspace remove")
					}
				}
				target := filepath.Join(filepath.Dir(root), "qwer")
				run(root, "workspace", "add", "--name", "qwer", "--no-colocate", target)
				activePath := root
				if active == "other" {
					activePath = filepath.Join(filepath.Dir(root), "other")
					run(root, "workspace", "add", "--name", "other", "--no-colocate", activePath)
				}
				model := newWorkspaceTestUI(t, root)
				test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
				if active == "other" {
					activateWorkspaceFromPanel(t, model, "other")
				}
				selected := run(activePath, "log", "-r", model.revisions.SelectedRevision().CommitId, "--no-graph", "-T", "change_id")
				test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: "qwer"}))
				require.Equal(t, active, model.context.WorkspaceName)
				key := tea.KeyPressMsg{Code: 'd', Text: "d"}
				if operation == "forget" {
					key = tea.KeyPressMsg{Code: 'f', Text: "f"}
				}
				test.SimulateModel(model, model.Update(key))
				model.width, model.height = 160, 30
				view := ansi.Strip(model.View())
				assert.Contains(t, view, strings.ToUpper(operation[:1])+operation[1:]+" workspace qwer?")
				assert.Contains(t, view, "The active workspace will remain "+active)
				test.SimulateModel(model, model.Update(intents.OptionSelect{Delta: 1}))
				test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}))
				assert.Equal(t, active, model.context.WorkspaceName)
				assert.Equal(t, activePath, model.context.Location)
				assert.DirExists(t, root)
				assert.NotContains(t, run(root, "workspace", "list"), "qwer")
				if operation == "remove" {
					assert.NoDirExists(t, target)
				} else {
					assert.DirExists(t, target)
				}
				assert.Equal(t, selected, run(activePath, "log", "-r", model.revisions.SelectedRevision().CommitId, "--no-graph", "-T", "change_id"))
			})
		}
	}
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
	activateWorkspaceFromPanel(t, model, "feature")
	assert.Equal(t, "feature", model.context.WorkspaceName)
	activateWorkspaceFromPanel(t, model, "default")
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Nil(t, model.dialog(), "returning to the launch workspace must not prompt for its known path")
}

func activateWorkspaceFromPanel(t *testing.T, model *Model, name string) {
	t.Helper()
	previous := model.context.WorkspaceName
	test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: name}))
	require.Equal(t, previous, model.context.WorkspaceName, "clicks must not activate a workspace")
	test.SimulateModel(model, model.Update(intents.Apply{}))
}

func TestWorkspaceRevealSelectsWorkingCopyWithoutActivation(t *testing.T) {
	root, run := workspaceTestRepo(t)
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	model := newWorkspaceTestUI(t, root)
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyDown}))
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 's', Text: "s"}))
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Equal(t, root, model.context.Location)
	assert.True(t, model.revisions.IsFocused())
	assert.Equal(t, run(feature, "log", "-r", "@", "--no-graph", "-T", "change_id"),
		run(root, "log", "-r", model.revisions.SelectedRevision().CommitId, "--no-graph", "-T", "change_id"))
}

func TestWorkspaceUpdateStaleUpdatesBrowsedWorkspaceWithoutActivation(t *testing.T) {
	root, run := workspaceTestRepo(t)
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	run(root, "describe", "-r", "feature@", "-m", "updated working copy")
	model := newWorkspaceTestUI(t, root)
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyDown}))
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: 'u', Text: "u"}))
	assert.Equal(t, "default", model.context.WorkspaceName)
	assert.Equal(t, root, model.context.Location)
	assert.Contains(t, run(feature, "status"), "updated working copy")
}

func TestConfiguredWorkspaceGraphLabelColorFollowsActivation(t *testing.T) {
	root, run := workspaceTestRepo(t)
	run(root, "config", "set", "--repo", `colors."working_copy working_copies"`, `{fg="red",bold=true}`)
	feature := filepath.Join(filepath.Dir(root), "feature")
	run(root, "workspace", "add", "--name", "feature", "--no-colocate", feature)
	model := newWorkspaceTestUI(t, root)
	assertColors := func(active, inactive string) {
		t.Helper()
		ctx := model.context
		out, err := ctx.RunCommandImmediate(jj.Log("all()", 0, ctx.JJConfig.Templates.Log))
		require.NoError(t, err)
		styles := make(map[string]lipgloss.Style)
		for _, row := range parser.ParseRows(bytes.NewReader(out)) {
			for _, line := range row.Lines {
				for _, segment := range line.Segments {
					for _, name := range []string{active, inactive} {
						if strings.Contains(segment.Text, name+"@") {
							styles[name] = segment.Style
						}
					}
				}
			}
		}
		require.Contains(t, styles, active)
		require.Contains(t, styles, inactive)
		assert.Equal(t, lipgloss.Color("1"), styles[active].GetForeground(), "active workspace should be red")
		assert.True(t, styles[active].GetBold())
		assert.Equal(t, lipgloss.Color("2"), styles[inactive].GetForeground(), "other workspaces should retain jj's green")
	}
	assertColors("default", "feature")
	test.SimulateModel(model, model.Update(intents.OpenWorkspaces{}))
	test.SimulateModel(model, model.Update(workspacepane.ItemClickedMsg{Name: "feature"}))
	assertColors("default", "feature")
	test.SimulateModel(model, model.Update(intents.Apply{}))
	assertColors("feature", "default")
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
