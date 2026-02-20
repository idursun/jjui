# Migrating to v0.10 (current trunk)

v0.10 replaces the legacy keybinding system with a unified **actions + bindings** architecture. All keyboard input now flows through a single pipeline:

**KeyMsg → Dispatcher → Binding → Action → Intent → Model**

This guide covers what changed and how to update your configuration.

---

## TL;DR

**If you have `[custom_commands]` or `[keys]` in your config, run:**

```sh
jjui --config --migrate
```

This converts them automatically. Then you're done.

**If you have `[leader]` sequences**, rewrite them as `seq` bindings:

```toml
[[bindings]]
action = "ui.open_revset"
seq = ["g", "r"]
scope = "revisions"
```

**If you want to add new custom actions**, use `[[actions]]` + `[[bindings]]` in `config.toml`, or `config.lua` for scripting:

```toml
[[actions]]
name = "my-action"
lua = 'flash("hello")'

[[bindings]]
action = "my-action"
key = "H"
scope = "revisions"
```

Read on for the full reference.

---

## What Was Removed

| Old | Replacement |
|-----|-------------|
| `[custom_commands]` | `[[actions]]` + `[[bindings]]` |
| `[leader]` + leader key sequences | `seq = [...]` in `[[bindings]]` |
| `[keys]` key overrides | `[[bindings]]` with matching `scope` |

**Startup warnings:** If your config still contains `[custom_commands]` or `[leader]`, jjui prints a warning to stderr but continues. `[keys]` is silently ignored. None of them crash the app, but they have no effect.

---

## Automated Migration

Run the migration command to convert `[custom_commands]` and `[keys]` automatically:

```sh
jjui --config --migrate
```

What it does:

1. Creates `config.old.toml` as a backup (only on first run).
2. Converts each `[custom_commands]` entry to an `[[actions]]` + `[[bindings]]` block.
3. Converts `[keys]` entries to matching `[[bindings]]` blocks.
4. Removes the legacy sections from `config.toml`.

**Limitations of migration:**
- `[leader]` entries are removed but **not** converted — rewrite them as `seq` bindings manually (see below).
- Commands with `show = "diff"` are skipped with a warning — the output-in-diff-viewer feature is not supported in the new system.

---

## Actions and Bindings

### Defining a custom action

Custom actions run Lua scripts. They are defined in `[[actions]]` blocks:

```toml
[[actions]]
name = "copy-diff"
lua = '''
local diff = jj("diff", "-r", context.change_id(), "--git")
copy_to_clipboard(diff)
'''
```

Required fields: `name` (string), `lua` (string).

### Binding a custom action to a key

```toml
[[bindings]]
action = "copy-diff"
key = "Y"
scope = "revisions"
desc = "copy diff to clipboard"
```

### Binding a built-in action

```toml
[[bindings]]
action = "ui.open_revset"
key = "L"
scope = "revisions"
desc = "revset"
```

### Binding fields

| Field | Required | Description |
|-------|----------|-------------|
| `action` | yes | Action ID (built-in or custom) |
| `scope` | yes | Where the binding is active |
| `key` | one of key/seq | Key or array of keys |
| `seq` | one of key/seq | Multi-key sequence (min 2 keys) |
| `desc` | no | Label shown in the status bar help |
| `args` | no | Arguments passed to built-in actions |

`key` and `seq` accept a single string or an array:

```toml
key = "r"
key = ["up", "k"]   # both keys trigger the same action
```

### Scope reference

The full list of scopes and available built-in actions is in:

```
internal/config/default/bindings.toml
```

Common scopes: `revisions`, `revisions.rebase`, `revisions.squash`, `revisions.details`, `oplog`, `ui`, `diff`.

---

## Replacing `[keys]` (key rebinding)

The old `[keys]` table mapped action names to keys. Replace each entry with a `[[bindings]]` block referencing the equivalent built-in action.

**Before:**

