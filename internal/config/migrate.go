package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

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

	sourceFile := configFile
	if _, err := os.Stat(backupFile); err == nil {
		sourceFile = backupFile
		fmt.Fprintf(stdout, "Migrating from '%s' to new format.\n", backupFile)
	}

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading config file: %v\n", err)
		return 1
	}
	content := string(data)

	commands, err := parseCustomCommands(content)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing config: %v\n", err)
		return 1
	}

	keys, err := parseKeys(content)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing keys: %v\n", err)
		return 1
	}

	if len(commands) == 0 && len(keys) == 0 {
		fmt.Fprintln(stdout, "No [custom_commands] or [keys] found in config. Nothing to migrate.")
		return 0
	}

	if sourceFile == configFile {
		if err := os.WriteFile(backupFile, data, 0o644); err != nil {
			fmt.Fprintf(stderr, "Error creating backup: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Backup created: %s\n", backupFile)
	}

	var warnings []string

	var actions []actionResult
	for _, cmd := range commands {
		result, err := convertCommand(cmd)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Skipping %q: %s", cmd.name, err.Error()))
			continue
		}
		actions = append(actions, result)
	}

	var keyBindings []bindingResult
	if len(keys) > 0 {
		var keyWarnings []string
		keyBindings, keyWarnings = convertKeys(keys)
		warnings = append(warnings, keyWarnings...)
	}

	for _, w := range warnings {
		fmt.Fprintf(stderr, "Warning: %s\n", w)
	}

	if len(actions) == 0 && len(keyBindings) == 0 {
		fmt.Fprintln(stdout, "No commands or keys could be converted.")
		return 0
	}

	cleaned, err := removeLegacySections(content)
	if err != nil {
		fmt.Fprintf(stderr, "Error removing legacy sections: %v\n", err)
		return 1
	}

	var sb strings.Builder
	sb.WriteString(strings.TrimRight(cleaned, "\n\t "))
	sb.WriteString("\n\n")
	for _, a := range actions {
		sb.WriteString(a.toml)
		sb.WriteString("\n")
	}
	for _, b := range keyBindings {
		sb.WriteString(b.toml)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(configFile, []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintf(stderr, "Error writing config file: %v\n", err)
		return 1
	}

	if len(actions) > 0 {
		fmt.Fprintf(stdout, "Migrated %d command(s):\n", len(actions))
		for _, a := range actions {
			fmt.Fprintf(stdout, "  - %s (scope: %s)\n", a.name, a.scope)
		}
	}
	if len(keyBindings) > 0 {
		fmt.Fprintf(stdout, "Migrated %d key binding(s)\n", len(keyBindings))
	}

	return 0
}

type customCommand struct {
	name        string
	key         []string
	keySequence []string
	args        []string
	revset      string
	show        string
	lua         string
}

type actionResult struct {
	name  string
	scope string
	toml  string
}

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
		if l, ok := entry["lua"]; ok {
			if str, ok := l.(string); ok {
				cmd.lua = str
			}
		}
		if ks, ok := entry["key_sequence"]; ok {
			cmd.keySequence = toStringSlice(ks)
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

func convertCommand(cmd customCommand) (actionResult, error) {
	if cmd.show == "diff" {
		return actionResult{}, fmt.Errorf("show = \"diff\" is not supported in the new system")
	}

	var lua string
	var scope string

	if cmd.lua != "" {
		lua = cmd.lua
		scope = "revisions"
	} else if cmd.revset != "" {
		scope = determineScope([]string{cmd.revset})
		lua = fmt.Sprintf("revset.set(%s)", buildLuaArg(cmd.revset))
	} else {
		scope = determineScope(cmd.args)
		luaFunc := "jj"
		if cmd.show == "interactive" {
			luaFunc = "jj_interactive"
		}
		luaArgs := buildLuaArgs(cmd.args)
		lua = fmt.Sprintf("%s(%s)", luaFunc, strings.Join(luaArgs, ", "))
	}

	var sb strings.Builder

	sb.WriteString("[[actions]]\n")
	fmt.Fprintf(&sb, "name = %q\n", cmd.name)
	fmt.Fprintf(&sb, "lua = '''\n%s\n'''\n", lua)

	sb.WriteString("\n[[bindings]]\n")
	fmt.Fprintf(&sb, "action = %q\n", cmd.name)
	if len(cmd.keySequence) > 0 {
		fmt.Fprintf(&sb, "seq = %s\n", formatStringList(cmd.keySequence))
	} else if len(cmd.key) > 0 {
		fmt.Fprintf(&sb, "key = %s\n", formatStringList(cmd.key))
	}
	fmt.Fprintf(&sb, "scope = %q\n", scope)
	fmt.Fprintf(&sb, "desc = %q\n", cmd.name)

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
		return "revisions"
	}
	if strings.Contains(joined, "$operation_id") {
		return "oplog"
	}
	return "revisions"
}

