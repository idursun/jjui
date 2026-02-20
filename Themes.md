Themes allow for detailed control over the application's appearance.

The following configuration loads a base theme from `~/.config/jjui/themes/my-theme.toml`.

```toml
# ~/.config/jjui/config.toml
[ui]
theme = "my-theme"
```

```toml
# ~/.config/jjui/themes/my-theme.toml
"selected" = { bg = "red", bold = true }
```
## Available themes.

- [@vic](https://x.com/oeiuwq) maintains a collection of [462 jjui color themes generated from base16 and base64 color schemes](https://github.com/vic/tinted-jjui).

## Overriding theme colors

You can still override certain parts of theme by defining the overrides inside `ui.colors` section. For example the following configuration will load the theme from `my-theme` file but will set the `selected` style's background colour to red.

```toml
[ui]
theme = "my-theme"

[ui.colors]
"selected" = { bg = "red" }
```

> [!NOTE]
> Theme support is actively being developed. The information on this page is subject to change as the application evolves.

## Light and Dark Theme Syntax

Theme configuration now allows you to choose different themes for light and dark modes. Use the following syntax in your configuration:

```toml
[ui.theme]
light = "my-light-theme"
dark = "my-dark-theme"
```

The previous syntax is still supported, which sets both light and dark themes to the same value:

```toml
[ui]
theme = "my-theme"
```

This flexibility allows you to customize the appearance of `jjui` for different display environments.

## Style Format

Themes are defined as a series of key-value pairs in a TOML file. Each entry consists of a **selector** (the key) that targets a UI element, and a **style table** (the value) that defines its appearance.

**Example:**
```toml
"selected" = { fg = "#FF8C00", bg = "#2B2B2B", bold = true }
"border" = { fg = "bright black" }
```

### Style Properties

The style table can contain any of the following properties:

| Property        | Type    | Description                              |
| --------------- | ------- | ---------------------------------------- |
| `fg`            | `Color` | Sets the foreground (text) color.        |
| `bg`            | `Color` | Sets the background color.               |
| `bold`          | `bool`  | If `true`, makes the text bold.          |
| `underline`     | `bool`  | If `true`, adds an underline to the text. |
| `strikethrough` | `bool`  | If `true`, adds a strikethrough line.    |
| `italic`        | `bool`  | If `true`, makes the text italic.         |

### Color Formats

Colors can be specified in one of three formats:

*   **TrueColor (Hex):** A string representing a hex color code (e.g., `"#FF4500"`).
*   **Base16 Names:** A string for standard terminal colors (e.g., `red`, `bright green`, `white`).
*   **ANSI256 Codes:** An integer from `0` to `255`.

---

## Selector Inheritance

Style resolution uses a fallback system to apply styles. This allows you to define general styles and override them with more specific ones, reducing repetition.

You can use broad styles like `"selected"` or `"border"` to apply globally. Then you can define component-level styles like `"revisions"` to style an entire section. Then, you can use more specific selectors like `"revisions selected"` to override the appearance for a specific state within that component.

**Resolution Example for `"revisions details selected"` selector:**
1.  The engine looks for a `"revisions details selected"` style.
2.  It then inherits from `"revisions details"`.
3.  It then inherits from `"revisions"`
3.  It then inherits from `"details selected"`.
4.  It then inherits from `"details"`.
5.  Finally, it inherits from the base `"selected"` style.

---

## UI Elements & Selectors

### Global Styles

These are base elements that apply throughout the application unless overridden by a more specific selector.

*   `text`: The default style for all text.
*   `dimmed`: Less important text, such as hints, descriptions, and inactive elements.
*   `selected`: The style for a currently highlighted or active item in a list or menu.
*   `border`: The style for borders around windows, panes, and pop-ups.
*   `title`: The style for titles in windows, panes, and menus.
*   `shortcut`: The style for keyboard shortcuts (e.g., `[Enter]`, `[q]`).
*   `matched`: The style for the part of the text that matches user input, typically in a completion or filter.

### Operation-Specific Styles

These styles appear during interactive operations like `rebase`, `squash` or `duplicate`.

*   `source_marker`: The marker for the revision being moved or acted upon.
*   `target_marker`: The marker for the destination of the operation.

### Application Sections

#### Revset (Top Bar)

The input bar at the top of the screen.

*   `revset title`: The "Revset:" label.
*   `revset text`: The user input area. It's recommended to make this `bold`.
*   **Completions Dropdown:**
    *   `revset completion selected`: The highlighted item in the completions list.
    *   `revset completion matched`: The part of a completion that matches the input.
    *   `revset completion dimmed`: The auto-suggested part of a completion.

#### Revisions / Oplog (Main List View)

The central list of commits or operations.

*   `revisions`: The base style for the entire list area.
*   `revisions selected`: The currently highlighted line.
*   `revisions dimmed`: Hint text shown during interactive operations.

#### Status (Bottom Bar)

The bar at the bottom showing the current mode and available actions.

*   `status`: The base style for the entire bar. A distinct `bg` is recommended.
*   `status title`: The current mode indicator (e.g., `NORMAL`). A contrasting `bg` helps it stand out.
*   The actions also uses `shortcut` and `dimmed` styles.

#### Evolog (Sub-List View)

The pop-up list showing the evolution history for a revision.

*   `evolog`: Base style for the view.
*   `evolog selected`: The highlighted item. Can be styled differently from `revisions selected` to show which pane is active.

#### Menus (Git, Bookmarks, etc.)

Pop-up menus for primary actions.

*   `menu`: Base style for menus. Should have a `border`.
*   Selected item is styled with `menu selected`
*   Filter is styled with `menu matched`. 
*   Items use `menu shortcut`, `menu title`, and `menu dimmed` styles.

#### Help Window

The pop-up window displaying key-bindings and help text.

*   `help`: The base style for the window. **To avoid a "patchy" look, define a `bg` color here.** This color will serve as the background for the entire content area.
*   The window uses `border` and `title` styles.

#### Preview (Side Pane)

The pane on the right that shows diffs or other details.

*   `preview`: The base style for the pane.
*   Uses `preview border` style for its frame.

#### Confirmation Dialog

The small inline dialog for confirmations (e.g., "Abandon all?").

*   `confirmation`: Base style for the dialog. Should have a `border`.
*   The message uses the global `text` style.
*   Options use `selected` (for the highlighted choice) and `dimmed` (for other choices) styles.

### Example "Fire" theme

<img alt="fire theme screenshot" src="https://github.com/user-attachments/assets/36be82ac-d489-4c85-9485-05af6983f5c8" />

```toml
"text" = { fg = "#F0E6D2", bg = "#1C1C1C" }
"dimmed" = { fg = "#888888" }
"selected" = { bg = "#4B2401", fg = "#FFD700" }
"border" = { fg = "#3A3A3A" }
"title" = { fg = "#FF8C00", bold = true }
"shortcut" = { fg = "#FFA500" }
"matched" = { fg = "#FFD700", underline = true }
"source_marker" = { bg = "#6B2A00", fg = "#FFFFFF" }
"target_marker" = { bg = "#800000", fg = "#FFFFFF" }
"revisions rebase source_marker" = { bold = true }
"revisions rebase target_marker" = { bold = true }
"status" = { bg = "#1A1A1A" }
"status title" = { fg = "#000000", bg = "#FF4500", bold = true }
"status shortcut" = { fg = "#FFA500" }
"status dimmed" = { fg = "#888888" }
"revset text" = { bold = true }
"revset completion selected" = { bg = "#4B2401", fg = "#FFD700" }
"revset completion matched" = { bold = true }
"revset completion dimmed" = { fg = "#505050" }
"revisions selected" = { bold = true }
"oplog selected" = { bold = true }
"evolog selected" = { bg = "#403010", fg = "#FFD700", bold = true }
"help" = { bg = "#2B2B2B" }
"help title" = { fg = "#FF8C00", bold = true, underline = true }
"help border" = { fg = "#3A3A3A" }
"menu" = { bg = "#2B2B2B" }
"menu title" = { fg = "#FF8C00", bold = true }
"menu shortcut" = { fg = "#FFA500" }
"menu dimmed" = { fg = "#888888" }
"menu border" = { fg = "#3A3A3A" }
"menu selected" = { bg = "#4B2401", fg = "#FFD700" }
"confirmation" = { bg = "#2B2B2B" }
"confirmation text" = { fg = "#F0E6D2" }
"confirmation selected" = { bg = "#4B2401", fg = "#FFD700" }
"confirmation dimmed" = { fg = "#888888" }
"confirmation border" = { fg = "#FF4500" }
"undo" = { bg = "#2B2B2B" }
"undo confirmation dimmed" = { fg = "#888888" }
"undo confirmation selected" = { bg = "#4B2401", fg = "#FFD700" }
"preview" = { fg = "#F0E6D2" }
```