package main

import (
	"path/filepath"
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func newRepoWithSideWorkspace(t *testing.T) *testRepo {
	t.Helper()
	repo := newTestRepo(t)
	repo.JJ("describe", "-m", "main work")
	repo.JJ("workspace", "add", "--name", "side", "-r", "@-", filepath.Join(t.TempDir(), "side"))
	return repo
}

func workspaceNames(t *testing.T, repo *testRepo) string {
	return runCommand(t, repo.Path(), repo.Env(), "jj", "workspace", "list", "--ignore-working-copy", "--color", "never", "-T", `name ++ "\n"`)
}

func Test_WorkspaceObjects_ForgetsFocusedWorkspace(t *testing.T) {
	t.Parallel()
	repo := newRepoWithSideWorkspace(t)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "side@")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	focusTextOnRow(t, session, ctx, "side@")
	if err := session.SendKey(ghostty.KeyX, "x", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return !strings.Contains(workspaceNames(t, repo), "side"), nil
	})
}

func Test_WorkspaceObjects_SwitchesToFocusedWorkspace(t *testing.T) {
	t.Parallel()
	repo := newRepoWithSideWorkspace(t)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "side@")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	focusTextOnRow(t, session, ctx, "side@")
	if err := session.SendKey(ghostty.KeyS, "s", 0); err != nil {
		t.Fatal(err)
	}
	// After the switch the side workspace's commit is the working copy.
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		for _, line := range screen {
			if strings.Contains(line, "side@") && strings.HasPrefix(strings.TrimSpace(line), "@") {
				return true
			}
		}
		return false
	}); err != nil {
		t.Fatalf("the log did not switch to the side workspace: %v", err)
	}
}

func Test_WorkspaceObjects_RenamesCurrentWorkspace(t *testing.T) {
	t.Parallel()
	repo := newRepoWithSideWorkspace(t)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "default@")
	focusTextOnRow(t, session, ctx, "default@")
	if err := session.SendKey(ghostty.KeyR, "r", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText("2"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "default2@") }); err != nil {
		t.Fatalf("the rename did not edit the label in place: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return strings.Contains(workspaceNames(t, repo), "default2"), nil
	})
}
