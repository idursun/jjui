package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ghostty "go.mitchellh.com/libghostty"
)

func startJJUITest(t *testing.T, want string) (*testRepo, *ptySession, context.Context) {
	t.Helper()

	jjuiBin := jjuiBinary(t)
	repo := newTestRepo(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, want)
	return repo, session, ctx
}

func startJJUITestWithRepo(t *testing.T, jjuiBin string, repo *testRepo, want string) (*ptySession, context.Context) {
	t.Helper()
	session, ctx := startJJUIProcessAt(t, jjuiBin, repo.Path(), repo.Env())
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, want)
	}); err != nil {
		t.Fatalf("jjui did not render %q: %v", want, err)
	}

	return session, ctx
}

func startJJUIProcessAt(t *testing.T, jjuiBin, repo string, env []string) (*ptySession, context.Context) {
	t.Helper()

	term, err := ghostty.NewTerminal(ghostty.WithSize(100, 30))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Close() })

	session, err := startPTY(term, []string{jjuiBin, repo}, repo, env, 100, 30)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	t.Cleanup(func() {
		if t.Failed() {
			logFailureDiagnostics(t, session, repo, env)
		}
	})
	return session, ctx
}

func jjuiBinary(t *testing.T) string {
	t.Helper()
	jjuiBin := os.Getenv("JJUI_BIN")
	if jjuiBin == "" {
		t.Fatal("JJUI_BIN is not set; run the E2E suite with docker-compose or provide JJUI_BIN")
	}
	return jjuiBin
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()

	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	configHome := filepath.Join(root, "config")
	jjuiConfig := filepath.Join(root, "jjui-config")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(jjuiConfig, 0o755); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"HOME=" + filepath.Join(root, "home"),
		"XDG_CONFIG_HOME=" + configHome,
		"JJUI_CONFIG_DIR=" + jjuiConfig,
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	}

	runCommand(t, root, env, "jj", "config", "set", "--user", "user.name", "PTY Test")
	runCommand(t, root, env, "jj", "config", "set", "--user", "user.email", "pty@example.com")
	runCommand(t, root, env, "jj", "git", "init", repo)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("created by the PTY test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCommand(t, repo, env, "jj", "commit", "-m", "initial")

	return &testRepo{t: t, path: repo, env: env}
}

// testRepo is a fluent builder for the common repository setup used by PTY
// scenarios. It keeps repository paths and command environments together so
// history-building steps can be chained without repeating harness plumbing.
type testRepo struct {
	t    *testing.T
	path string
	env  []string
}

func (r *testRepo) Append(path, content string) *testRepo {
	r.t.Helper()
	appendToFile(r.t, filepath.Join(r.path, path), content)
	return r
}

