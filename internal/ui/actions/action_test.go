package actions

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestAction_UnmarshalTOML_WithSimpleSyntax(t *testing.T) {
	actionData := "some_action"
	var action Action
	err := action.UnmarshalTOML(actionData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.NotNil(t, action)
	assert.Equal(t, "some_action", action.Id)
}

func TestAction_UnmarshalTOML_WithComplexSyntax(t *testing.T) {
	actionData := `
id = "complex_action"
next = [
   { jj = ["log", "-p"], next = ["refresh"] },
   { wait = "refresh" },
]
`
	var action Action

	err := toml.Unmarshal([]byte(actionData), &action)
	assert.NoError(t, err)
	assert.NotNil(t, action.Next)
	assert.Len(t, action.Next, 2)
	assert.Equal(t, "run", action.Next[0].Id)
	assert.Equal(t, "wait refresh", action.Next[1].Id)
}

func TestAction_UnmarshalTOML_ImplicitArgs(t *testing.T) {
	actionData := `
action = { id = "flash.add", message = "$output" }
`
	var data struct {
		Action Action `toml:"action"`
	}

	err := toml.Unmarshal([]byte(actionData), &data)
	assert.NoError(t, err)
	assert.NotNil(t, data.Action)
	assert.Equal(t, "flash.add", data.Action.Id)
	message, exists := data.Action.Args["message"]
	assert.True(t, exists)
	assert.Equal(t, "$output", message)
}
