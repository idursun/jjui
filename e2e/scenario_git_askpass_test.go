//go:build e2e

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

const (
	gitAskpassURL      = "https://example.invalid/repo.git"
	gitAskpassUsername = "Username for '" + gitAskpassURL + "':"
	gitAskpassPassword = "Password for '" + gitAskpassURL + "':"
)

func Test_GitAskpass_PromptsAndSubmitsCredentials(t *testing.T) {
	t.Parallel()
	repo, resultPath := newGitAskpassRepo(t, false)
	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	startGitPush(t, session, ctx)

	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassUsername)
	})
	if err != nil {
		t.Fatalf("username prompt did not render: %v", err)
	}
	if screenContains(screen, "Remotes:") {
		t.Fatalf("Git Operations content remained visible behind askpass prompt:\n%s", formatScreen(screen))
	}
	if err := session.SendText("test-user"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "test-user")
	}); err != nil {
		t.Fatalf("username was not echoed: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassPassword)
	}); err != nil {
		t.Fatalf("password prompt did not render: %v", err)
	}
	secret := "test-secret"
	if err := session.SendText(secret); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassPassword) &&
			screenContains(screen, "***********") &&
			!screenContains(screen, secret)
	}); err != nil {
		t.Fatalf("password was not masked: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		data, err := os.ReadFile(resultPath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return string(data) == "test-user\n"+secret+"\n", err
	})

	quitJJUI(t, session, ctx)
}

func Test_GitAskpass_CancelStopsPrompt(t *testing.T) {
	t.Parallel()
	repo, resultPath := newGitAskpassRepo(t, true)
	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	startGitPush(t, session, ctx)

	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassUsername)
	})
	if err != nil {
		t.Fatalf("username prompt did not render: %v", err)
	}
	if screenContains(screen, "Remotes:") {
		t.Fatalf("Git Operations content remained visible behind askpass prompt:\n%s", formatScreen(screen))
	}
	if err := session.SendText("test-user"); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassPassword)
	}); err != nil {
		t.Fatalf("password prompt did not render: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		data, err := os.ReadFile(resultPath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return string(data) == "cancelled\n", err
	})

	quitJJUI(t, session, ctx)
}

func newGitAskpassRepo(t *testing.T, cancel bool) (*testRepo, string) {
	t.Helper()
	repo := newTestRepo(t)
	resultPath := filepath.Join(filepath.Dir(repo.Path()), "askpass-result")
	fakeGitDir := filepath.Join(filepath.Dir(repo.Path()), "fake-git-bin")
	if err := os.MkdirAll(fakeGitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	script := "#!/bin/sh\nset -eu\n" +
		"username=\"$(\"$GIT_ASKPASS\" \"" + gitAskpassUsername + "\")\"\n"
	if cancel {
		script += "if \"$GIT_ASKPASS\" \"" + gitAskpassPassword + "\"; then\n" +
			"  exit 1\n" +
			"else\n" +
			"  printf '%s\\n' cancelled > \"$JJUI_TEST_ASKPASS_RESULT\"\n" +
			"  exit 1\n" +
			"fi\n"
	} else {
		script += "password=\"$(\"$GIT_ASKPASS\" \"" + gitAskpassPassword + "\")\"\n" +
			"printf '%s\\n%s\\n' \"$username\" \"$password\" > \"$JJUI_TEST_ASKPASS_RESULT\"\n"
	}
	if err := os.WriteFile(filepath.Join(fakeGitDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	repo.env = mergeEnvironment(repo.env, []string{
		"PATH=" + fakeGitDir + ":" + environmentValue(repo.env, "PATH"),
		"JJUI_TEST_ASKPASS_RESULT=" + resultPath,
	})
	repo.JJ("git", "remote", "add", "origin", gitAskpassURL).
		JJ("describe", "-m", "remote main").
		Bookmark("main", "@")
	writeJJUIConfig(t, repo.Env(), "[askpass]\nenabled = true\n")
	return repo, resultPath
}

func startGitPush(t *testing.T, session *ptySession, ctx context.Context) {
	t.Helper()
	if err := session.SendKey(ghostty.KeyG, "g", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "origin")
	}); err != nil {
		t.Fatalf("git operations menu did not show origin: %v", err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "git push")
	}); err != nil {
		t.Fatalf("git push actions did not render: %v", err)
	}
	if err := session.SendKey(ghostty.KeyB, "b", 0); err != nil {
		t.Fatal(err)
	}
}

func quitJJUI(t *testing.T, session *ptySession, ctx context.Context) {
	t.Helper()
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