```toml
[keys]
abandon = "x"
```

**After:**

```toml
[[bindings]]
action = "revisions.abandon"
key = "x"
scope = "revisions"
```

---

## Replacing `[leader]` (key sequences)

Leader key sequences become `seq` bindings.

**Before:**

```toml
[leader.gr]
help = "Open revset"
send = ["L"]
```

**After:**

```toml
[[bindings]]
action = "ui.open_revset"
seq = ["g", "r"]
scope = "revisions"
desc = "open revset"
```

The `seq` field takes an ordered array of keys. When the user presses the first key, jjui enters a pending state and waits for subsequent keys.

---

## Replacing `[custom_commands]`

### Simple jj command

**Before:**

```toml
[custom_commands."show diff"]
key = ["U"]
args = ["diff", "-r", "$change_id", "--git"]
```

**After:**

```toml
[[actions]]
name = "show-diff"
lua = '''
jj_async("diff", "-r", context.change_id(), "--git")
'''

[[bindings]]
action = "show-diff"
key = "U"
scope = "revisions"
desc = "show diff"
```

### Interactive command

**Before:**

```toml
[custom_commands."resolve vscode"]
key = ["R"]
args = ["resolve", "--tool", "vscode"]
show = "interactive"
```

**After:**

```toml
[[actions]]
name = "resolve-vscode"
lua = '''
jj_interactive("resolve", "--tool", "vscode")
'''

[[bindings]]
action = "resolve-vscode"
key = "R"
scope = "revisions"
desc = "resolve in vscode"
```

### Revset command

**Before:**

```toml
[custom_commands."show descendants"]
key = ["M"]
revset = "::$change_id"
```

**After:**

```toml
[[actions]]
name = "show-descendants"
lua = '''
revset.set("::" .. context.change_id())
'''

[[bindings]]
action = "show-descendants"
key = "M"
scope = "revisions"
desc = "show descendants"
```

### Lua custom command (unchanged)

Lua scripts in `[custom_commands]` migrate directly — the same Lua API is available.

**Before:**

```toml
[custom_commands."set-revset"]
key = ["+"]
lua = '''
revset.set("bookmarks()")
'''
```

**After:**

```toml
[[actions]]
name = "set-revset"
lua = '''
revset.set("bookmarks()")
'''

[[bindings]]
action = "set-revset"
key = "+"
scope = "revisions"
desc = "set revset"
```

---

## `config.lua` — Programmatic Setup

As an alternative to TOML, you can register actions and bindings from Lua in `config.lua`. Both files live in the same directory as `config.toml`:

- `~/.config/jjui/config.toml`
- `~/.config/jjui/config.lua`

At startup, jjui loads `config.lua` and calls `setup(config)` if it is defined. The `config` parameter exposes two helpers:

### `config.action(name, fn, opts?)`

Registers a Lua action, optionally with an inline binding.

```lua
function setup(config)
  config.action("copy-diff", function()
    local diff = jj("diff", "-r", context.change_id(), "--git")
    copy_to_clipboard(diff)
  end, {
    key = "Y",
    scope = "revisions",
    desc = "copy diff",
  })
end
```

`opts` fields:

| Field | Description |
|-------|-------------|
| `key` | Key or array of keys (mutually exclusive with `seq`) |
| `seq` | Sequence of keys (mutually exclusive with `key`) |
| `scope` | Required when `key` or `seq` is set |
| `desc` | Optional description for the status bar |

### `config.bind({...})`

Adds a binding for an action defined elsewhere (TOML or another `config.action` call).

```lua
function setup(config)
  config.bind({
    action = "ui.open_revset",
    key = "R",
    scope = "revisions",
    desc = "revset",
  })
end
```

> **Note:** `args` is not supported in `config.bind`. Use built-in action bindings with args in `config.toml` instead.

---

## Plugins via `require`

`config.lua` can load modules from the config directory using `require`. The search paths are:

