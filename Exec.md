# Exec Feature

The `jj exec` feature in `jjui` brings command execution capabilities inspired by Vim's `:` key. It allows users to run both Jujutsu (`jj`) commands and arbitrary shell commands directly from the UI, using the status bar line-editor for input.

## Overview

- **Footer Line-Editor:** Commands are entered in the status bar, similar to how searches are performed with `/`.
- **Context-Aware Placeholders:** You can use placeholders like `$file`, `$checked_files`, `$change_id`, `$operation_id`, `$commit_id`, and `$checked_commit_ids` in your commands. These will be replaced with the relevant context value, just as in custom commands.

## Running Commands

### `:` Key — Jujutsu Interactive Commands

Pressing `:` opens the line-editor for entering `jj` commands. These commands are run interactively, allowing `jj` to use the terminal for its own TUI if needed.

**Examples:**
- `: help` — Show help for jj
- `: restore -i` — Run jj restore in interactive mode
- `: squash -i` — Run jj squash interactively

### `$` Key — Shell Commands

Pressing `$` opens the line-editor for entering shell commands. These are run via `$SHELL -c "<input>"`, giving you full control of stdio and allowing you to see all output and errors as you would in a normal terminal.

**Examples:**
- `$ man jj` — Open the manual for jj
- `$ jq . $file | bat --paging always` — Process the current file with jq and display with bat
- `$ htop` — Run htop interactively

## Interactive Behavior

- Programs are run interactively with full control of stdio.
- The main UI does not show a status spinner or error notification for these commands.
- If a program terminates quickly (less than 5 seconds), it is assumed to be non-interactive (e.g., `: version` or `$ ls -la`). In these cases, jjui will ask for confirmation before closing the terminal and returning to the main UI.

## Log Batching in Revisions

Log batching is now enabled by default in jjui. Instead of loading all revisions at startup, jjui will load the first 50 revisions and fetch additional chunks as you scroll past the 50th revision. This greatly improves startup times for large repositories, while having minimal impact on small ones.

Previously, jjui loaded the entire log before displaying revisions. With batching, you get a faster and more responsive experience.

You can control this feature with the following configuration value:

```toml
[revisions]
log_batching = false
```

Set `log_batching` to `false` if you want to revert to the old behavior and load all revisions at once.

Note: This feature was previously experimental and controlled by `experimental_log_batching_enabled`. It is now enabled by default and the configuration location has changed as shown above.