func (r *testRepo) Write(path, content string) *testRepo {
	r.t.Helper()
	target := filepath.Join(r.path, path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
	return r
}

func (r *testRepo) Commit(message string) *testRepo {
	r.t.Helper()
	r.JJ("commit", "-m", message)
	return r
}

func (r *testRepo) New(revsets ...string) *testRepo {
	r.t.Helper()
	r.JJ(append([]string{"new"}, revsets...)...)
	return r
}

func (r *testRepo) Bookmark(name, revset string) *testRepo {
	r.t.Helper()
	r.JJ("bookmark", "create", name, "-r", revset)
	return r
}

func (r *testRepo) JJ(args ...string) *testRepo {
	r.t.Helper()
	runCommand(r.t, r.path, r.env, "jj", args...)
	return r
}

func (r *testRepo) Path() string {
	return r.path
}

func (r *testRepo) Env() []string {
	return append([]string(nil), r.env...)
}

func writeJJUIConfig(t *testing.T, env []string, content string) {
	t.Helper()
	writeJJUIFile(t, env, "config.toml", content)
}

func writeJJUILuaConfig(t *testing.T, env []string, content string) {
	t.Helper()
	writeJJUIFile(t, env, "config.lua", content)
}

func writeGlobalJJUIConfig(t *testing.T, env []string, content string) {
	t.Helper()
	writeGlobalJJUIFile(t, env, "config.toml", content)
}

func writeGlobalJJUILuaConfig(t *testing.T, env []string, content string) {
	t.Helper()
	writeGlobalJJUIFile(t, env, "config.lua", content)
}

func writeJJUIPlugin(t *testing.T, env []string, name, content string) {
	t.Helper()
	configDir := jjuiConfigDir(t, env)
	pluginsDir := filepath.Join(configDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, name+".lua"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeRepoJJUIConfig(t *testing.T, repo, content string) {
	t.Helper()
	writeRepoJJUIFile(t, repo, "config.toml", content)
}

func writeRepoJJUILuaConfig(t *testing.T, repo, content string) {
	t.Helper()
	writeRepoJJUIFile(t, repo, "config.lua", content)
}

func writeJJUIFile(t *testing.T, env []string, name, content string) {
	t.Helper()
	configDir := jjuiConfigDir(t, env)
	if err := os.WriteFile(filepath.Join(configDir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeRepoJJUIFile(t *testing.T, repo, name, content string) {
	t.Helper()
	configDir := filepath.Join(repo, ".jjui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeGlobalJJUIFile(t *testing.T, env []string, name, content string) {
	t.Helper()
	configHome := environmentValue(env, "XDG_CONFIG_HOME")
	if configHome == "" {
		home := environmentValue(env, "HOME")
		if home == "" {
			t.Fatal("HOME is not configured")
		}
		configHome = filepath.Join(home, ".config")
	}
	configDir := filepath.Join(configHome, "jjui")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func jjuiConfigDir(t *testing.T, env []string) string {
	t.Helper()
	configDir := environmentValue(env, "JJUI_CONFIG_DIR")
	if configDir == "" {
		t.Fatal("JJUI_CONFIG_DIR is not configured")
	}
	return configDir
}

func withoutJJUIConfigDir(env []string) []string {
	result := make([]string, 0, len(env))
	for _, value := range env {
		if strings.HasPrefix(value, "JJUI_CONFIG_DIR=") {
			continue
		}
		result = append(result, value)
	}
	return result
}

func writeJJUITheme(t *testing.T, env []string, name, content string) {
	t.Helper()
	configDir := environmentValue(env, "JJUI_CONFIG_DIR")
	if configDir == "" {
		t.Fatal("JJUI_CONFIG_DIR is not configured")
	}
	themesDir := filepath.Join(configDir, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, name+".toml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func appendToFile(t *testing.T, path, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func runCommand(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	output, err := commandOutput(dir, env, name, args...)
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return output
}

func commandOutput(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = mergeEnvironment(os.Environ(), env)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func environmentValue(env []string, name string) string {
	prefix := name + "="
	for _, value := range env {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return ""
}

// Harness is a small convenience layer for scenarios that exercise Lua
// configuration. The lower-level session remains available for tests that
// need exact terminal events or styled-screen assertions.
type Harness struct {
	t       *testing.T
	repo    *testRepo
	bin     string
	session *ptySession
	ctx     context.Context
}

func NewHarness(t *testing.T) *Harness {
	t.Helper()
	repo := newTestRepo(t)
	return &Harness{t: t, repo: repo, bin: jjuiBinary(t)}
}

func (h *Harness) Repo() string { return h.repo.Path() }

func (h *Harness) Env() []string { return h.repo.Env() }

func (h *Harness) WriteConfigTOML(content string) {
	h.t.Helper()
	writeJJUIConfig(h.t, h.repo.Env(), content)
}

func (h *Harness) WriteConfigLua(content string) {
	h.t.Helper()
	writeJJUILuaConfig(h.t, h.repo.Env(), content)
}

func (h *Harness) WriteGlobalConfigTOML(content string) {
	h.t.Helper()
	writeGlobalJJUIConfig(h.t, h.repo.Env(), content)
}

func (h *Harness) WriteGlobalConfigLua(content string) {
	h.t.Helper()
	writeGlobalJJUILuaConfig(h.t, h.repo.Env(), content)
}

func (h *Harness) WritePlugin(name, content string) {
	h.t.Helper()
	writeJJUIPlugin(h.t, h.repo.Env(), name, content)
}

func (h *Harness) WriteRepoConfigTOML(content string) {
	h.t.Helper()
	writeRepoJJUIConfig(h.t, h.repo.Path(), content)
}

func (h *Harness) WriteRepoConfigLua(content string) {
	h.t.Helper()
	writeRepoJJUILuaConfig(h.t, h.repo.Path(), content)
}

func (h *Harness) UseRepositoryConfig() {
	h.t.Helper()
	h.repo.env = withoutJJUIConfigDir(h.repo.env)
}

func (h *Harness) Start(want string) {
	h.t.Helper()
	h.StartProcess()
	h.WaitText(want)
}

func (h *Harness) StartProcess() {
	h.t.Helper()
	h.session, h.ctx = startJJUIProcessAt(h.t, h.bin, h.repo.Path(), h.repo.Env())
}

func (h *Harness) Key(name string) {
	h.t.Helper()
	key, text, mods, err := harnessKey(name)
	if err != nil {
		h.t.Fatal(err)
	}
	if err := h.session.SendKey(key, text, mods); err != nil {
		h.t.Fatalf("send key %q: %v", name, err)
	}
}

func (h *Harness) Text(value string) {
	h.t.Helper()
	if err := h.session.SendText(value); err != nil {
		h.t.Fatalf("send text: %v", err)
	}
}

func (h *Harness) WaitText(want string) []string {
	h.t.Helper()
	screen, err := h.session.WaitForScreen(h.ctx, func(screen []string) bool {
		return screenContains(screen, want)
	})
	if err != nil {
		h.t.Fatalf("waiting for %q: %v", want, err)
	}
	h.session.addTrace(fmt.Sprintf("matched text %q", want))
	return screen
}

func (h *Harness) WaitNoText(want string) []string {
	h.t.Helper()
	screen, err := h.session.WaitForScreen(h.ctx, func(screen []string) bool {
		return !screenContains(screen, want)
	})
	if err != nil {
		h.t.Fatalf("waiting for %q to disappear: %v", want, err)
	}
	h.session.addTrace(fmt.Sprintf("matched absence of %q", want))
	return screen
}

func (h *Harness) WaitRowContaining(want string) (int, string) {
	h.t.Helper()
	screen := h.WaitText(want)
	row := screenRowIndexContaining(screen, want)
	if row < 0 {
		h.t.Fatalf("text %q was not found in the captured screen", want)
	}
	return row, screen[row]
}

func (h *Harness) WaitTextInOrder(wants ...string) []string {
	h.t.Helper()
	screen, err := h.session.WaitForScreen(h.ctx, func(screen []string) bool {
		position := -1
		for _, want := range wants {
			row := screenRowIndexContaining(screen, want)
			if row < 0 || row < position {
				return false
			}
			position = row
		}
		return true
	})
	if err != nil {
		h.t.Fatalf("waiting for text in order %q: %v", wants, err)
	}
	h.session.addTrace(fmt.Sprintf("matched text in order %q", wants))
	return screen
}

func (h *Harness) WaitStyled(predicate func(styledScreen) bool) styledScreen {
	h.t.Helper()
	screen, err := h.session.WaitForStyledScreen(h.ctx, predicate)
	if err != nil {
		h.t.Fatalf("waiting for styled screen: %v", err)
	}
	h.session.addTrace("matched styled screen")
	return screen
}

func (h *Harness) WaitStableText(want string, samples int) []string {
	h.t.Helper()
	screen, err := h.session.WaitForStableScreen(h.ctx, samples, func(screen []string) bool {
		return screenContains(screen, want)
	})
	if err != nil {
		h.t.Fatalf("waiting for stable %q: %v", want, err)
	}
	h.session.addTrace(fmt.Sprintf("matched stable text %q", want))
	return screen
}

func (h *Harness) ExpectStartupError(wants ...string) {
	h.t.Helper()
	if err := h.session.ExpectExit(h.ctx, 1); err != nil {
		h.t.Fatalf("startup did not exit with status 1: %v", err)
	}
	output := h.session.RawOutput()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			h.t.Fatalf("startup output does not contain %q:\n%s", want, tailString(output, 64*1024))
		}
	}
}

func (h *Harness) Quit() {
	h.t.Helper()
	h.Key("q")
	if err := h.session.WaitContext(h.ctx); err != nil {
		h.t.Fatalf("jjui exited with error: %v", err)
	}
}

var harnessLetterKeys = map[byte]ghostty.Key{
	'A': ghostty.KeyA, 'B': ghostty.KeyB, 'C': ghostty.KeyC, 'D': ghostty.KeyD,
	'E': ghostty.KeyE, 'F': ghostty.KeyF, 'G': ghostty.KeyG, 'H': ghostty.KeyH,
	'I': ghostty.KeyI, 'J': ghostty.KeyJ, 'K': ghostty.KeyK, 'L': ghostty.KeyL,
	'M': ghostty.KeyM, 'N': ghostty.KeyN, 'O': ghostty.KeyO, 'P': ghostty.KeyP,
	'Q': ghostty.KeyQ, 'R': ghostty.KeyR, 'S': ghostty.KeyS, 'T': ghostty.KeyT,
	'U': ghostty.KeyU, 'V': ghostty.KeyV, 'W': ghostty.KeyW, 'X': ghostty.KeyX,
	'Y': ghostty.KeyY, 'Z': ghostty.KeyZ,
}

func harnessKey(name string) (ghostty.Key, string, ghostty.Mods, error) {
	if len(name) == 1 {
		char := name[0]
		upper := char
		if upper >= 'a' && upper <= 'z' {
			upper -= 'a' - 'A'
		}
		if key, ok := harnessLetterKeys[upper]; ok {
			var mods ghostty.Mods
			if char >= 'A' && char <= 'Z' {
				mods = ghostty.ModShift
			}
			return key, name, mods, nil
		}
		switch name {
		case " ":
			return ghostty.KeySpace, name, 0, nil
		case "/":
			return ghostty.KeySlash, name, 0, nil
		case "'":
			return ghostty.KeyQuote, name, 0, nil
		}
	}

	special := map[string]struct {
		key  ghostty.Key
		text string
	}{
		"Enter":      {ghostty.KeyEnter, ""},
		"Escape":     {ghostty.KeyEscape, ""},
		"Space":      {ghostty.KeySpace, " "},
		"ArrowUp":    {ghostty.KeyArrowUp, ""},
		"ArrowDown":  {ghostty.KeyArrowDown, ""},
		"ArrowLeft":  {ghostty.KeyArrowLeft, ""},
		"ArrowRight": {ghostty.KeyArrowRight, ""},
	}
	if event, ok := special[name]; ok {
		return event.key, event.text, 0, nil
	}
	return 0, "", 0, fmt.Errorf("unsupported harness key %q", name)
}
