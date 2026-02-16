package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTheme(t *testing.T) {
	themeData := []byte(`
title = { fg = "blue", bold = true }
selected = { fg = "white", bg = "blue" }
error = "red"
`)

	theme, err := loadTheme(themeData, nil)
	require.NoError(t, err)

	expected := map[string]Color{
		"title":    {Fg: "blue", Bold: true},
		"selected": {Fg: "white", Bg: "blue"},
		"error":    {Fg: "red"},
	}

	assert.EqualExportedValues(t, expected, theme)
}

func TestLoadThemeWithBase(t *testing.T) {
	baseTheme := map[string]Color{
		"title":    {Fg: "green", Bold: true},
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
		"title":    {Fg: "magenta", Bold: true},
		"selected": {Fg: "yellow", Bg: "blue"},
		"error":    {Fg: "red"},
		"border":   {Fg: "white"},
	}

	assert.EqualExportedValues(t, expected, theme)
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

	require.NoError(t, cfg.Load(content))
	require.Len(t, cfg.Actions, 3)
	assert.Equal(t, "print('override')", cfg.Actions[0].Lua)
	assert.Equal(t, "my_action", cfg.Actions[1].Name)
	assert.Equal(t, "new_action", cfg.Actions[2].Name)
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

	require.NoError(t, cfg.Load(content))

	assert.Contains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "revisions.move_up",
		Key:    StringList{"up"},
	})
	assert.NotContains(t, cfg.Bindings, BindingConfig{
		Scope:  "revisions",
		Action: "ui.open_git",
		Seq:    StringList{"g", "g"},
	})
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

	require.NoError(t, cfg.Load(content))

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
