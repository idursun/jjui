//go:build e2e

package main

import (
	"os"
	"path/filepath"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

const sshAskpassPrompt = "Enter passphrase for key '/tmp/jjui-test-key':"

func Test_SSHAskpass_PromptsAndSubmitsPassphrase(t *testing.T) {
	t.Parallel()
	repo, resultPath := newSSHAskpassRepo(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	startGitPush(t, session, ctx)

	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, sshAskpassPrompt)
	}); err != nil {
		t.Fatalf("SSH passphrase prompt did not render: %v", err)
	}

	secret := "test-passphrase"
	if err := session.SendText(secret); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, sshAskpassPrompt) &&
			screenContains(screen, "***************") &&
			!screenContains(screen, secret)
	}); err != nil {
		t.Fatalf("SSH passphrase was not masked: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		data, err := os.ReadFile(resultPath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return string(data) == secret+"\n", err
	})

	quitJJUI(t, session, ctx)
}

func newSSHAskpassRepo(t *testing.T) (*testRepo, string) {
	t.Helper()
	repo := newTestRepo(t)
	resultPath := filepath.Join(filepath.Dir(repo.Path()), "ssh-askpass-result")
	fakeSSHDir := filepath.Join(filepath.Dir(repo.Path()), "fake-ssh-bin")
	if err := os.MkdirAll(fakeSSHDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fakeGit := "#!/bin/sh\nset -eu\nexec ssh \"$@\"\n"
	if err := os.WriteFile(filepath.Join(fakeSSHDir, "git"), []byte(fakeGit), 0o755); err != nil {
		t.Fatal(err)
	}
	fakeSSH := "#!/bin/sh\nset -eu\n" +
		"passphrase=\"$(\"$SSH_ASKPASS\" \"" + sshAskpassPrompt + "\")\"\n" +
		"printf '%s\\n' \"$passphrase\" > \"$JJUI_TEST_SSH_ASKPASS_RESULT\"\n"
	if err := os.WriteFile(filepath.Join(fakeSSHDir, "ssh"), []byte(fakeSSH), 0o755); err != nil {
		t.Fatal(err)
	}

	repo.env = mergeEnvironment(repo.env, []string{
		"PATH=" + fakeSSHDir + ":" + environmentValue(repo.env, "PATH"),
		"JJUI_TEST_SSH_ASKPASS_RESULT=" + resultPath,
	})
	repo.JJ("git", "remote", "add", "origin", "git@github.com:example/repo.git").
		JJ("describe", "-m", "ssh remote").
		Bookmark("main", "@")
	return repo, resultPath
}
