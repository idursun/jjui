## Summary

Preview commands in `jjui` currently run through the immediate command path, which captures stdout via `exec.Command(...).Output()`.

That subprocess does not run in a PTY sized to the preview pane, so width-sensitive tools such as `jj diff` only see the outer terminal width (or a default width), not the preview pane width.

This is visible with `jj` configured to use `difft` as the diff formatter: the preview opens with wrapping/layout based on the wrong width.

This change sets `COLUMNS` and `LINES` for preview subprocesses from the current preview pane size before running `jj`.

## Why this helps

`jj` already forwards terminal width to external diff tools like `difft`, but it can only forward the width it sees for its own process. In preview mode, that width is not the pane width.

By setting `COLUMNS`/`LINES` in `jjui`, `jj diff` and other terminal-width-aware preview commands can size themselves to the preview pane without requiring users to manually thread `$preview_width` through their config.

## Scope

- Adds `RunCommandImmediateWithEnv(...)` to the immediate command runner
- Uses it only for preview refreshes
- Leaves existing immediate command behavior unchanged elsewhere

## Testing

- Reproduced with `jjui` preview and `jj` configured with `ui.diff-formatter = "difft"`
- Verified that preview width matched the preview pane after this patch
