This file provides guidance to coding agents working with code in this repository.

If you need architectural advice or deeper design context, read @ARCHITECTURE.md.

## Project Overview

`jjui` is a terminal user interface (TUI) for the Jujutsu (`jj`) version control system, built in Go using the Bubble Tea framework.

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
```

## Working Rules

- Models handle `intents.Intent`, not raw `tea.KeyMsg`.
- Most UI renders through the immediate view system. Prefer `common.ImmediateModel` and `ViewRect(...)` over string-building views.
- Use `render.DisplayContext` and `render.TextBuilder` for drawing and interactive text. Register mouse interactions through the display context.
- Do not hand-edit generated action files. Regenerate `internal/ui/actions/catalog_gen.go` and `internal/ui/actionmeta/builtins_gen.go` with `go run ./cmd/genactions`.
- When changing action bindings or intent annotations, run the generator and keep the staleness test passing.

## Key Locations

- `cmd/jjui/main.go` - CLI entry point and application startup
- `internal/ui/ui.go` - Root UI model, top-level routing, and dispatch orchestration
- `internal/ui/intents/` - Intent types and `//jjui:bind` annotations
- `internal/ui/dispatch/` - Binding dispatch and action resolution
- `internal/ui/render/` - Immediate-mode rendering primitives
- `internal/config/` - Configuration loading and default bindings
- `test/` - UI-oriented test helpers and simulators

## Requirements

- Go 1.24.2+
- `jj` v0.37+

## Verification Expectations

- For broad or cross-package changes, run `go test ./...`.
- For targeted changes, run the most relevant package tests in addition to any generator or formatting step the change requires.
- After modifying intent annotations or built-in actions, run `go run ./cmd/genactions` and the relevant tests, including `go test ./cmd/genactions`.

## Adding New Actions

When adding new functionality, follow these steps:

1. Create an intent type in `internal/ui/intents/` with a `//jjui:bind` annotation declaring the scope, action, and field mappings.
2. Run `go run ./cmd/genactions` to regenerate the catalog and metadata.
3. Handle the intent in the appropriate model's `Update` method.
4. Add a default binding in `internal/config/default/bindings.toml` if needed.

A staleness test (`cmd/genactions/main_test.go:TestGeneratedCatalogIsUpToDate`) ensures generated code stays in sync with annotations.