- `<config_dir>/?.lua`

**`~/.config/jjui/plugins/my_plugin.lua`:**

```lua
local M = {}

function M.setup(config)
  config.action("my-action", function()
    flash("hello from plugin")
  end, {
    key = "H",
    scope = "revisions",
    desc = "my action",
  })
end

return M
```

**`~/.config/jjui/config.lua`:**

```lua
local my_plugin = require("plugins.my_plugin")

function setup(config)
  my_plugin.setup(config)
end
```

---

## Sharing Lua Helpers Between TOML and `config.lua`

`config.lua` and `[[actions]].lua` scripts run in the same Lua VM. Global functions defined in `config.lua` are available to TOML action scripts.

**`config.lua`:**

```lua
function format_diff(change_id)
  local out, err = jj("diff", "-r", change_id, "--git")
  if err then return nil, err end
  return out, nil
end
```

**`config.toml`:**

```toml
[[actions]]
name = "copy-diff"
lua = '''
local diff, err = format_diff(context.change_id())
if err then
  flash({ text = err, error = true })
else
  copy_to_clipboard(diff)
end
'''
```

---

## Lua API Reference

All APIs are available as top-level globals and also under `jjui.*`.

### Command helpers

| Function | Description |
|----------|-------------|
| `jj(...)` | Run jj command immediately, returns `(output, err)` |
| `jj_async(...)` | Run jj command asynchronously |
| `jj_interactive(...)` | Run interactive jj command (like `diffedit`) |
| `exec_shell(command)` | Run a shell command |

### UI helpers

| Function | Description |
|----------|-------------|
| `flash("text")` | Show a flash message |
| `flash({ text, error, sticky })` | Show a flash message with options |
| `copy_to_clipboard(text)` | Copy text to clipboard, returns `(ok, err)` |
| `split_lines(text, keep_empty?)` | Split a string by newlines into an array |
| `choose(options)` | Show a selection menu, returns the chosen string or `nil` |
| `choose({ options, title })` | Show a selection menu with a title |
| `input({ title, prompt })` | Show a text input, returns the entered string or `nil` |

### Context helpers

Available as `context.*`:

| Function | Description |
|----------|-------------|
| `context.change_id()` | Currently selected change ID |
| `context.commit_id()` | Currently selected commit ID |
| `context.file()` | Currently selected file (in details view) |
| `context.operation_id()` | Currently selected operation ID (in oplog) |
| `context.checked_files()` | Array of checked file paths |
| `context.checked_change_ids()` | Array of checked change IDs |
| `context.checked_commit_ids()` | Array of checked commit IDs |

### Revisions helpers

Available as `revisions.*`:

| Function | Description |
|----------|-------------|
| `revisions.current()` | Current change ID |
| `revisions.checked()` | Array of checked change IDs |
| `revisions.refresh({ keep_selections, selected_revision })` | Refresh the revision list |
| `revisions.navigate({ by, page, target, to, fallback, ensureView, allowStream })` | Move cursor |
| `revisions.start_squash({ files })` | Start squash operation |
| `revisions.start_rebase({ source, target })` | Start rebase operation |
| `revisions.open_details()` | Open details view |
| `revisions.start_inline_describe()` | Start inline describe |

### Revset helpers

Available as `revset.*`:

| Function | Description |
|----------|-------------|
| `revset.set(value)` | Set the active revset |
| `revset.reset()` | Reset to the default revset |
| `revset.current()` | Return the current revset string |
| `revset.default()` | Return the default revset string |

---

## Alternate Base Bindings

`bindings_profile` lets you replace the built-in default bindings entirely. Set it to a path (relative to config dir or absolute) pointing to a bindings TOML file:

```toml
bindings_profile = "vim_bindings.toml"
```

Your `[[bindings]]` overlays are then applied on top of the profile instead of the built-in defaults. Use `:builtin` to explicitly restore the built-in defaults.
