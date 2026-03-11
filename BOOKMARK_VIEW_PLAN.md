# Bookmark Side Pane Plan

## Goal

Add a non-modal interactive bookmark view to `jjui` as a right-hand side pane.

The intended workflow is:

1. User presses `X` (`shift+x` in bindings).
2. A bookmark pane opens on the right side of the screen.
3. Focus moves into the bookmark pane.
4. User navigates bookmarks and runs bookmark-oriented actions directly from the pane.
5. User presses `X` again to close the pane.
6. User can press `Tab` to switch focus between the revisions pane and the bookmark pane while both remain visible.

This is intentionally different from the existing modal bookmark operations menu. The new feature should be bookmark-centered, not command-centered.

## Product Shape

### What We Are Building

Build a side pane that:

- keeps the current revisions view visible on the left
- shows a bookmark list on the right
- lets the bookmark pane own keyboard focus
- supports direct actions on the selected bookmark
- can be toggled on and off without entering a modal dialog

### What We Are Not Changing Initially

- Keep the existing `b` bookmark overlay intact.
- Keep the existing `B` set-bookmark flow intact.
- Do not replace current bookmark management behavior in the first version.

This keeps risk low and makes the new side pane easy to test independently.

## UX Proposal

### Open/Close

- `X` toggles the bookmark pane.
- Opening the pane moves focus into it.
- Closing the pane returns focus to revisions.

### Focus

- `Tab` switches focus between revisions and bookmarks when the pane is open.
- The focused pane owns navigation keys.
- The unfocused pane remains visible but passive.

### Layout

- Left side: existing revisions view
- Right side: bookmark pane
- Target width: about 45% to 50% of the content area for initial testing

For v1, the bookmark pane and preview pane should be mutually exclusive.

Reason:

- `jjui` already has a single split-based secondary pane architecture.
- Reusing that architecture is simpler than introducing nested side panes.
- Preview and bookmark pane can coexist later if the feature proves valuable.

Suggested v1 behavior:

- If preview is visible and user opens bookmarks, hide preview.
- Optionally remember preview visibility and restore it when the bookmark pane closes.

### Bookmark Pane Contents

The bookmark pane should be row-oriented:

- one row per logical bookmark
- local/remote/tracked/conflict state shown as badges
- optional short target summary if it fits
- filter field at the top
- key help/footer at the bottom

This should feel closer to `jj bookmark list` than to the current bookmark operations menu.

### Initial Selection

When the pane opens:

- if the selected revision has one or more bookmarks, preselect one of them
- otherwise preserve the last pane selection if possible
- otherwise select the first row

This blends the global bookmark-browser workflow with the current revision-focused workflow.

## Keymap Proposal

These are the initial bindings for the bookmark pane scope.

- `j` / `k`: move selection down/up
- `J` / `K`: half-page or page movement
- `/`: open filter
- `enter`: reveal bookmark in revisions if already visible
- `n`: `jj new <bookmark>`
- `e`: `jj edit <bookmark>`
- `r`: rename bookmark
- `d`: delete bookmark
- `f`: forget bookmark
- `t`: track bookmark
- `u`: untrack bookmark
- `m`: move bookmark
- `tab`: switch focus to revisions
- `X`: close pane
- `esc`: clear filter first, then close pane if not filtering

Notes:

- `X` in config will be `shift+x`.
- `J` and `K` will be `shift+j` and `shift+k`.
- For consistency with the rest of `jjui`, we can later decide whether lowercase `j/k` plus page bindings are sufficient and whether uppercase movement is worth keeping.

## Action Semantics

### Direct Actions

The side pane should run actions directly on the selected bookmark rather than opening a second command list.

Primary actions:

- `n`: create new revision from bookmark
- `e`: edit bookmark target
- `r`: rename
- `d`: delete
- `f`: forget
- `t`: track
- `u`: untrack
- `m`: move

### Enter Behavior

For v1, `enter` should be conservative:

- if the selected bookmark target is already visible in the current revisions list, move the left-side revision selection to it
- otherwise show a flash message instead of changing the revset automatically

Reason:

- automatically changing the revset is powerful but can be surprising
- users in the issue discussion explicitly disliked workflows that force them to restore the revset

We can add an explicit "show in log" action later if desired.

## Architecture Proposal

## 1. Add A Dedicated Bookmark Pane Model

Create a new model package, for example:

- `internal/ui/bookmarkpane/`

This model should be separate from:

- `internal/ui/bookmarks/` which is the current modal operations overlay

The new pane model should be:

- an immediate-mode renderer
- list-based
- independently focusable
- able to expose visibility/focus state to the root UI

Suggested responsibilities:

- fetch and store bookmark rows
- manage selection
- manage filtering
- render the right pane
- execute bookmark-related commands

## 2. Add Root-Level Pane Focus State

`ui.Model` currently assumes a single primary interaction surface plus modal overlays and special editors.

Add a small explicit focus enum in the root model, something like:

- revisions pane focused
- bookmark pane focused

The root should use that state to:

