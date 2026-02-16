# Contributing to Jujutsu UI

Thank you for your interest in contributing to `jjui`! 

We welcome contributions from the community and appreciate your help in making this project better.

## Getting Started

### Prerequisites

- Go 1.24.2 or later
- [Jujutsu version control system](https://github.com/jj-vcs/jj) v0.36 or later

### Setting up the Development Environment

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```shell
   git clone https://github.com/YOUR_USERNAME/jjui.git
   cd jjui
   ```
3. Build the project:
   ```shell
   go build ./cmd/jjui
   ```

## Contribution Guidelines

### Bug Fixes

**Bug fix pull requests are always welcome!** If you've found a bug and have a fix for it, feel free to submit a pull request directly. Please include:

- A clear description of the bug you're fixing
- Steps to reproduce the issue (if applicable)
- Your solution and why it fixes the problem

### New Features

For new features, we prefer a discussion-first approach:

1. **Open an issue first** to discuss the feature and potential implementations
2. Wait for feedback from maintainers and the community
3. Implement the agreed-upon design
4. Submit a pull request with the implementation

This process helps ensure that:
- The feature aligns with the project's goals
- We avoid duplicate work
- The implementation follows the project's patterns and conventions

### Pull Request Process

1. Ensure your code builds successfully: `go run ./cmd/jjui`
2. If you have made a change to the dependencies then update the `nix/vendor-hash` file.
3. (Optional) Submit your pull request with screenshots (if applicable)

### Code Style and Standards

- Follow standard Go conventions and formatting
- Use `go fmt` to format your code
- Ensure compatibility with the minimum supported `jj` version (v0.36+)

### Code Generation

If you add or modify intent types with `//jjui:bind` annotations in `internal/ui/intents/`, you must regenerate the action catalog:

```shell
go run ./cmd/genactions
```

This produces `catalog_gen.go` and `builtins_gen.go`. A staleness test will fail if generated code is out of sync with annotations.

### Testing

- Test your changes with different scenarios and configurations.
- If adding new features, consider adding appropriate test cases (although I know it is a pain at the moment)

## Development Tips

- The main entry point is in `cmd/jjui/main.go`
- UI components are organized in the `internal/ui/` directory
- jj integration logic is in the `internal/jj/` directory
- Configuration handling is in the `internal/config/` directory
- Default configuration and key bindings are in the `internal/config/default` directory
- Operations (rebase, squash, split, etc.) are implemented as operations the `internal/ui/operations/` directory
- Set `DEBUG=1` environment variable for printing debug messages to `debug.log` file

## Getting Help

- Check the [wiki](https://github.com/idursun/jjui/wiki) for detailed documentation
- Look at existing issues for similar problems or questions
- Feel free to ask questions in new issues

---

Thank you for contributing to `jjui`! 
