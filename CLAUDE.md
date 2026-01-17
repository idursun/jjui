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

# Enable debug logging (writes to debug.log)
DEBUG=1 ./jjui
```

## Architecture

### Core Structure

- **Entry point**: `cmd/jjui/main.go` - Handles CLI flags, configuration loading, and initializes the Bubble Tea program
- **Main UI model**: `internal/ui/ui.go` - Root model that orchestrates all UI components and handles global key bindings

### Key Packages

**`internal/ui/`** - UI components following the Bubble Tea Model-View-Update pattern:
- `revisions/` - Main revision list view with operations (rebase, squash, abandon, etc.)
- `operations/` - Individual operations that can be performed on revisions (each operation is a separate model)
- `context/` - Application context (`MainContext`) shared across components, holds selected items, command runner, and custom commands
- `common/` - Shared types, messages, and interfaces used across UI components
- `intents/` - Intent types that represent user actions (Navigate, StartRebase, etc.)
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
- `config.go` - Main config struct with UI, keybindings, and revset settings
- `keys.go` - Key binding definitions and mapping
- `loader.go` - TOML configuration file loading

### Component Communication

Components communicate through Bubble Tea messages (`tea.Msg`). Key message types:
- `common.RefreshMsg` - Triggers revision list refresh
- `common.SelectionChanged` - Notifies when selected revision changes
- `intents.Intent` - User actions that get handled by the revisions model

### Custom Commands (will be deprecated)

Users can define custom commands in their config that get bound to keys. Custom command types:
- `CustomRunCommand` - Executes shell commands
- `CustomRevsetCommand` - Changes the current revset
- `CustomLuaCommand` - Runs Lua scripts

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

## Ongoing Refactoring

### Intent-Based Input Handling

There is an ongoing effort to decouple input handling from model logic via the intent system (`internal/ui/intents/`). The goal is for intents to be the single mechanism for triggering model actions:

- Key bindings → Intent → Model handles intent
- Mouse actions → Intent → Model handles intent
- Lua scripts → Intent → Model handles intent

When adding new functionality, prefer creating an intent type and handling it in `handleIntent()` rather than directly coupling key handlers to model mutations.

### Future: Unified Action/Binding Configuration

The current system has separate concepts for leader keys and custom commands. The planned direction is to consolidate these into a single unified `action` and `binding` configuration system. When working on leader or custom command code, be aware this area is subject to significant redesign.