- decide which scope is primary
- decide which visible pane receives unmatched input
- decide what `Tab` does when the pane is open

This will require extending the logic around:

- primary scope selection
- always-on scopes
- unmatched key routing

## 3. Reuse The Existing Split Infrastructure

`jjui` already has split rendering and drag support.

Use the same split model for the bookmark pane rather than introducing a separate layout system.

Recommended v1 approach:

- keep a single secondary pane slot
- secondary pane is either preview or bookmark pane
- bookmark pane uses a horizontal split with revisions on the left and bookmarks on the right

This should minimize changes to rendering and resize behavior.

## 4. Add A New Scope And Intents

Do not overload the existing `bookmarks` scope used by the modal overlay.

Add a distinct scope for the side pane, for example:

- `bookmark_view`

Add root UI intents for:

- toggle bookmark pane
- focus next pane

Add bookmark-pane intents for:

- navigate
- page navigate
- open filter
- apply selected action
- new from bookmark
- edit bookmark
- rename
- delete
- forget
- track
- untrack
- move
- close

This keeps the command surface clean and avoids collisions with the current modal bookmark view.

## 5. Keep Existing Bookmark Features Intact

Do not remove or merge these in v1:

- modal bookmark overlay
- set-bookmark operation
- target picker

The new side pane should be additive.

## Data Model Proposal

The side pane should be built on top of bookmark rows, not command rows.

Each row should represent one logical bookmark and contain:

- display name
- canonical command target, such as `name` or `name@remote`
- whether a local bookmark exists
- remote entries and tracking state
- conflict state
- target commit id
- optional change id and one-line description if we enrich the loader

### Why Not Reuse The Existing Modal Bookmark Model

The current `internal/ui/bookmarks` model is built around precomputed actions such as:

- move `<bookmark>` to `<revision>`
- delete `<bookmark>`
- track `<bookmark>@<remote>`

That is the wrong shape for a bookmark browser because:

- it renders commands, not bookmarks
- it is tied to a selected revision
- it mixes browsing and action selection into one list

The new side pane should keep the browsing layer separate from the actions.

## Command Surface Changes

The current JJ command helpers already cover:

- set
- move
- delete
- forget
- track
- untrack

The side pane will likely also need wrappers for:

- rename
- possibly create
- possibly explicit helpers for `edit` and `new` if those are not already easily reusable

The plan should assume that command-surface additions are part of the implementation.

## Suggested Implementation Phases

## Phase 1: Skeleton Pane

- add root toggle intent
- add `shift+x` binding
- add bookmark pane model with placeholder rendering
- render it on the right using the existing split
- add visibility and focus state
- add `Tab` switching between revisions and bookmark pane

Deliverable:

- pane opens/closes
- pane takes focus
- focus can move back and forth

## Phase 2: Bookmark List

- load bookmarks into bookmark rows
- render rows in the pane
- implement selection and scrolling
- implement initial selection based on current revision bookmarks

Deliverable:

- usable bookmark browser without actions

## Phase 3: Filtering

- add `/` filter input
- support matching by bookmark name and possibly remote name
- add clear/cancel behavior

Deliverable:

- bookmark pane can be used efficiently in large repos

## Phase 4: Direct Actions

- `n`, `e`, `r`, `d`, `f`, `t`, `u`, `m`
- refresh both revisions and bookmark rows after mutations
- show flash messages for success/failure where appropriate

Deliverable:

- side pane becomes useful for daily workflows

## Phase 5: Reveal / Left-Pane Sync

- implement conservative `enter` behavior
- if target revision is visible, select it on the left
- otherwise flash a message

Deliverable:

- panes feel connected without surprising revset changes

## Phase 6: Polish

- footer help text
- better badges and row density
- remember previous selection when reopening
- optionally restore preview visibility after closing bookmarks

## Testing Plan

At minimum, add tests for:

- toggle open/close behavior
- focus switching with `Tab`
- routing of key input when bookmark pane is focused
- bookmark list loading and selection
- filter behavior
- direct action command invocation
- refresh behavior after mutations
- preview interaction when opening/closing bookmark pane

Where possible, follow the existing UI tests around:

- stacked models
- target picker
- bookmark overlay behavior

## Open Questions

These do not need to block v1, but should be decided before polishing:

- Should `enter` eventually reveal, edit, or open an action menu?
- Should bookmark rows show only local bookmarks by default, or all local + remote bookmarks?
- Should sorting be alphabetical, recency-based, or user-configurable?
- Should conflict state get a dedicated visual treatment?
- Should opening the pane automatically pre-filter to bookmarks on the selected revision when any exist?
- Should preview state be restored after closing the bookmark pane?

## Recommended v1 Decisions

To keep the first implementation tight:

- use a dedicated right-hand side pane
- make preview and bookmark pane mutually exclusive
- use direct action hotkeys
- keep `enter` conservative
- keep existing modal bookmark features untouched
- add new scope and intents instead of reusing the modal bookmark scope

That should produce a feature that is coherent, testable, and close to the desired workflow without creating large architectural churn.
