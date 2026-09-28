# Architecture

This document describes how `jjui` is structured today.

## Overview

`jjui` is built on `bubbletea/v2`, but the UI is not organized as a classic string-returning Bubble Tea app.

The current architecture combines:

- Bubble Tea for the event loop, message passing, and process lifecycle
- Immediate-mode rendering for most of the UI
- A generated action and intent catalog that decouples key bindings from model behavior
- A root UI model in [`internal/ui/ui.go`](internal/ui/ui.go) that owns composition, routing, and focus decisions

At a high level, the runtime flow is:

`cmd/jjui/main.go` -> `ui.New(...)` -> Bubble Tea event loop -> root UI routing -> immediate rendering -> cached frame output

## Entry Points

There are two important entry points:

- Process entry point: [`cmd/jjui/main.go`](cmd/jjui/main.go)
- UI entry point: [`internal/ui/ui.go`](internal/ui/ui.go)

`main.go` handles startup concerns such as config loading, theme setup, Lua VM initialization, and Bubble Tea program creation.

`ui.go` is the entry point to the application UI. It constructs the root model, initializes the dispatcher and resolver, owns the top-level view tree, and performs key binding to intent resolution.

`ui.New(...)` returns a small Bubble Tea wrapper around the real UI model. That wrapper exists to throttle rendering and cache frames.

## Rendering Model

`jjui` uses immediate view rendering.

Instead of having each model primarily build and return a string, most visible UI is rendered into a shared display context from [`internal/ui/render`](internal/ui/render).

Core pieces:

- [`internal/ui/render/display_context.go`](internal/ui/render/display_context.go) accumulates draw operations, effects, and mouse interactions for a frame
- `ViewRect(...)` methods render directly into the shared display context
- The root UI model creates a new `DisplayContext` every time it recomputes a frame
- After child models finish drawing, the display context renders into an ultraviolet screen buffer, which is then turned into the final terminal string

This means rendering is compositional and layout-driven:

1. `ui.go` chooses the active layout
2. Child models receive layout boxes
3. Each child draws primitives into the display context
4. The root model renders the accumulated operations into the terminal buffer

The `render/` package contains the primitives that make this work:

- draw operations
- effects such as dim/highlight/fill
- text building helpers
- list rendering helpers
- interaction registration for mouse handling
- z-index ordering

## Frame Scheduling and Caching

The root UI is wrapped by a small model in [`internal/ui/ui.go`](internal/ui/ui.go) that caches the last rendered frame.

The wrapper does two things:

- it lets `Update(...)` process messages immediately
- it only recomputes `View()` on an 8ms tick

In practice, `jjui` pushes at most one new frame every 8ms while continuing to process as many messages as Bubble Tea delivers in between those ticks. The last rendered frame is cached and reused until the next scheduled render.

This reduces redundant redraw work while keeping the message loop responsive.

## Input Architecture

The input pipeline is intentionally split into separate layers:

`key` -> `binding` -> `action` -> `intent` -> `model handler`

This separation is important. Models do not own raw key bindings. Models handle intents.

### Bindings and Actions

Bindings are configured as scoped runtime bindings. The dispatcher in [`internal/ui/dispatch/dispatcher.go`](internal/ui/dispatch/dispatcher.go) resolves key presses against the active scope chain.

The dispatcher supports:

- single-key bindings
- multi-key sequences
- scope precedence from innermost to outermost

### Intents

Intents are the application-level actions that models handle. The base interface lives in [`internal/ui/intents/intent.go`](internal/ui/intents/intent.go).

The architectural rule is:

- bindings decide how a capability is invoked
- intents describe what capability should happen
- models implement the behavior for those intents

This keeps feature behavior independent from the specific keys or scripts that trigger it.

### Generated Catalog

Intent types are annotated with `//jjui:bind` directives in [`internal/ui/intents`](internal/ui/intents).

Those annotations are used by [`cmd/genactions`](cmd/genactions) to generate:

- the internal action-to-intent lookup in [`internal/ui/actions`](internal/ui/actions)
- builtin action metadata in [`internal/ui/actionmeta`](internal/ui/actionmeta)
- the builtin Lua action surface exposed under `jjui.builtin.*`

The generated catalog is the bridge between declarative action identifiers and concrete intent values.

### Resolver

The resolver in [`internal/ui/dispatch/resolver.go`](internal/ui/dispatch/resolver.go) extends dispatch from bindings to actual behavior.

Resolution order is:

1. active operation override
2. configured Lua action override
3. generated builtin action catalog

Once an intent is resolved, the root UI routes it to the owning model.

## View Layers and Scope Routing

