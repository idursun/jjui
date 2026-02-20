# Jujutsu UI

`jjui` is a terminal user interface for working with [Jujutsu version control system](https://github.com/jj-vcs/jj). I have built it according to my own needs and will keep adding new features as I need them. I am open to feature requests and contributions.


### Migrating from v0.9
* [Migrating to v0.10](./Migrating-to-v0.10): Actions, bindings, config.lua, and what replaced custom commands and leader keys

## Features

### Core Features
* [Revisions](./Revisions): Main view for exploring and managing revisions
* [Preview](./Preview): View diffs and details of commits and files (with mouse-aware scrolling and auto placement)
* [Details](./Details): Explore files in a commit, with diff, split, and restore capabilities
* [Oplog](./Oplog): View the operation log of your repository
* [Command Execution](./Command-Execution): Execute shell and jj commands directly from `jjui`
* [Custom Commands](./Custom-Commands): Define your own commands with custom keybindings, key sequences, and Lua scripting
* [Fuzzy File Finder](./Fuzzy-File-Finder): Fuzzy find on all files and explore changes on them.
* [Ace Jump](./Ace-Jump): Quickly move between revisions using fast key strokes.
* [Exec](./Exec): Run interactive jj and shell commands
* [Flash Messages](./Flash-Messages): See command output and errors in the UI
* [Bookmarks](./Bookmarks): Manage bookmarks with a pop-up menu
* [Git](./Git): Run git commands from a pop-up menu

### Revision Operations
* [Rebase](./Rebase): Interactive rebase of commits
* [Absorb](./Absorb): Automatically absorb changes into commits
* [Abandon](./Abandon): Remove commits from your history
* [Duplicate](./Duplicate): Copy revisions with interactive UI
* [Squash](./Squash): Squash multiple revisions into one
* [Evolog](./Evolog): Explore and restore changes from the evolution log
* [Inline Describe](./Inline-Describe): Edit revision descriptions inline
* [Bookmark](./Bookmark): Set or move bookmarks on revisions

### Customization
* [Configuration](./Configuration): How to configure JJUI to your needs
* [Command Line Options](./Command-Line-Options): Customize behavior with command line flags
* [Leader Key](./Leader-Key): Create powerful mnemonic shortcuts with nested keymaps
* [Themes](./Themes): Customize the appearance of the UI with detailed styling options
* [Tracing](./Tracing): Highlight lanes and dim out-of-lane revisions