// Longer tokens first so $checked_change_ids matches before $change_id.
var placeholders = []struct {
	token       string
	replacement string
}{
	{"$checked_change_ids", "context.checked_change_ids()"},
	{"$checked_files", "context.checked_files()"},
	{"$change_id", "context.change_id()"},
	{"$operation_id", "context.operation_id()"},
	{"$file", "context.file()"},
}

func buildLuaArg(arg string) string {
	var parts []string
	remaining := arg

	for remaining != "" {
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
			parts = append(parts, fmt.Sprintf("%q", remaining))
			break
		}

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

type keyMapping struct {
	action string
	scope  string
	args   map[string]string
}

var keyMappings = map[string]keyMapping{
	"up":                      {action: "revisions.move_up", scope: "revisions"},
	"down":                    {action: "revisions.move_down", scope: "revisions"},
	"scroll_up":               {action: "revisions.page_up", scope: "revisions"},
	"scroll_down":             {action: "revisions.page_down", scope: "revisions"},
	"jump_to_parent":          {action: "revisions.jump_to_parent", scope: "revisions"},
	"jump_to_children":        {action: "revisions.jump_to_children", scope: "revisions"},
	"jump_to_working_copy":    {action: "revisions.jump_to_working_copy", scope: "revisions"},
	"cancel":                  {action: "ui.cancel", scope: "ui"},
	"toggle_select":           {action: "revisions.toggle_select", scope: "revisions"},
	"new":                     {action: "revisions.new", scope: "revisions"},
	"commit":                  {action: "revisions.commit", scope: "revisions"},
	"refresh":                 {action: "revisions.refresh", scope: "revisions"},
	"abandon":                 {action: "revisions.abandon", scope: "revisions"},
	"diff":                    {action: "revisions.diff", scope: "revisions"},
	"quit":                    {action: "ui.quit", scope: "ui"},
	"expand_status":           {action: "ui.expand_status", scope: "ui"},
	"describe":                {action: "revisions.describe", scope: "revisions"},
	"edit":                    {action: "revisions.edit", scope: "revisions"},
	"force_edit":              {action: "revisions.force_edit", scope: "revisions"},
	"diffedit":                {action: "revisions.diff_edit", scope: "revisions"},
	"absorb":                  {action: "revisions.absorb", scope: "revisions"},
	"split":                   {action: "revisions.split", scope: "revisions"},
	"split_parallel":          {action: "revisions.split_parallel", scope: "revisions"},
	"undo":                    {action: "ui.open_undo", scope: "revisions"},
	"redo":                    {action: "ui.open_redo", scope: "revisions"},
	"revset":                  {action: "ui.open_revset", scope: "revisions"},
	"exec_jj":                 {action: "ui.exec_jj", scope: "revisions"},
	"exec_shell":              {action: "ui.exec_shell", scope: "revisions"},
	"ace_jump":                {action: "revisions.ace_jump", scope: "revisions"},
	"quick_search":            {action: "ui.quick_search", scope: "revisions"},
	"quick_search_cycle":      {action: "revisions.quick_search_next", scope: "revisions.quick_search"},
	"quick_search_cycle_back": {action: "revisions.quick_search_prev", scope: "revisions.quick_search"},
	"suspend":                 {action: "ui.suspend", scope: "ui"},
	"set_parents":             {action: "revisions.set_parents", scope: "revisions"},

	"rebase.mode":         {action: "revisions.open_rebase", scope: "revisions"},
	"rebase.revision":     {action: "revisions.rebase.set_source", scope: "revisions.rebase", args: map[string]string{"source": "revision"}},
	"rebase.source":       {action: "revisions.rebase.set_source", scope: "revisions.rebase", args: map[string]string{"source": "source"}},
	"rebase.branch":       {action: "revisions.rebase.set_source", scope: "revisions.rebase", args: map[string]string{"source": "branch"}},
	"rebase.target":       {action: "revisions.rebase.target_picker", scope: "revisions.rebase"},
	"rebase.after":        {action: "revisions.rebase.set_target", scope: "revisions.rebase", args: map[string]string{"target": "after"}},
	"rebase.before":       {action: "revisions.rebase.set_target", scope: "revisions.rebase", args: map[string]string{"target": "before"}},
	"rebase.onto":         {action: "revisions.rebase.set_target", scope: "revisions.rebase", args: map[string]string{"target": "onto"}},
	"rebase.insert":       {action: "revisions.rebase.set_target", scope: "revisions.rebase", args: map[string]string{"target": "insert"}},
	"rebase.skip_emptied": {action: "revisions.rebase.skip_emptied", scope: "revisions.rebase"},

	"revert.mode":   {action: "revisions.open_revert", scope: "revisions"},
	"revert.target": {action: "revisions.revert.target_picker", scope: "revisions.revert"},
	"revert.after":  {action: "revisions.revert.set_target", scope: "revisions.revert", args: map[string]string{"target": "after"}},
	"revert.before": {action: "revisions.revert.set_target", scope: "revisions.revert", args: map[string]string{"target": "before"}},
	"revert.onto":   {action: "revisions.revert.set_target", scope: "revisions.revert", args: map[string]string{"target": "onto"}},
	"revert.insert": {action: "revisions.revert.set_target", scope: "revisions.revert", args: map[string]string{"target": "insert"}},

	"duplicate.mode":   {action: "revisions.open_duplicate", scope: "revisions"},
	"duplicate.target": {action: "revisions.duplicate.target_picker", scope: "revisions.duplicate"},
	"duplicate.after":  {action: "revisions.duplicate.set_target", scope: "revisions.duplicate", args: map[string]string{"target": "after"}},
	"duplicate.before": {action: "revisions.duplicate.set_target", scope: "revisions.duplicate", args: map[string]string{"target": "before"}},
	"duplicate.onto":   {action: "revisions.duplicate.set_target", scope: "revisions.duplicate", args: map[string]string{"target": "onto"}},

	"squash.mode":                    {action: "revisions.open_squash", scope: "revisions"},
	"squash.target":                  {action: "revisions.squash.target_picker", scope: "revisions.squash"},
	"squash.keep_emptied":            {action: "revisions.squash.keep_emptied", scope: "revisions.squash"},
	"squash.use_destination_message": {action: "revisions.squash.use_destination_msg", scope: "revisions.squash"},
	"squash.interactive":             {action: "revisions.squash.interactive", scope: "revisions.squash"},

	"details.mode":                    {action: "revisions.open_details", scope: "revisions"},
	"details.close":                   {action: "revisions.details.cancel", scope: "revisions.details"},
	"details.split":                   {action: "revisions.details.split", scope: "revisions.details"},
	"details.split_parallel":          {action: "revisions.details.split_parallel", scope: "revisions.details"},
	"details.squash":                  {action: "revisions.details.squash", scope: "revisions.details"},
	"details.restore":                 {action: "revisions.details.restore", scope: "revisions.details"},
	"details.absorb":                  {action: "revisions.details.absorb", scope: "revisions.details"},
	"details.diff":                    {action: "revisions.details.diff", scope: "revisions.details"},
	"details.select":                  {action: "revisions.details.toggle_select", scope: "revisions.details"},
	"details.revisions_changing_file": {action: "revisions.details.revisions_changing_file", scope: "revisions.details"},

	"evolog.mode":    {action: "revisions.open_evolog", scope: "revisions"},
	"evolog.diff":    {action: "revisions.evolog.diff", scope: "revisions.evolog"},
	"evolog.restore": {action: "revisions.evolog.restore", scope: "revisions.evolog"},

	"preview.mode":           {action: "ui.preview_toggle", scope: "revisions"},
	"preview.toggle_bottom":  {action: "ui.preview_toggle_bottom", scope: "revisions"},
	"preview.scroll_up":      {action: "ui.preview_scroll_up", scope: "ui.preview"},
	"preview.scroll_down":    {action: "ui.preview_scroll_down", scope: "ui.preview"},
	"preview.half_page_down": {action: "ui.preview_half_page_down", scope: "ui.preview"},
	"preview.half_page_up":   {action: "ui.preview_half_page_up", scope: "ui.preview"},
	"preview.expand":         {action: "ui.preview_expand", scope: "ui.preview"},
	"preview.shrink":         {action: "ui.preview_shrink", scope: "ui.preview"},

	"bookmark.mode":    {action: "ui.open_bookmarks", scope: "revisions"},
	"bookmark.set":     {action: "revisions.bookmark_set", scope: "revisions"},
	"bookmark.delete":  {action: "bookmarks.bookmark_delete", scope: "bookmarks"},
	"bookmark.move":    {action: "bookmarks.bookmark_move", scope: "bookmarks"},
	"bookmark.forget":  {action: "bookmarks.bookmark_forget", scope: "bookmarks"},
	"bookmark.track":   {action: "bookmarks.bookmark_track", scope: "bookmarks"},
	"bookmark.untrack": {action: "bookmarks.bookmark_untrack", scope: "bookmarks"},

	"inline_describe.mode":   {action: "revisions.inline_describe", scope: "revisions"},
	"inline_describe.accept": {action: "revisions.inline_describe.accept", scope: "revisions.inline_describe"},
	"inline_describe.editor": {action: "revisions.inline_describe.editor", scope: "revisions.inline_describe"},

	"git.mode":  {action: "ui.open_git", scope: "revisions"},
	"git.push":  {action: "git.push", scope: "git"},
	"git.fetch": {action: "git.fetch", scope: "git"},

	"oplog.mode":    {action: "ui.open_oplog", scope: "revisions"},
	"oplog.restore": {action: "oplog.restore", scope: "oplog"},
	"oplog.revert":  {action: "oplog.revert", scope: "oplog"},

	"file_search.toggle": {action: "file_search.toggle", scope: "file_search"},
	"file_search.up":     {action: "file_search.move_up", scope: "file_search"},
	"file_search.down":   {action: "file_search.move_down", scope: "file_search"},
	"file_search.accept": {action: "file_search.apply", scope: "file_search"},
	"file_search.edit":   {action: "file_search.edit", scope: "file_search"},

	"diff_view.scroll_up":      {action: "diff.scroll_up", scope: "diff"},
	"diff_view.scroll_down":    {action: "diff.scroll_down", scope: "diff"},
	"diff_view.page_up":        {action: "diff.page_up", scope: "diff"},
	"diff_view.page_down":      {action: "diff.page_down", scope: "diff"},
	"diff_view.half_page_up":   {action: "diff.half_page_up", scope: "diff"},
	"diff_view.half_page_down": {action: "diff.half_page_down", scope: "diff"},
	"diff_view.left":           {action: "diff.left", scope: "diff"},
	"diff_view.right":          {action: "diff.right", scope: "diff"},
	"diff_view.close":          {action: "ui.cancel", scope: "diff"},
}

var skippedRootKeys = map[string]bool{
	"custom_commands": true,
	"leader":          true,
	"apply":           true,
	"force_apply":     true,
}

func parseKeys(content string) (map[string]any, error) {
	var raw map[string]any
	if _, err := toml.Decode(content, &raw); err != nil {
		return nil, err
	}

	section, ok := raw["keys"]
	if !ok {
		return nil, nil
	}

	keys, ok := section.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("[keys] is not a table")
	}

	return keys, nil
}

