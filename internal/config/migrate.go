package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Migrate converts legacy [custom_commands] config to [[actions]] + [[bindings]].
// It returns an exit code (0 for success, non-zero for failure).
func Migrate() int {
	return migrateWith(os.Stdout, os.Stderr)
}

func migrateWith(stdout, stderr io.Writer) int {
	configFile := getConfigFilePath()
	if configFile == "" {
		fmt.Fprintln(stderr, "Error: could not determine config file path")
		return 1
	}

	configDir := filepath.Dir(configFile)
	backupFile := filepath.Join(configDir, "config.old.toml")

	// Determine source file: use config.old.toml if it exists (re-run), otherwise config.toml
	sourceFile := configFile
	if _, err := os.Stat(backupFile); err == nil {
		sourceFile = backupFile
		fmt.Fprintln(stdout, "Old file found. Migrating from old file.")
	}

	// Read source config
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading config file: %v\n", err)
		return 1
	}
	content := string(data)

	// Check if custom_commands exists
	commands, err := parseCustomCommands(content)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing config: %v\n", err)
		return 1
	}
	if len(commands) == 0 {
		fmt.Fprintln(stdout, "No [custom_commands] found in config. Nothing to migrate.")
		return 0
	}

	// Create backup if it doesn't exist yet
	if sourceFile == configFile {
		if err := os.WriteFile(backupFile, data, 0o644); err != nil {
			fmt.Fprintf(stderr, "Error creating backup: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Backup created: %s\n", backupFile)
	}

	// Convert commands
	var actions []actionResult
	var warnings []string
	for _, cmd := range commands {
		result, err := convertCommand(cmd)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Skipping %q: %s", cmd.name, err.Error()))
			continue
		}
		actions = append(actions, result)
	}

	// Print warnings
	for _, w := range warnings {
		fmt.Fprintf(stderr, "Warning: %s\n", w)
	}

	if len(actions) == 0 {
		fmt.Fprintln(stdout, "No commands could be converted.")
		return 0
	}

	// Remove [custom_commands] section from config
	cleaned := removeCustomCommandsSection(content)

	// Append generated actions and bindings
	var sb strings.Builder
	sb.WriteString(strings.TrimRight(cleaned, "\n\t "))
	sb.WriteString("\n\n")
	for _, a := range actions {
		sb.WriteString(a.toml)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(configFile, []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintf(stderr, "Error writing config file: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Migrated %d command(s):\n", len(actions))
	for _, a := range actions {
		fmt.Fprintf(stdout, "  - %s (scope: %s)\n", a.name, a.scope)
	}

	return 0
}

type customCommand struct {
	name   string
	key    []string
	args   []string
	revset string
	show   string
}

type actionResult struct {
	name  string
	scope string
	toml  string
}

// parseCustomCommands extracts custom_commands from raw TOML content.
func parseCustomCommands(content string) ([]customCommand, error) {
	var raw map[string]any
	if _, err := toml.Decode(content, &raw); err != nil {
		return nil, err
	}

	section, ok := raw["custom_commands"]
	if !ok {
		return nil, nil
	}

	cmds, ok := section.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("[custom_commands] is not a table")
	}

	// Sort names for deterministic output
	names := make([]string, 0, len(cmds))
	for name := range cmds {
		names = append(names, name)
	}
	sort.Strings(names)

	var result []customCommand
	for _, name := range names {
		val := cmds[name]
		entry, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("custom_commands.%s: expected a table", name)
		}

		cmd := customCommand{name: name}

		if k, ok := entry["key"]; ok {
			cmd.key = toStringSlice(k)
		}
		if a, ok := entry["args"]; ok {
			cmd.args = toStringSlice(a)
		}
		if r, ok := entry["revset"]; ok {
			if str, ok := r.(string); ok {
				cmd.revset = str
			}
		}
		if s, ok := entry["show"]; ok {
			if str, ok := s.(string); ok {
				cmd.show = str
			}
		}

		result = append(result, cmd)
	}

	return result, nil
}

func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{val}
	default:
		return nil
	}
}

