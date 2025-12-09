package context

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoad_CustomCommands(t *testing.T) {
	content := `
[custom_commands]
"show diff" = { key = ["ctrl+d"],  args = ["diff", "-r", "$revision", "--color", "--always"], show = "diff" }
"restore evolog" = { key = ["ctrl+e"],  args = ["op", "restore", "-r", "$revision"] }
"resolve vscode" = { key = ["ctrl+r"],  args = ["resolve", "--tool", "vscode"], show = "interactive" }
"update revset" = { key = ["M"],  revset = "::$change_id" }
"sequence command" = { key_sequence = ["g", "s"], args = ["status"], desc = "status for change" }
`
	registry, err := LoadCustomCommands(content)
	assert.NoError(t, err)
	assert.Len(t, registry, 5)

	testCases := []struct {
		name        string
		commandName string
		testFunc    func(t *testing.T, cmd CustomCommand)
	}{
		{
			name:        "diff command",
			commandName: "show diff",
			testFunc: func(t *testing.T, cmd CustomCommand) {
				runCmd, ok := cmd.(CustomRunCommand)
				assert.True(t, ok, "Command should be CustomRunCommand")
				assert.Equal(t, []string{"ctrl+d"}, runCmd.Key)
				assert.Equal(t, []string{"diff", "-r", "$revision", "--color", "--always"}, runCmd.Args)
				assert.Equal(t, config.ShowOptionDiff, runCmd.Show)
				assert.Equal(t, "show diff", runCmd.Name)
			},
		},
		{
			name:        "restore command",
			commandName: "restore evolog",
			testFunc: func(t *testing.T, cmd CustomCommand) {
				runCmd, ok := cmd.(CustomRunCommand)
				assert.True(t, ok, "Command should be CustomRunCommand")
				assert.Equal(t, []string{"ctrl+e"}, runCmd.Key)
				assert.Equal(t, []string{"op", "restore", "-r", "$revision"}, runCmd.Args)
				assert.Equal(t, config.ShowOption(""), runCmd.Show)
				assert.Equal(t, "restore evolog", runCmd.Name)
			},
		},
		{
			name:        "resolve command",
			commandName: "resolve vscode",
			testFunc: func(t *testing.T, cmd CustomCommand) {
				runCmd, ok := cmd.(CustomRunCommand)
				assert.True(t, ok, "Command should be CustomRunCommand")
				assert.Equal(t, []string{"ctrl+r"}, runCmd.Key)
				assert.Equal(t, []string{"resolve", "--tool", "vscode"}, runCmd.Args)
				assert.Equal(t, config.ShowOptionInteractive, runCmd.Show)
				assert.Equal(t, "resolve vscode", runCmd.Name)
			},
		},
		{
			name:        "update revset command",
			commandName: "update revset",
			testFunc: func(t *testing.T, cmd CustomCommand) {
				revsetCmd, ok := cmd.(CustomRevsetCommand)
				assert.True(t, ok, "Command should be CustomRevsetCommand")
				assert.Equal(t, []string{"M"}, revsetCmd.Key)
				assert.Equal(t, "::$change_id", revsetCmd.Revset)
				assert.Equal(t, "update revset", revsetCmd.Name)
			},
		},
		{
			name:        "sequence command",
			commandName: "sequence command",
			testFunc: func(t *testing.T, cmd CustomCommand) {
				runCmd, ok := cmd.(CustomRunCommand)
				assert.True(t, ok, "Command should be CustomRunCommand")
				assert.Equal(t, []string{"g", "s"}, runCmd.KeySequence)
				assert.Equal(t, []string{"status"}, runCmd.Args)
				assert.Equal(t, "sequence command", runCmd.Name)
				assert.Equal(t, "status for change", runCmd.Desc)
				labeled, ok := any(runCmd).(LabeledCommand)
				assert.True(t, ok)
				assert.Equal(t, "status for change", labeled.Label())
				seq := runCmd.Sequence()
				assert.Len(t, seq, 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := registry[tc.commandName]
			assert.NotNil(t, cmd)
			tc.testFunc(t, cmd)
		})
	}
}
