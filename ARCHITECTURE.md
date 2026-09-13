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

## Focus and Scope Tree

There is currently no separate generic focus-tree subsystem.

The UI focus tree is hardcoded in [`internal/ui/ui.go`](internal/ui/ui.go), mainly through logic such as:

- `primaryScope()`
- `alwaysOnScopes()`
- `dispatchScopes()`
- `routeIntentByOwner(...)`
- `handleUnmatched(...)`

That code determines:

- which model is considered focused
- which scopes are currently active
- which always-on scopes remain available
- where an intent or unmatched key should be routed

This keeps control flow explicit, but it also means UI composition and focus behavior are centralized in `ui.go`.

## Root UI Responsibilities

The root model in [`internal/ui/ui.go`](internal/ui/ui.go) is responsible for more than just top-level layout.

It currently owns:

- composition of major views such as revisions, preview, diff, status, oplog, and stacked dialogs
- dispatch scope selection
- action and intent routing
- top-level lifecycle actions like quit, help, undo, redo, preview toggling, and overlays
- mouse interaction handoff through the current display context
- split layout state for the preview pane

This file is the architectural center of the UI.

## Mouse and Interaction Handling

Mouse handling follows the same immediate rendering model.

During rendering, components register clickable or scrollable regions with the display context. When Bubble Tea delivers a mouse event, the root model forwards it to the active `DisplayContext`, which resolves the topmost matching interaction and optionally emits a new Bubble Tea message.

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
- focus and dispatch scope selection are currently hardcoded in `ui.go`
- frames are cached and only recomputed every 8ms, while messages continue to be processed in between
