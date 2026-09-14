//go:build e2e

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

const (
	gitAskpassUsernamePrefix = "Username for 'http://127.0.0.1:"
	gitAskpassPasswordPrefix = "Password for '"
)

func Test_GitAskpass_PromptsAndSubmitsCredentials(t *testing.T) {
	t.Parallel()
	repo, credentialsPath := newGitAskpassRepo(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	startGitPush(t, session, ctx)

	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassUsernamePrefix)
	})
	if err != nil {
		t.Fatalf("username prompt did not render: %v", err)
	}
	if screenContains(screen, "Remotes:") || screenContains(screen, "git push") {
		t.Fatalf("background Git content remained visible behind askpass prompt:\n%s", formatScreen(screen))
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
		return screenContains(screen, gitAskpassPasswordPrefix)
	}); err != nil {
		t.Fatalf("password prompt did not render: %v", err)
	}
	secret := "test-secret"
	if err := session.SendText(secret); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassPasswordPrefix) &&
			screenContains(screen, "***********") &&
			!screenContains(screen, secret)
	}); err != nil {
		t.Fatalf("password was not masked: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		data, err := os.ReadFile(credentialsPath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return string(data) == "test-user\n"+secret+"\n", err
	})

	quitJJUI(t, session, ctx)
}

func Test_GitAskpass_CancelStopsPrompt(t *testing.T) {
	t.Parallel()
	repo, credentialsPath := newGitAskpassRepo(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	startGitPush(t, session, ctx)

	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassUsernamePrefix)
	})
	if err != nil {
		t.Fatalf("username prompt did not render: %v", err)
	}
	if screenContains(screen, "Remotes:") || screenContains(screen, "git push") {
		t.Fatalf("background Git content remained visible behind askpass prompt:\n%s", formatScreen(screen))
	}
	if err := session.SendText("test-user"); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, gitAskpassPasswordPrefix)
	}); err != nil {
		t.Fatalf("password prompt did not render: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "Git process failed")
	}); err != nil {
		t.Fatalf("Git command did not terminate after canceling the prompt: %v", err)
	}
	if _, err := os.Stat(credentialsPath); !os.IsNotExist(err) {
		t.Fatalf("canceled Git command sent credentials: %v", err)
	}

	quitJJUI(t, session, ctx)
}

func newGitAskpassRepo(t *testing.T) (*testRepo, string) {
	t.Helper()
	repo := newTestRepo(t)
	credentialsPath := filepath.Join(filepath.Dir(repo.Path()), "askpass-credentials")
	gitURL, closeServer := newGitCredentialServer(t, credentialsPath)
	t.Cleanup(closeServer)
	fakeGitDir := filepath.Join(filepath.Dir(repo.Path()), "fake-git-bin")
	if err := os.MkdirAll(fakeGitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nset -u\nexec \"$JJUI_REAL_GIT\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(fakeGitDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	repo.env = mergeEnvironment(repo.env, []string{
		"PATH=" + fakeGitDir + ":" + environmentValue(repo.env, "PATH"),
		"JJUI_REAL_GIT=" + realGit,
	})
	repo.JJ("git", "remote", "add", "origin", gitURL).
		JJ("describe", "-m", "remote main").
		Bookmark("main", "@")
	return repo, credentialsPath
}

func newGitCredentialServer(t *testing.T, resultPath string) (string, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="jjui-e2e"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := os.WriteFile(resultPath, []byte(fmt.Sprintf("%s\n%s\n", username, password)), 0o600); err != nil {
			t.Logf("recording Git credentials: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	return server.URL, server.Close
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
