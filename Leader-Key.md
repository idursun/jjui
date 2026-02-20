> Leader key is inspired by vim's Leader, and emacs' hydra/which-key/transient.

Leader is a prefix key that allows navigating a tree of keymaps where leafs are actions to be performed in the UI as if the user typed them directly.

When Leader is activated (default keybinding is backslash: `\`), the user can navigate a keymap tree using single letter keystrokes. Actions are leafs on this tree and represent key sequences sent to the UI.

Leader keymaps represent mnemonic shortcuts that can do anything that is possible via jjui key-bindings, and allow user defined workflows that fit each person's mental model.

## Leader configuration (on your [config.toml](https://github.com/idursun/jjui/wiki/Configuration))

Leader keymaps are configured via the `leader` table.

> [!IMPORTANT]
> Each entry is identified by an alphanumeric `key-sequence`.
>
> And it can have the following optional attributes:
>
>
> `context`: An array of [context placeholders](https://github.com/idursun/jjui/blob/main/internal/jj/commands.go#L9-L14) required to enable this entry.
>
> `help`: Human message to show for keybinding.
>
> `send`: An array of keys to send into the UI.
>
> Each element of the `send` array is either a [tea.keyName](https://github.com/charmbracelet/bubbletea/blob/b224818d994537a25de86e2658fb9f437ea0baf4/key.go#L261) like `enter`, `ctrl+s`, `down`, etc. or a custom string sent directly into the UI.  

## Examples

The most basic example could be a Leader key `h` that sends `?` into the main UI.

From the main UI, use the following key sequence to invoke it: `\` `h`

```toml
[leader.h]
help = "Help"
send = ["?"]
```

More interesting are nested keymaps:

```toml
[leader.n]
context = ["$change_id"]
help = "New change"

[leader.na]
help = "After"
send = [ ":", "new -A \"$change_id\"", "enter" ]

[leader.nb]
help = "Before"
send = [ ":", "new -B \"$change_id\"", "enter" ]
```

In this example, `\n` does not have a `send` sequence, it is only used to set a `help` message for the `n` keymap.
But it does have a `context` requirement, meaning that the `n` key and its nested keymaps are only visible when `$change_id`
context value is available. That is, when a revision is selected.

From the main UI, entering the following key sequence: `\na` will
send the configured keys into jjui's event loop as if typed by the user:

- `:` Opens "exec jj" interactive command.
- types `new -A $change_id`
- executes the command by sending enter key.

## Contribute Leader keys that might be useful to others

Leader keys are intended to be user defined, so they fit your workflow and help you
easily do repetitive tasks.

You are free to adapt these examples as you like.

If you want to share some keymaps that might be valuable or serve as inspiration for
others, this is the place :).

#### Edit a file from revision detail. (idea from [#184](https://github.com/idursun/jjui/issues/184))

The following key `\E` is enabled only when the cursor is over a file in details view.
It will open `$file` after making `$change_id` current.

```toml
[leader.e]
context = [ "$file", "$change_id" ]
help = "Edit file in @"
send = [ "$", "$EDITOR $file", "enter" ]

[leader.E]
context = [ "$file", "$change_id" ]
help = "Edit file in change"
send = [ "$", "jj edit \"$change_id\" && $EDITOR $file", "enter" ]
```

#### Save the current revset under a new alias. (idea from [#169](https://github.com/idursun/jjui/issues/169))

NOTE: uses [gum](https://github.com/charmbracelet/gum) to prompt for the revset alias.

```toml
[leader.R]
context = [ "$revset" ]
help = "Save revset"
send = [ "$", "jj config set --repo revset-aliases.$(gum input --placeholder 'Revset Alias') $revset", "enter" ]
```

#### Create a bookmark on current change  (idea from [#67](https://github.com/idursun/jjui/issues/67))

```toml
[leader.bn]
help = "Set new bookmark"
send = [ "$", "jj bookmark set -r \"$change_id\" $(gum input --placeholder \"Name of the new bookmark\")", "enter" ]
```