type bindingResult struct {
	toml string
}

func convertKeys(keys map[string]any) ([]bindingResult, []string) {
	var bindings []bindingResult
	var warnings []string

	type entry struct {
		lookupKey string
		values    []string
	}
	var entries []entry

	topKeys := make([]string, 0, len(keys))
	for k := range keys {
		topKeys = append(topKeys, k)
	}
	sort.Strings(topKeys)

	for _, k := range topKeys {
		v := keys[k]
		if skippedRootKeys[k] {
			continue
		}
		switch val := v.(type) {
		case string:
			entries = append(entries, entry{lookupKey: k, values: []string{val}})
		case []any:
			entries = append(entries, entry{lookupKey: k, values: toStringSlice(val)})
		case map[string]any:
			subKeys := make([]string, 0, len(val))
			for sk := range val {
				subKeys = append(subKeys, sk)
			}
			sort.Strings(subKeys)
			for _, sk := range subKeys {
				sv := val[sk]
				switch svVal := sv.(type) {
				case string:
					entries = append(entries, entry{lookupKey: k + "." + sk, values: []string{svVal}})
				case []any:
					entries = append(entries, entry{lookupKey: k + "." + sk, values: toStringSlice(svVal)})
				default:
					warnings = append(warnings, fmt.Sprintf("keys.%s.%s: unsupported value type", k, sk))
				}
			}
		default:
			warnings = append(warnings, fmt.Sprintf("keys.%s: unsupported value type", k))
		}
	}

	for _, e := range entries {
		mapping, ok := keyMappings[e.lookupKey]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("keys.%s: unknown key, skipping", e.lookupKey))
			continue
		}

		var sb strings.Builder
		sb.WriteString("[[bindings]]\n")
		fmt.Fprintf(&sb, "action = %q\n", mapping.action)
		fmt.Fprintf(&sb, "key = %s\n", formatStringList(e.values))
		fmt.Fprintf(&sb, "scope = %q\n", mapping.scope)
		if len(mapping.args) > 0 {
			argKeys := make([]string, 0, len(mapping.args))
			for ak := range mapping.args {
				argKeys = append(argKeys, ak)
			}
			sort.Strings(argKeys)
			for _, ak := range argKeys {
				fmt.Fprintf(&sb, "args.%s = %q\n", ak, mapping.args[ak])
			}
		}
		bindings = append(bindings, bindingResult{toml: sb.String()})
	}

	return bindings, warnings
}

func removeLegacySections(content string) (string, error) {
	var raw map[string]any
	if _, err := toml.Decode(content, &raw); err != nil {
		return "", err
	}

	delete(raw, "custom_commands")
	delete(raw, "keys")
	delete(raw, "leader")

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(raw); err != nil {
		return "", err
	}
	return buf.String(), nil
}
