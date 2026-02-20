Custom commands can be defined in the `custom_commands` section of the configuration:

```toml
[custom_commands]
"show diff" = { key = ["U"], args = ["diff", "-r", "$change_id", "--color", "always"], show = "diff" }
"show oplog diff" = { key = ["ctrl+o"], args = ["op", "show",  "$operation_id", "--color", "always"], show = "diff" }
"resolve vscode" = { key = ["R"], args = ["resolve", "--tool", "vscode"], show = "interactive" }
"new main" = { args = ["new", "main"] }
"tug" = { key = ["ctrl+t"], args = ["bookmark", "move", "--from", "closest_bookmark($change_id)", "--to", "closest_pushable($change_id)"] }
```

Custom commands can have placeholder arguments like `$change_id`, `$operation_id`, `$file`, and `$revset`.

You can also use the following placeholders for more advanced context-aware commands:
- `$commit_id`: Refers to the currently selected commit ID.
- `$checked_commit_ids`: Refers to all checked (selected) commit IDs.
- `$checked_files`: Refers to all checked (selected) files.

Custom commands can also change the revset. For example, the following custom command will filter the view to only show descendants of the selected revision:

```toml
[custom_commands]
"show after revisions" = { key = ["M"], revset = "::$change_id" }
```

There is also a `show` argument which you can set it to be: 
* `none` (default); command will run as is and will only be displayed in the status bar.
* `diff`: the output of the command will be displayed in the diff viewer.
* `interactive`: the command run in interactive mode similar to `diffedit`, `split`, `commit` etc.

Custom commands menu can be opened by pressing `x` key. Each custom command can have a dedicated optional custom key binding which you can use to invoke it without having to open the custom commands menu.

Custom command window is context aware so it won't display the commands that have place holder but not applicable to the selected item.

<img width="1047" height="808" alt="image" src="https://github.com/user-attachments/assets/9039003f-6a74-446d-95c9-ab23dca1ebdf" />

## Key bindings and key sequences

Each custom command can be bound either to a simple key (via the `key` field) or to a multi-key sequence (via the `key_sequence` field).

```toml
[custom_commands]
"bookmark list" = { key_sequence = ["w", "b", "l"], desc = "bookmarks list", lua = '''
    revset.set("bookmarks() | remote_bookmarks()")
'''
```

- `key`: single or chorded keybindings (e.g. `["ctrl+t"]`).
- `key_sequence`: a sequence of keys pressed one after another (e.g. `["g", "s"]`).
- `desc`: optional human-friendly label shown in the custom command menu and in the key sequence overlay.

When you press the first key of any defined `key_sequence`, jjui shows a small overlay listing matching sequences and their descriptions. If you complete a sequence, the corresponding command runs; if you pause for a few seconds or press a non-matching key, the overlay is dismissed.

## Lua custom commands (experimental)

In addition to `args` or `revset`, custom commands can execute embedded Lua scripts using the `lua` field.

```toml
[custom_commands.append_to_revset]
    key = ["+"]
    lua = '''
    local change_id = revisions.current()
    if not change_id then return end

    local current = revset.current()
    local bumped = false
    local updated = current:gsub("ancestors%(" .. change_id .. "%s*,%s*(%d+)%)", function(n)
        bumped = true
        return "ancestors(" .. change_id .. ", " .. (tonumber(n) + 1) .. ")"
    end, 1)

    if not bumped then
        updated = current .. " | ancestors(" .. change_id .. ", 2)"
    end

    revset.set(updated)
    '''
```

Lua scripts run inside jjui and can access a small API:

- `revisions.current()` → current change ID (or `nil`).
- `revisions.checked()` → list of checked change IDs.
- `revisions.refresh{ keep_selections?, selected_revision? }` → refresh revisions.
- `revisions.navigate{ by?, page?, target?, to?, fallback?, ensureView?, allowStream? }` → move the revision cursor.
- `revisions.start_squash{ files? }`, `revisions.start_rebase{ source?, target? }`, `revisions.open_details()`, `revisions.start_inline_describe()`.
- `revset.set(value)`, `revset.reset()`, `revset.current()`, `revset.default()`.
- `jj{...}` → run a `jj` command and get `(output, err)`.
- `jj_async{...}` → run a `jj` command asynchronously.
- `flash(message)` → show a transient flash message.
- `copy_to_clipboard(text)` → copy text to the system clipboard.

This feature is experimental and the API may change between releases.

### Example custom commands from the community

#### Move commit up and down

```toml
[custom_commands."move commit down"]
key = ["J"]
args = ["rebase", "-r", "$change_id", "--insert-before", "$change_id-"]

[custom_commands."move commit up"]
key = ["K"]
args = ["rebase", "-r", "$change_id", "--insert-after", "$change_id+"]
```

#### Toggle selected revision as parent to the working copy

```toml
"toggle parent" = { key = ["ctrl+p"], args = ["rebase", "-r", "@", "-d", "all:(parents(@) | $change_id) ~ (parents(@) & $change_id)"] }
```

#### New `N`ote commit (insert an empty commit inline after `@`. Idea from [#278](https://github.com/idursun/jjui/issues/278))
```toml
[custom_commands."new note commit"]
key = ["N"]
args = ["new", "--no-edit", "-A", "$change_id"]
```
- Hit `<ret>` to immediately begin editing its description message inline! (`inline describe`)
- This is great for keeping TODO notes as chains of empty commits

### Loading jj aliases as custom_commands. (idea from [#211](https://github.com/idursun/jjui/issues/211))

You can use the following to import all your aliases as custom commands.

```bash
echo "[custom_commands]" >> $JJUI_CONFIG_DIR/config.toml
jj config list aliases -T '"\"" ++ name ++ "\" = { args = [ \"" ++ name ++ "\" ] }\n"' --no-pager |\
    sed -e 's/aliases.//g' |\
    tee -a $JJUI_CONFIG_DIR/config.toml
```