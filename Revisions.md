# Revisions

The Revisions view is the default interface when you start `jjui`. It provides a powerful and interactive way to explore your repository's history, inspect changes, and perform revision operations.

## Overview

The Revisions view is divided into three main sections:

### 1. Revset Bar
The top line displays the current revset, which determines the set of revisions shown in the log graph. You can change the revset by pressing `L`. Previously used revsets are kept in session history, and you can cycle through them using the `up` and `down` arrow keys.

If you start `jjui` with `--revset/-r`, that expression becomes the session’s default. When editing the revset, pressing `Enter` on an empty input resets to this default instead of clearing it.

### 2. Log Graph
The log graph visualizes your repository's commit history, generated from the output of the `jj log` command. The currently selected revision is highlighted using the configured "selected" background color. Navigate between revisions using `j` (down) and `k` (up). Press `@` to quickly jump back to the working copy revision.

### 3. Status Bar
The status bar provides contextual information about the current selection and available actions.

## Preview Pane

You can open the preview pane by pressing `p`. This panel displays detailed information about the selected revision, file, or operation, using commands defined in your [configuration](./Configuration.md). By default, the following commands are used:

```toml
[preview]
revision_command = ["show", "--color", "always", "-r", "$change_id"]
oplog_command = ["op", "show", "$operation_id", "--color", "always"]
file_command = ["diff", "--color", "always", "-r", "$change_id", "$file"]
```

While the preview pane is open, you can scroll using `ctrl+n` (down), `ctrl+p` (up), `ctrl+d` (half page down), and `ctrl+u` (half page up).

For more details, see the [Preview](./Preview.md) page.

## Quick Search

You can quickly search within the revisions view without leaving it:

- Press `/` to start quick search and type a term.
- Matches are highlighted case-insensitively across the visible log.
- Press `'` (single quote) to jump to the next match.
- Press `Enter` or `Esc` to clear the search and return to normal navigation.

## Revision Operations

From the Revisions view, you can perform a variety of operations:

- **Details View**: Press `l` to open the details view for the selected revision. Here, you can split, restore, and inspect files. See [Details](./Details.md).
- **Rebase**: Press `r` to enter rebase mode and move revisions, branches, or sources. See [Rebase](./Rebase.md).
- **Absorb**: Press `A` to run `jj absorb` on the highlighted revision. See [Absorb](./Absorb.md).
- **Abandon**: Press `a` to abandon selected revisions. See [Abandon](./Abandon.md).
- **Duplicate**: Press `y` to duplicate selected revisions. See [Duplicate](./Duplicate.md).
- **Squash**: Press `S` to squash selected revisions into one. See [Squash](./Squash.md).
- **Inline Describe**: Press `Enter` to edit the description of a revision in an inline editor. See [Inline Describe](./Inline-Describe.md).
- **Set Bookmark**: Press `B` to set or move a bookmark on the selected revision.
- **Bookmark Menu**: Press `b` to open the bookmark pop-up menu for managing bookmarks. See [Bookmarks](./Bookmarks.md).

## Mouse Interaction

When running in a terminal that supports mouse events, you can:

- Scroll the revisions list using the mouse wheel.
- Click on revisions to move the selection.
- Drag the preview divider to resize the preview pane.

## Customization

You can customize the appearance and behavior of the Revisions view using the configuration file. For example, you can set the number of revisions shown with the `--limit` flag or configure colors and key bindings. See [Configuration](./Configuration.md) and [Command Line Options](./Command-Line-Options.md) for more information.