There is no generic focus-tree subsystem. The root model keeps its views in one ordered list, and that list drives rendering, key routing, closing and message delivery.

### The layer list

The revisions view is always at the bottom. Views opened over it are kept as layers, ordered bottom to top, in [`internal/ui/layers.go`](internal/ui/layers.go):

- a **screen** (oplog, diff, annotation) takes the whole view and hides the layers below it
- a **dialog** (git, bookmarks, undo, redo, help, choose, input, target picker, command history) is drawn over the layers below it

The top layer is the active one:

- rendering draws the topmost screen (or the revisions view), then every dialog above it
- only the top layer receives keys
- closing (`CloseViewMsg`, or the cancel fallback) removes the top layer
- non-input messages are broadcast to every layer, so hidden layers stay up to date

Opening a dialog replaces the dialog on top. Opening a screen moves it to the top, keeping at most one screen of each kind.

The **primary** view is the one shown in the split beside the preview or bookmark pane: the oplog when it is open, otherwise revisions. Unhandled keys go to the primary view, and the split panes only join key routing while the primary view is the top layer.

The password prompt, flash messages and the status line sit outside the list and are handled around it.

This rendering order is separate from `revisions.Model`'s own layers stack, which holds the active revisions operation and its transient overlays.

### Scope chain

For each key, `dispatchScopes()` in [`internal/ui/ui.go`](internal/ui/ui.go) builds a chain of scopes from innermost to outermost:

1. the password prompt, if any
2. the status line, while it takes input (exec, quick search, file search)
3. the revset editor, while editing
4. the top layer's scopes, or the primary view's scopes followed by the split pane's scopes
5. the revset scope, when not editing
6. the root `ui` scope

Each scope's leak policy decides how far routing continues past it:

- `LeakAll`: every outer scope stays visible
- `LeakGlobal`: only outer scopes marked `Global` stay visible
- `LeakNone`: routing stops here, typically while editing text

Two scopes are `Global`: the root `ui` scope, which is available everywhere, and the preview. The preview cannot take focus, so being `Global` lets its hotkeys work through the primary view's own `LeakGlobal` modes such as details and evolog. It is only in the chain while the primary view is on top.

## Root UI Responsibilities

The root model in [`internal/ui/ui.go`](internal/ui/ui.go) is responsible for more than just top-level layout.

It currently owns:

- the view layer list: revisions, oplog, diff, annotation, and dialogs
- dispatch scope selection from that list
- action and intent routing
- top-level lifecycle actions like quit, help, undo, redo, preview toggling, and overlays
- mouse interaction handoff through the current display context
- split layout state for the preview pane

This file is the architectural center of the UI.

## Mouse and Interaction Handling

Mouse handling follows the same immediate rendering model.

During rendering, components register clickable, scrollable, or draggable regions with the display context. When Bubble Tea delivers a mouse event, the root model forwards it to the active `DisplayContext`, which resolves the topmost matching interaction and optionally emits a new Bubble Tea message.

A region either carries a fixed message (`AddInteraction`) or computes its message from the mouse event (`AddInteractionFn`), for example a scroll delta via `render.WheelDelta` or a drag start position.

This means mouse interaction targets are derived from the current frame rather than kept as long-lived widgets.

## Lua Integration

Lua is integrated as another way to invoke actions, not as a separate UI system.

Configured actions may resolve to Lua scripts, and generated builtin actions are also exposed to Lua. That keeps Lua in the same action/intention architecture instead of creating a parallel command model.

The relevant runtime pieces are:

- [`internal/scripting/lua.go`](internal/scripting/lua.go)
- [`internal/ui/actionmeta`](internal/ui/actionmeta)
- [`internal/ui/actions`](internal/ui/actions)

Lua getters for live model state use the root's `ApplicationStateProvider`.
Typed selection remains available through the existing `SelectionProvider`,
while model-specific getters implement explicit `StateProvider` properties.
The root resolves public namespace paths by asking retained owners, including
hidden split content and revision operation layers; focus and display state do
not affect reads. A new getter should add a local property to its owning model,
route the public path in `ui.go`, and document the function in generated Lua
types. State changes continue to use intents.

## Architectural Summary

If you are changing behavior in `jjui`, the main mental model is:

- Bubble Tea runs the event loop
- `ui.go` is the root orchestrator
- rendering is immediate-mode through `render/`
- models handle intents, not keys
- `//jjui:bind` annotations generate the action catalog and builtin Lua surface
- one ordered list of view layers in `ui.go` decides what is drawn, which view receives keys, and what closes first
- frames are cached and only recomputed every 8ms, while messages continue to be processed in between
