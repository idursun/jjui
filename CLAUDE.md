# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

jjui is a terminal user interface (TUI) for the Jujutsu (jj) version control system, built in Go using the Bubble Tea framework. It provides an interactive interface for common jj operations like rebase, squash, bookmarks, and more.

## Build & Development Commands

```bash
# Build the application
go build ./cmd/jjui

# Install locally
go install ./...

# Run all tests
go test ./...

# Run a specific test
go test -run TestName ./path/to/package

# Run tests with verbose output
go test -v ./...

# Regenerate action catalog after changing intent annotations
go run ./cmd/genactions

# Enable debug logging (writes to debug.log)
DEBUG=1 ./jjui
```

## Architecture

### Core Structure

- **Entry point**: `cmd/jjui/main.go` - Handles CLI flags, configuration loading, and initializes the Bubble Tea program
- **Main UI model**: `internal/ui/ui.go` - Root model that orchestrates all UI components and delegates input to the dispatch pipeline

### Key Packages

**`internal/ui/`** - UI components following the Bubble Tea Model-View-Update pattern:
- `revisions/` - Main revision list view with operations (rebase, squash, abandon, etc.)
- `operations/` - Individual operations that can be performed on revisions (each operation is a separate model)
- `context/` - Application context (`MainContext`) shared across components, holds selected items and command runner
- `common/` - Shared types, messages, and interfaces used across UI components
- `intents/` - Intent types that represent user actions (Navigate, StartRebase, etc.), annotated with `//jjui:bind` for code generation
- `dispatch/` - Dispatch pipeline: `Dispatcher` resolves key presses against scoped bindings, `Resolver` maps actions to intents
- `actions/` - Generated action-to-intent catalog (`catalog_gen.go`)
- `actionmeta/` - Generated action metadata for validation and command palette (`builtins_gen.go`)
- `render/` - Immediate-mode rendering primitives (DisplayContext, TextBuilder, interactions)

### Immediate View System (DisplayContext)

Most UI models render via the immediate view system instead of returning strings.

- **Render entrypoint**: models implement `common.ImmediateModel` with `ViewRect(dl *render.DisplayContext, box layout.Box)`.
- **Frame lifecycle**: the root model (`internal/ui/ui.go`) creates a `render.DisplayContext` each frame, calls `ViewRect` on children, then renders the accumulated operations to the terminal.
- **Drawing**: use `DisplayContext` APIs (`AddDraw`, `AddFill`, effects, windows) rather than concatenating strings.
- **Interactive text**: use `render.TextBuilder` (`dl.Text(...).Styled(...).Clickable(...).Done()`) to build clickable/interactive UI segments.
- **Mouse interactions**: register interactions via `DisplayContext` (or `TextBuilder.Clickable`) so `ProcessMouseEvent` can route clicks.

**`internal/jj/`** - Jujutsu command builders:
- `commands.go` - Functions that build jj command arguments (Log, Rebase, Squash, etc.)
- `commit.go` - Commit/revision data structures

**`internal/parser/`** - Parsing jj output:
- `streaming_log_parser.go` - Parses jj log output incrementally for the revision list
- `row.go` - Parsed row structures with commit info and graph segments

**`internal/config/`** - Configuration management:
- `config.go` - Main config struct with UI, actions, bindings, and revset settings
- `loader.go` - TOML configuration file loading and overlay merging for actions/bindings
- `default/bindings.toml` - Declarative default key bindings (scoped, supports single-key and multi-key sequences)

**`cmd/genactions/`** - Code generator that scans `//jjui:bind` annotations on intent types and produces `catalog_gen.go` and `builtins_gen.go`

### Input Dispatch Pipeline

All keyboard input flows through a single pipeline:

**KeyMsg → Dispatcher → Binding → Action → Intent → Model.Update**

- The `Dispatcher` resolves key presses against scoped bindings, supporting both single-key and multi-key sequence bindings (replacing leader keys).
- Scopes form a chain from innermost to outermost; the dispatcher walks the chain to find the first matching binding.
- The `Resolver` maps actions to intents via the generated catalog, checking operation overrides, built-in actions, and Lua actions in order.
- Models only respond to intents — they never handle `tea.KeyMsg` directly.

### Component Communication

Components communicate through Bubble Tea messages (`tea.Msg`). Key message types:
- `common.RefreshMsg` - Triggers revision list refresh
- `common.SelectionChanged` - Notifies when selected revision changes
- `intents.Intent` - User actions that get handled by models

### Test Utilities

- `test/` package provides helpers for testing UI components
- `test/simulate.go` - Simulates key presses and user interactions
- `test/log_builder.go` - Builds mock jj log output for tests
- `test/test_command_runner.go` - Mock command runner for testing

## Dependencies

- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) - TUI framework
- **Lip Gloss** (`github.com/charmbracelet/lipgloss`) - Terminal styling
- **gopher-lua** (`github.com/yuin/gopher-lua`) - Lua scripting support

## Requirements

- Go 1.24.2+
- jj v0.36+ (Jujutsu VCS)

## Adding New Actions

When adding new functionality, follow these steps:

1. Create an intent type in `internal/ui/intents/` with a `//jjui:bind` annotation declaring the scope, action, and field mappings.
2. Run `go run ./cmd/genactions` to regenerate the catalog and metadata.
3. Handle the intent in the appropriate model's `Update` method.
4. Add a default binding in `internal/config/default/bindings.toml` if needed.

A staleness test (`cmd/genactions/main_test.go:TestGeneratedCatalogIsUpToDate`) ensures generated code stays in sync with annotations.