// convertCommand converts a single custom command to action + binding TOML.
func convertCommand(cmd customCommand) (actionResult, error) {
	if cmd.show == "diff" {
		return actionResult{}, fmt.Errorf("show = \"diff\" is not supported in the new system")
	}

	var lua string
	var scope string

	if cmd.revset != "" {
		// Revset command → revset.set(...)
		scope = determineScope([]string{cmd.revset})
		lua = fmt.Sprintf("revset.set(%s)", buildLuaArg(cmd.revset))
	} else {
		// Regular command → jj(...) or jj_interactive(...)
		scope = determineScope(cmd.args)
		luaFunc := "jj"
		if cmd.show == "interactive" {
			luaFunc = "jj_interactive"
		}
		luaArgs := buildLuaArgs(cmd.args)
		lua = fmt.Sprintf("%s(%s)", luaFunc, strings.Join(luaArgs, ", "))
	}

	var sb strings.Builder

	// Generate [[actions]]
	sb.WriteString("[[actions]]\n")
	fmt.Fprintf(&sb, "name = %q\n", cmd.name)
	fmt.Fprintf(&sb, "lua = '''\n%s\n'''\n", lua)

	// Generate [[bindings]]
	sb.WriteString("\n[[bindings]]\n")
	fmt.Fprintf(&sb, "action = %q\n", cmd.name)
	if len(cmd.key) > 0 {
		fmt.Fprintf(&sb, "key = %s\n", formatStringList(cmd.key))
	}
	fmt.Fprintf(&sb, "scope = %q\n", scope)

	return actionResult{
		name:  cmd.name,
		scope: scope,
		toml:  sb.String(),
	}, nil
}

func determineScope(args []string) string {
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "$file") || strings.Contains(joined, "$checked_files") {
		return "revisions.details"
	}
	if strings.Contains(joined, "$change_id") || strings.Contains(joined, "$checked_change_ids") {
		return "ui.revisions"
	}
	if strings.Contains(joined, "$operation_id") {
		return "ui.oplog"
	}
	return "ui.revisions"
}

// placeholders maps placeholder tokens to their Lua context expressions.
var placeholders = []struct {
	token       string
	replacement string
}{
	// Longer tokens first so $checked_change_ids matches before $change_id.
	{"$checked_change_ids", "context.checked_change_ids()"},
	{"$checked_files", "context.checked_files()"},
	{"$change_id", "context.change_id()"},
	{"$operation_id", "context.operation_id()"},
	{"$file", "context.file()"},
}

func buildLuaArg(arg string) string {
	// Scan through the arg, replacing all placeholder occurrences.
	var parts []string
	remaining := arg

	for remaining != "" {
		// Find the earliest placeholder match in remaining.
		bestIdx := -1
		var bestPlaceholder struct {
			token       string
			replacement string
		}
		for _, p := range placeholders {
			idx := strings.Index(remaining, p.token)
			if idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
				bestIdx = idx
				bestPlaceholder = p
			}
		}

		if bestIdx < 0 {
			// No more placeholders — rest is literal.
			parts = append(parts, fmt.Sprintf("%q", remaining))
			break
		}

		// Add literal prefix if any.
		if bestIdx > 0 {
			parts = append(parts, fmt.Sprintf("%q", remaining[:bestIdx]))
		}
		parts = append(parts, bestPlaceholder.replacement)
		remaining = remaining[bestIdx+len(bestPlaceholder.token):]
	}

	if len(parts) == 0 {
		return `""`
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, " .. ")
}

func buildLuaArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		result = append(result, buildLuaArg(arg))
	}
	return result
}

func formatStringList(items []string) string {
	if len(items) == 1 {
		return fmt.Sprintf("%q", items[0])
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// removeCustomCommandsSection removes the [custom_commands] section from TOML content.
func removeCustomCommandsSection(content string) string {
	// Match [custom_commands] header and everything until the next section header or EOF
	re := regexp.MustCompile(`(?m)^\[custom_commands\]\s*\n(?:(?:\s*(?:#[^\n]*)?\n)|(?:[^\[\n][^\n]*\n)|(?:"[^"]*"\s*=\s*\{[^}]*\}\s*\n))*`)
	return re.ReplaceAllString(content, "")
}
