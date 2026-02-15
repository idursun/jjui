package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertCommand_ChangeId(t *testing.T) {
	cmd := customCommand{
		name: "my-edit",
		key:  []string{"E"},
		args: []string{"edit", "$change_id"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "my-edit", result.name)
	assert.Equal(t, "ui.revisions", result.scope)
	assert.Contains(t, result.toml, "lua = '''\njj(\"edit\", context.change_id())\n'''")
	assert.Contains(t, result.toml, `key = "E"`)
	assert.Contains(t, result.toml, `scope = "ui.revisions"`)
}

func TestConvertCommand_EmbeddedPlaceholder(t *testing.T) {
	cmd := customCommand{
		name: "prune",
		key:  []string{"ctrl+a"},
		args: []string{"abandon", "-r", "$change_id::"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "ui.revisions", result.scope)
	assert.Contains(t, result.toml, `context.change_id() .. "::"`)
}

func TestConvertCommand_PlaceholderWithPrefix(t *testing.T) {
	cmd := customCommand{
		name: "log-from",
		key:  []string{"L"},
		args: []string{"log", "-r", "::$change_id"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Contains(t, result.toml, `"::" .. context.change_id()`)
}

func TestConvertCommand_PlaceholderWithPrefixAndSuffix(t *testing.T) {
	cmd := customCommand{
		name: "range",
		key:  []string{"r"},
		args: []string{"log", "-r", "trunk()..$change_id-"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Contains(t, result.toml, `"trunk().." .. context.change_id() .. "-"`)
}

func TestConvertCommand_CheckedChangeIds(t *testing.T) {
	cmd := customCommand{
		name: "rebase-multi",
		key:  []string{"R"},
		args: []string{"rebase", "-r", "$checked_change_ids"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "ui.revisions", result.scope)
	assert.Contains(t, result.toml, `context.checked_change_ids()`)
}

func TestConvertCommand_File(t *testing.T) {
	cmd := customCommand{
		name: "restore-file",
		key:  []string{"r"},
		args: []string{"restore", "$file"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "revisions.details", result.scope)
	assert.Contains(t, result.toml, `context.file()`)
}

func TestConvertCommand_CheckedFiles(t *testing.T) {
	cmd := customCommand{
		name: "restore-files",
		key:  []string{"R"},
		args: []string{"restore", "$checked_files"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "revisions.details", result.scope)
	assert.Contains(t, result.toml, `context.checked_files()`)
}

func TestConvertCommand_OperationId(t *testing.T) {
	cmd := customCommand{
		name: "op-restore",
		key:  []string{"r"},
		args: []string{"op", "restore", "$operation_id"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "ui.oplog", result.scope)
	assert.Contains(t, result.toml, `context.operation_id()`)
}

func TestConvertCommand_NoPlaceholders(t *testing.T) {
	cmd := customCommand{
		name: "status",
		key:  []string{"s"},
		args: []string{"status"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "ui.revisions", result.scope)
	assert.Contains(t, result.toml, "lua = '''\njj(\"status\")\n'''")
}

func TestConvertCommand_Interactive(t *testing.T) {
	cmd := customCommand{
		name: "split-interactive",
		key:  []string{"S"},
		args: []string{"split", "-r", "$change_id"},
		show: "interactive",
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Contains(t, result.toml, `jj_interactive(`)
}

func TestConvertCommand_DiffSkipped(t *testing.T) {
	cmd := customCommand{
		name: "show-diff",
		key:  []string{"d"},
		args: []string{"diff", "-r", "$change_id"},
		show: "diff",
	}
	_, err := convertCommand(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "diff")
}

func TestConvertCommand_MultipleKeys(t *testing.T) {
	cmd := customCommand{
		name: "my-cmd",
		key:  []string{"a", "b"},
		args: []string{"status"},
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Contains(t, result.toml, `key = ["a", "b"]`)
}

func TestParseCustomCommands(t *testing.T) {
	content := `
[custom_commands]
"my-edit" = { key = ["E"], args = ["edit", "$change_id"] }
"status" = { key = ["s"], args = ["status"] }
`
	cmds, err := parseCustomCommands(content)
	require.NoError(t, err)
	assert.Len(t, cmds, 2)
	// Sorted alphabetically
	assert.Equal(t, "my-edit", cmds[0].name)
	assert.Equal(t, "status", cmds[1].name)
}

func TestParseCustomCommands_NoSection(t *testing.T) {
	content := `
[ui]
auto_refresh_interval = 5
`
	cmds, err := parseCustomCommands(content)
	require.NoError(t, err)
	assert.Nil(t, cmds)
}

func TestParseCustomCommands_WithShow(t *testing.T) {
	content := `
[custom_commands]
"split" = { key = ["S"], args = ["split", "-r", "$change_id"], show = "interactive" }
`
	cmds, err := parseCustomCommands(content)
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	assert.Equal(t, "interactive", cmds[0].show)
}

func TestRemoveCustomCommandsSection(t *testing.T) {
	content := `[ui]
auto_refresh_interval = 5

[custom_commands]
"my-edit" = { key = ["E"], args = ["edit", "$change_id"] }
"status" = { key = ["s"], args = ["status"] }

[preview]
show_at_start = true
`
	result := removeCustomCommandsSection(content)
	assert.NotContains(t, result, "custom_commands")
	assert.NotContains(t, result, "my-edit")
	assert.Contains(t, result, "[ui]")
	assert.Contains(t, result, "[preview]")
}

func TestMigrate_BackupCreation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", dir)

	configContent := `
[custom_commands]
"my-edit" = { key = ["E"], args = ["edit", "$change_id"] }
`
	configFile := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0o644))

	code := Migrate()
	assert.Equal(t, 0, code)

	// Verify backup was created
	backupFile := filepath.Join(dir, "config.old.toml")
	backupData, err := os.ReadFile(backupFile)
	require.NoError(t, err)
	assert.Equal(t, configContent, string(backupData))

	// Verify config was updated
	updatedData, err := os.ReadFile(configFile)
	require.NoError(t, err)
	updated := string(updatedData)
	assert.NotContains(t, updated, "[custom_commands]")
	assert.Contains(t, updated, "[[actions]]")
	assert.Contains(t, updated, "[[bindings]]")
}

func TestMigrate_ReRunUsesOldFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", dir)

	// config.old.toml has the original custom_commands
	oldContent := `
[custom_commands]
"status" = { key = ["s"], args = ["status"] }
`
	// config.toml was already migrated (no custom_commands)
	currentContent := `
[[actions]]
name = "outdated"
lua = "jj()"
`

	configFile := filepath.Join(dir, "config.toml")
	backupFile := filepath.Join(dir, "config.old.toml")
	require.NoError(t, os.WriteFile(configFile, []byte(currentContent), 0o644))
	require.NoError(t, os.WriteFile(backupFile, []byte(oldContent), 0o644))

	code := Migrate()
	assert.Equal(t, 0, code)

	// Verify backup was NOT overwritten
	backupData, err := os.ReadFile(backupFile)
	require.NoError(t, err)
	assert.Equal(t, oldContent, string(backupData))

	// Verify config.toml was re-generated from old file
	updatedData, err := os.ReadFile(configFile)
	require.NoError(t, err)
	updated := string(updatedData)
	assert.NotContains(t, updated, "[custom_commands]")
	assert.NotContains(t, updated, "outdated")
	assert.Contains(t, updated, `name = "status"`)
	assert.Contains(t, updated, `jj("status")`)
}

func TestMigrate_NoCustomCommands(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", dir)

	configContent := `
[ui]
auto_refresh_interval = 5
`
	configFile := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0o644))

	code := Migrate()
	assert.Equal(t, 0, code)

	// Config should be unchanged
	data, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, configContent, string(data))
}

func TestDetermineScope(t *testing.T) {
	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{"edit", "$change_id"}, "ui.revisions"},
		{[]string{"rebase", "$checked_change_ids"}, "ui.revisions"},
		{[]string{"restore", "$file"}, "revisions.details"},
		{[]string{"restore", "$checked_files"}, "revisions.details"},
		{[]string{"op", "restore", "$operation_id"}, "ui.oplog"},
		{[]string{"status"}, "ui.revisions"},
		{[]string{"abandon", "-r", "$change_id::"}, "ui.revisions"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, determineScope(tt.args))
		})
	}
}

func TestBuildLuaArgs(t *testing.T) {
	args := []string{"edit", "-r", "$change_id"}
	result := buildLuaArgs(args)
	assert.Equal(t, []string{`"edit"`, `"-r"`, `context.change_id()`}, result)
}

func TestBuildLuaArg_EmbeddedPlaceholder(t *testing.T) {
	assert.Equal(t, `context.change_id() .. "::"`, buildLuaArg("$change_id::"))
	assert.Equal(t, `"::" .. context.change_id()`, buildLuaArg("::$change_id"))
	assert.Equal(t, `"trunk().." .. context.change_id() .. "-"`, buildLuaArg("trunk()..$change_id-"))
}

func TestConvertCommand_Revset(t *testing.T) {
	cmd := customCommand{
		name:   "show after revisions",
		key:    []string{"M"},
		revset: "::$change_id",
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "ui.revisions", result.scope)
	assert.Contains(t, result.toml, `revset.set("::" .. context.change_id())`)
}

func TestConvertCommand_RevsetNoPlaceholder(t *testing.T) {
	cmd := customCommand{
		name:   "show trunk",
		key:    []string{"T"},
		revset: "trunk()..@",
	}
	result, err := convertCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "ui.revisions", result.scope)
	assert.Contains(t, result.toml, `revset.set("trunk()..@")`)
}

func TestParseCustomCommands_WithRevset(t *testing.T) {
	content := `
[custom_commands]
"show after" = { key = ["M"], revset = "::$change_id" }
`
	cmds, err := parseCustomCommands(content)
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	assert.Equal(t, "::$change_id", cmds[0].revset)
	assert.Nil(t, cmds[0].args)
}

func TestBuildLuaArg_MultiplePlaceholders(t *testing.T) {
	arg := "all:(parents(@) | $change_id) ~ (parents(@) & $change_id)"
	result := buildLuaArg(arg)
	assert.Equal(t, `"all:(parents(@) | " .. context.change_id() .. ") ~ (parents(@) & " .. context.change_id() .. ")"`, result)
}
