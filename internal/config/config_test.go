package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
[ui]
highlight_light = "#a0a0a0"
`
	config := load(content)
	assert.Equal(t, "#a0a0a0", config.UI.HighlightLight)
}

func TestLoad_CustomCommands(t *testing.T) {
	content := `
[custom_commands]
"show diff" = { key = ["ctrl+d"],  args = ["diff", "-r", "$revision", "--color", "--always"], show = "diff" }
"restore evolog" = { key = ["ctrl+e"],  args = ["op", "restore", "-r", "$revision"] }
"resolve vscode" = { key = ["ctrl+r"],  args = ["resolve", "--tool", "vscode"], show = "interactive" }
`
	config := load(content)
	assert.Len(t, config.CustomCommands, 3)

	testCases := []struct {
		name        string
		commandName string
		expected    CustomCommandDefinition
	}{
		{
			name:        "diff command",
			commandName: "show diff",
			expected: CustomCommandDefinition{
				Key:  []string{"ctrl+d"},
				Args: []string{"diff", "-r", "$revision", "--color", "--always"},
				Show: ShowOptionDiff,
			},
		},
		{
			name:        "restore command",
			commandName: "restore evolog",
			expected: CustomCommandDefinition{
				Key:  []string{"ctrl+e"},
				Args: []string{"op", "restore", "-r", "$revision"},
				Show: "",
			},
		},
		{
			name:        "resolve command",
			commandName: "resolve vscode",
			expected: CustomCommandDefinition{
				Key:  []string{"ctrl+r"},
				Args: []string{"resolve", "--tool", "vscode"},
				Show: ShowOptionInteractive,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := config.CustomCommands[tc.commandName]
			assert.Equal(t, tc.expected.Key, cmd.Key)
			assert.Equal(t, tc.expected.Args, cmd.Args)
			assert.Equal(t, tc.expected.Show, cmd.Show)
		})
	}
}
