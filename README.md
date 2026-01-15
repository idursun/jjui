[![Build & Test](https://github.com/idursun/jjui/actions/workflows/go.yml/badge.svg)](https://github.com/idursun/jjui/actions/workflows/go.yml)

# Jujutsu UI

`jjui` is a terminal user interface for working with [Jujutsu version control system](https://github.com/jj-vcs/jj). I have built it according to my own needs and will keep adding new features as I need them. I am open to feature requests and contributions.

If you are new `jjui`, have a look at [previously on jjui](https://github.com/idursun/jjui/discussions/443).

## Features

### Change revset with auto-complete
You can change revset while enjoying auto-complete and signature help while typing.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_revset.gif)

### Rebase
You can rebase a revision or a branch onto another revision in the revision tree.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_rebase.gif)

See [Rebase](https://github.com/idursun/jjui/wiki/Rebase) for detailed information.

### Squash
You can squash revisions into one revision, by pressing `S`. The following revision will be automatically selected. However, you can change the selection using `j` and `k`.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_squash.gif)

### Show revision details

Pressing `l` (as in going right into the details of a revision) will open the details view of the revision you selected.

In this mode, you can:
- Restore selected files using `r` (press `i` in the dialog for interactive chunk restore)
- Split selected files using `s`
- View diffs of the highlighted by pressing `d`

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_details.gif)

For detailed information, see [Details](https://github.com/idursun/jjui/wiki/Details) wiki page.

### Bookmarks
You can move bookmarks to the revision you selected.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_bookmarks.gif)


### Op Log
You can switch to op log view by pressing `o`. Pressing `r` restores the selected operation. For more information, see [Op log](https://github.com/idursun/jjui/wiki/Oplog) wiki page.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_oplog.gif)

### Preview
You can open the preview window by pressing `p`. If the selected item is a revision, then the output of `jj show` command is displayed. Similarly, `jj diff` output is displayed for selected files,  and `jj op show` output is displayed for selected operations.

While the preview window is showing, you can press; `ctrl+n` to scroll one line down, `ctrl+p` to scroll one line up, `ctrl+d` to scroll half a page down, `ctrl+u` to scroll half a page up.

Additionally, you can press `d` to show the contents of preview in diff view.

For detailed information, see [Preview](https://github.com/idursun/jjui/wiki/Preview) wiki page.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_preview.gif)

### Custom Commands

You can configure custom commands with lua scripts to set up `jjui` for you special use case.

See [Custom Command ‐ Lua Scripting](https://github.com/idursun/jjui/wiki/Custom-Command-%E2%80%90-Lua-Scripting) for Lua API documentation and custom command examples.

Below is a demo of [Quick Revset Switcher custom command](https://github.com/idursun/jjui/wiki/Custom-Command-%E2%80%90-Lua-Scripting#quick-revset-switcher):

![GIF](https://raw.githubusercontent.com/wiki/idursun/jjui/gifs/jjui_custom_command.gif)

Additionally,
* View the diff of a revision by pressing `d`.
* Edit the description of a revision by pressing `D`
* Create a _new_ revision by pressing `n`
* Split a revision by pressing `s`.
* Abandon a revision by pressing `a`.
* Absorb a revision by pressing `A`.
* _Edit_ a revision by pressing `e`
* Git _push_/_fetch_ by pressing `g`
* Undo the last change by pressing `u`
* Redo the last change by pressing `U`
* Show evolog of a revision by pressing `v`
* Jump to a revision with ace jump by pressing `f`

## Configuration

See [configuration](https://github.com/idursun/jjui/wiki/Configuration) section in the wiki.

## Installation

### Windows

Use [WinGet](https://learn.microsoft.com/windows/package-manager/winget/):

```shell
winget install IbrahimDursun.jjui
```

Use [Scoop](https://scoop.sh/):

```shell
scoop bucket add extras
scoop install jjui
```


### Homebrew

The latest release of `jjui` is available on Homebrew core:

```shell
brew install jjui
```

### Archlinux (maintained by [@TeddyHuang-00](https://github.com/TeddyHuang-00))

The built `jjui` binary from latest release is available on the AUR:

```shell
paru -S jjui-bin
# OR
yay -S jjui-bin
```

### Nix

Available in nixpkgs (maintained by [@Adda0](https://github.com/Adda0)):
```shell
nix run nixpkgs#jjui
```

This repo also provides a flake (maintained by [@vic](https://github.com/vic) and [@doprz](https://github.com/doprz)) with [flake-compat](https://github.com/NixOS/flake-compat) and an overlay:
```shell
nix run github:idursun/jjui
```

For development:
```shell
nix develop github:idursun/jjui
```

### From go install

To install the latest released (or pre-released) version:

```shell
go install github.com/idursun/jjui/cmd/jjui@latest
```

To install the latest commit from `main`:

```shell
go install github.com/idursun/jjui/cmd/jjui@HEAD
```
To install the latest commit from `main` bypassing the local cache:

```shell
GOPROXY=direct go install github.com/idursun/jjui/cmd/jjui@HEAD
```

### From source

You can build `jjui` from source.

```shell
git clone https://github.com/idursun/jjui.git
cd jjui
go install ./...
```


### From pre-built binaries
You can download pre-built binaries from the [releases](https://github.com/idursun/jjui/releases) page.

## Compatibility

Minimum supported `jj` version is **v0.36**+.

## Contributing

Feel free to submit a pull request.

You can compile `jjui` by running `go build ./cmd/jjui` in the root of the repo.
