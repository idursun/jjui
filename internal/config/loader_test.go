package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestLoadTheme(t *testing.T) {
	themeData := []byte(`
title = { fg = "blue", bold = true }
selected = { fg = "white", bg = "blue" }
error = "red"
`)

	theme, err := loadTheme(themeData, nil)
	require.NoError(t, err)

	expected := map[string]Color{
		"title":    {Fg: "blue", Bold: boolPtr(true)},
		"selected": {Fg: "white", Bg: "blue"},
		"error":    {Fg: "red"},
	}

	assert.EqualExportedValues(t, expected, theme)
}

func TestLoadThemeWithBase(t *testing.T) {
	baseTheme := map[string]Color{
		"title":    {Fg: "green", Bold: boolPtr(true)},
		"selected": {Fg: "cyan", Bg: "black"},
		"error":    {Fg: "red"},
		"border":   {Fg: "white"},
	}

	partialOverride := []byte(`
title = { fg = "magenta", bold = true }
selected = { fg = "yellow", bg = "blue" }
`)

	theme, err := loadTheme(partialOverride, baseTheme)
	require.NoError(t, err)

	expected := map[string]Color{
		"title":    {Fg: "magenta", Bold: boolPtr(true)},
		"selected": {Fg: "yellow", Bg: "blue"},
		"error":    {Fg: "red"},
		"border":   {Fg: "white"},
	}

	assert.EqualExportedValues(t, expected, theme)
}

func findLastActionByName(actions []ActionConfig, name string) (ActionConfig, bool) {
	for i := len(actions) - 1; i >= 0; i-- {
		if actions[i].Name == name {
			return actions[i], true
		}
	}
	return ActionConfig{}, false
}

func TestLoad_MergesActionsByName(t *testing.T) {
	cfg := &Config{
		Actions: []ActionConfig{
			{Name: "open_details_alias", Lua: `print("default")`},
			{Name: "my_action", Lua: "return 1"},
		},
	}

	content := `
[[actions]]
name = "open_details_alias"
lua = "print('override')"

[[actions]]
name = "new_action"
lua = "return 2"
`

	require.NoError(t, cfg.Load(content, ""))
	require.Len(t, cfg.Actions, 4)
	action, ok := findLastActionByName(cfg.Actions, "open_details_alias")
	require.True(t, ok)
	assert.Equal(t, "print('override')", action.Lua)
	_, ok = findLastActionByName(cfg.Actions, "new_action")
	assert.True(t, ok)
}

func TestLoad_MergesBindingsByShadowRules(t *testing.T) {
	cfg := &Config{
		Bindings: []BindingConfig{
			{Scope: "revisions", Action: "revisions.move_up", Key: StringList{"k", "up"}},
			{Scope: "revisions", Action: "ui.open_git", Seq: StringList{"g", "g"}},
			{Scope: "ui", Action: "ui.quit", Key: StringList{"q"}},
		},
	}

	content := `
[[bindings]]
scope = "revisions"
action = "revisions.jump_to_parent"
key = "k"

[[bindings]]
scope = "revisions"
action = "ui.open_oplog"
seq = ["g", "g"]

[[bindings]]
scope = "ui"
action = "revset.cancel"
key = "esc"
`

	require.NoError(t, cfg.Load(content, ""))

	// original bindings remain in the slice (shadowing happens at runtime)
	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "revisions.move_up",
		Key:    StringList{"k", "up"},
	})
	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "ui.open_git",
		Seq:    StringList{"g", "g"},
	})
	// overlay bindings are appended
	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "revisions.jump_to_parent",
		Key:    StringList{"k"},
	})
	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "ui.open_oplog",
		Seq:    StringList{"g", "g"},
	})
	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "ui",
		Action: "ui.quit",
		Key:    StringList{"q"},
	})
	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "ui",
		Action: "revset.cancel",
		Key:    StringList{"esc"},
	})
}

func TestLoadDefaultConfig_SetsWindowTitleByDefault(t *testing.T) {
	cfg := loadDefaultConfig()

	assert.True(t, cfg.UI.SetWindowTitle)
}

func TestLoad_UISetWindowTitleCanBeDisabled(t *testing.T) {
	cfg := loadDefaultConfig()

	require.NoError(t, cfg.Load(`
[ui]
set_window_title = false
`, ""))

	assert.False(t, cfg.UI.SetWindowTitle)
}

func TestLoad_SeqBindingDoesNotInheritStaleKey(t *testing.T) {
	cfg := &Config{
		Bindings: []BindingConfig{
			{Scope: "ui", Action: "revset.cancel", Key: StringList{"esc"}},
		},
	}

	content := `
[[actions]]
name = "say hello"
lua = "flash('hello')"

[[bindings]]
action = "say hello"
seq = ["w", "h"]
scope = "ui"
`

	require.NoError(t, cfg.Load(content, ""))

	found := false
	for _, b := range cfg.Bindings {
		if b.Action != "say hello" || b.Scope != "ui" {
			continue
		}
		found = true
		assert.Empty(t, b.Key)
		assert.Equal(t, StringList{"w", "h"}, b.Seq)
	}
	assert.True(t, found, "expected merged binding for action 'say hello'")
}

func TestLoadFromDir_BindingsProfileRelativeToBaseDir(t *testing.T) {
	dir := t.TempDir()

	profileContent := `
[[bindings]]
scope = "revisions"
action = "revisions.move_up"
key = "j"
`
	err := os.WriteFile(filepath.Join(dir, "my-profile.toml"), []byte(profileContent), 0600)
	require.NoError(t, err)

	configContent := `bindings_profile = "my-profile.toml"`

	cfg := &Config{}
	err = cfg.Load(configContent, dir)
	require.NoError(t, err)

	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "revisions.move_up",
		Key:    StringList{"j"},
	})
}

