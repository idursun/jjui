package config

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/stretchr/testify/assert"
)

func TestLoad_Colours(t *testing.T) {
	content := `
[ui.colors]
"text" = "white"
"selected" = { fg = "blue", bg = "black" }
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.UI.Colors, 2)
	assert.Equal(t, "white", config.UI.Colors["text"].Fg)
	assert.Equal(t, "blue", config.UI.Colors["selected"].Fg)
	assert.Equal(t, "black", config.UI.Colors["selected"].Bg)
}

func TestLoad_Theme_Simple(t *testing.T) {
	content := `
[ui]
theme = "my-theme"
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, "my-theme", config.UI.Theme.Light)
	assert.Equal(t, "my-theme", config.UI.Theme.Dark)
}

func TestLoad_Theme_Nested(t *testing.T) {
	content := `
[ui.theme]
dark = "dark-theme"
light = "light-theme"
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, "dark-theme", config.UI.Theme.Dark)
	assert.Equal(t, "light-theme", config.UI.Theme.Light)
}

func TestLoad_AutoRefreshInterval(t *testing.T) {
	content := `
[ui]
auto_refresh_interval = 5000
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, 5000, config.UI.AutoRefreshInterval)
}

func TestLoad_Colors_StringAndObject(t *testing.T) {
	content := `
[ui.colors]
simple = "red"
complex = { fg = "blue", bg = "white", bold = true }
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.UI.Colors, 2)

	assert.Equal(t, "red", config.UI.Colors["simple"].Fg)
	assert.Equal(t, "", config.UI.Colors["simple"].Bg)
	assert.False(t, config.UI.Colors["simple"].Bold)

	assert.Equal(t, "blue", config.UI.Colors["complex"].Fg)
	assert.Equal(t, "white", config.UI.Colors["complex"].Bg)
	assert.True(t, config.UI.Colors["complex"].Bold)
}

func TestLoad_Actions(t *testing.T) {
	content := `
[actions]
  "revset.edit" = { args = { clear = true }, next = ["switch revset"]}
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.Actions, 1)
	action, exists := config.Actions["revset.edit"]
	assert.True(t, exists)
	assert.Equal(t, true, action.Args["clear"])
	assert.Equal(t, []actions.Action{{Id: "switch revset"}}, action.Next)
}

func TestLoad_Actions_Next(t *testing.T) {
	content := `
[actions]
  "revset.append" = { id = "revset.set", args = { revset = "$revset | ancestors($change_id, 1)" }, next = ["refresh"]}
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.Actions, 1)
	action, exists := config.Actions["revset.append"]
	assert.True(t, exists)
	assert.Equal(t, "$revset | ancestors($change_id, 1)", action.Args["revset"])
	assert.Equal(t, []actions.Action{{Id: "refresh"}}, action.Next)
}

func TestLoad_Actions_NestedNext(t *testing.T) {
	content := `
[actions]
  "new commit" = { next = [{ id = "run", args = { "jj" = ["log"] }, next = ["refresh"] }, "wait refresh", "close commit"] }
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.Actions, 1)
	action, exists := config.Actions["new commit"]
	assert.True(t, exists)
	assert.Len(t, action.Next, 3)
	assert.Len(t, action.Next[0].Next, 1)
}

func TestLoad_Actions_TableSyntax(t *testing.T) {
	content := `
[actions."show_root"]
  id = "run"
  args = { "jj" = ["root"] }
  next = [
    { id = "flash.add", args = { "message" = "$output" } },
    { id = "flash.add", args = { "message" = "hello" } },
  ]
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.Actions, 1)
	action, exists := config.Actions["show_root"]
	assert.True(t, exists)
	assert.Len(t, action.Next, 2)
}

func TestLoad_Actions_Shorthand(t *testing.T) {
	content := `
[actions."shorthand"]
  id = "run"
  args = { "jj" = ["root"] }
  next = [
    { choose = "simple_list", floating = "true" },
    { wait = { close = "simple_list" } },
	"refresh"
  ]
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.Actions, 1)
	action, exists := config.Actions["shorthand"]
	assert.True(t, exists)
	assert.Len(t, action.Next, 3)
	assert.Equal(t, "choose simple_list", action.Next[0].Id)
}

func TestLoad_ActionMap(t *testing.T) {
	content := `
[actions]
  "revset.edit" = { args = { clear = true }, next = ["switch revset"]}
[keymap.revisions]
  "j" = "revisions.down"
  "k" = "revisions.up"
  "L" = "revset.edit"
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.Bindings["revisions"], 3)
	assert.Equal(t, "revisions.down", config.Bindings["revisions"]["j"])
	assert.Equal(t, "revisions.up", config.Bindings["revisions"]["k"])
	assert.Equal(t, "revset.edit", config.Bindings["revisions"]["L"])
}