func TestLoadFromDir_BindingsProfileAbsPathIgnoresBaseDir(t *testing.T) {
	dir := t.TempDir()

	profileContent := `
[[bindings]]
scope = "revisions"
action = "revisions.move_down"
key = "k"
`
	profilePath := filepath.Join(dir, "abs-profile.toml")
	err := os.WriteFile(profilePath, []byte(profileContent), 0600)
	require.NoError(t, err)

	// Use an absolute path — baseDir should be irrelevant
	configContent := `bindings_profile = "` + profilePath + `"`

	cfg := &Config{}
	err = cfg.Load(configContent, "/some/other/dir")
	require.NoError(t, err)

	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "revisions.move_down",
		Key:    StringList{"k"},
	})
}

func TestEnvConfigDir_InvalidFallsBackToStandardConfig(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	standardConfigDir := filepath.Join(configHome, "jjui")
	require.NoError(t, os.MkdirAll(standardConfigDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(standardConfigDir, "config.toml"), []byte("limit = 10\n"), 0o600))

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("JJUI_CONFIG_DIR", filepath.Join(t.TempDir(), "missing"))

	assert.Equal(t, "", EnvConfigDir())
	assert.Equal(t, filepath.Join(standardConfigDir, "config.toml"), getConfigFilePath())
}

// TestConfigLayering verifies the precedence chain:
// - Without JJUI_CONFIG_DIR: global config then repo-local (repo wins on conflict)
// - With JJUI_CONFIG_DIR: env config only, both global and repo-local are skipped
func TestConfigLayering(t *testing.T) {
	envDir := t.TempDir()
	repoConfigDir := t.TempDir()

	// Global: sets all three fields
	globalConfig := `
limit = 10
[ui]
auto_refresh_interval = 5
flash_message_display_seconds = 7
`

	// Repo-local: overrides limit, leaves the others
	repoConfig := `
limit = 99
`

	// Env: only sets auto_refresh_interval; replaces both global and repo-local
	envConfig := `
[ui]
auto_refresh_interval = 30
`
	require.NoError(t, os.WriteFile(filepath.Join(envDir, "config.toml"), []byte(envConfig), 0600))
	t.Setenv("JJUI_CONFIG_DIR", envDir)

	// Simulate what main.go does when JJUI_CONFIG_DIR is set:
	// LoadConfigFile() routes through getConfigFilePath() which returns env dir,
	// so it loads the env config. Repo-local is skipped.
	cfg := &Config{}
	data, err := LoadConfigFile()
	require.NoError(t, err)
	require.NoError(t, cfg.Load(string(data), GetConfigDir()))

	// auto_refresh_interval comes from env config
	assert.Equal(t, 30, cfg.UI.AutoRefreshInterval)
	// global and repo values are absent — env replaced everything
	assert.Equal(t, 0, cfg.Limit)
	assert.Equal(t, 0, cfg.UI.FlashMessageDisplaySeconds)

	// Now simulate without JJUI_CONFIG_DIR: global then repo-local
	t.Setenv("JJUI_CONFIG_DIR", "")
	cfg2 := &Config{}
	require.NoError(t, cfg2.Load(globalConfig, ""))
	require.NoError(t, cfg2.Load(repoConfig, repoConfigDir))

	// flash_message_display_seconds: only in global, survives
	assert.Equal(t, 7, cfg2.UI.FlashMessageDisplaySeconds)
	// limit: global=10, overridden by repo=99
	assert.Equal(t, 99, cfg2.Limit)
	// auto_refresh_interval: only in global, survives
	assert.Equal(t, 5, cfg2.UI.AutoRefreshInterval)
}

func TestSetupLuaTypes_WritesTypesAndLuaRC(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", configDir)

	result, err := SetupLuaTypes()
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, filepath.Join(configDir, "types.lua"), result.TypesPath)
	assert.Equal(t, filepath.Join(configDir, ".luarc.json"), result.LuaRCPath)
	assert.True(t, result.LuaRCCreated)

	typesData, err := os.ReadFile(result.TypesPath)
	require.NoError(t, err)
	embeddedTypes, err := configFS.ReadFile("default/types.lua")
	require.NoError(t, err)
	assert.Equal(t, string(embeddedTypes), string(typesData))

	luaRCData, err := os.ReadFile(result.LuaRCPath)
	require.NoError(t, err)

	var luaRC struct {
		Workspace struct {
			Library []string `json:"library"`
		} `json:"workspace"`
	}
	require.NoError(t, json.Unmarshal(luaRCData, &luaRC))
	assert.Equal(t, []string{result.TypesPath}, luaRC.Workspace.Library)
}

func TestSetupLuaTypes_PreservesExistingLuaRC(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", configDir)

	luaRCPath := filepath.Join(configDir, ".luarc.json")
	existingLuaRC := []byte("{\"workspace\":{\"library\":[\"/tmp/custom-types.lua\"]}}\n")
	require.NoError(t, os.WriteFile(luaRCPath, existingLuaRC, 0o644))

	typesPath := filepath.Join(configDir, "types.lua")
	require.NoError(t, os.WriteFile(typesPath, []byte("stale"), 0o644))

	result, err := SetupLuaTypes()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.LuaRCCreated)

	luaRCData, err := os.ReadFile(luaRCPath)
	require.NoError(t, err)
	assert.Equal(t, string(existingLuaRC), string(luaRCData))

	typesData, err := os.ReadFile(typesPath)
	require.NoError(t, err)
	embeddedTypes, err := configFS.ReadFile("default/types.lua")
	require.NoError(t, err)
	assert.Equal(t, string(embeddedTypes), string(typesData))
}
