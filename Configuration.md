JJUI loads configuration on start up from the system's config directory. 

* Linux & macOS: `~/.config/jjui/config.toml`
* Windows: `%AppData%/jjui/config.toml` (Might have to manually create folder).
* Custom: `$JJUI_CONFIG_DIR/config.toml`

You can edit the configuration in your `$EDITOR` by passing the `--config` argument:

```bash
jjui --config
```
### Key bindings

Key binding support is limited by the key handling capabilities of the terminal emulator you are using.

**Note**: Order of the modifiers matter; for example `ctrl+alt+up` is read as `alt+ctrl+up`, so `alt` should come before `ctrl`.

**Note**: `shift` combined with letters is not supported by the underlying library that `jjui` is using for rendering and key handling. So, `ctrl+shift+f` is read as `ctrl+f`. However, `shift+f` can be defined as `F`, and it should work.

### Colours and Themes

UI appearance is configured through the theming system. You can customize the colors and styles of various UI elements in your config.toml file or by creating custom theme files.

For example, to customize the appearance of selected items:

```toml
[ui.colors]
"selected" = { bg = "your colour" }
```

For more detailed customization options, see the [Themes](./Themes) page.

### Overriding Default Revset and Log Format

By default, jjui reads and uses the default revset and log format as configured in jj. You can override these values in your configuration file:

```toml
[revisions]
template = 'builtin_log_compact' # overrides jj's templates.log
revset = ""  # overrides jj's revsets.log
```

This allows you to customize how revisions and logs are displayed in jjui, independent of your jj configuration.

### Log Batching in Revisions

Log batching is enabled by default in jjui. Instead of loading all revisions at startup, jjui loads the first 50 revisions and fetches more as you scroll. This improves startup times for large repositories, with minimal effect on small ones.

This feature is turned on by default but you can turn it off with the following configuration:

```toml
[revisions]
log_batching = false
```

### Suggest Mode in Exec Commands

When running custom exec jj or exec shell commands (not to be confused with custom commands), you can choose between several suggestion modes: *off*, *regex*, *fuzzy*.
They match the input for your custom exec command with your previously run custom exec commands.
You can configure the suggest mode with `suggest.exec.mode` configuration option:  

```toml
[suggest]
  [suggest.exec]
    mode = "fuzzy" # Accepted values: "off" (default), "regex", "fuzzy".
```

### Default configuration

You can find the default configuration in the repo here: [https://github.com/idursun/jjui/blob/main/internal/config/default/config.toml](https://github.com/idursun/jjui/blob/main/internal/config/default/config.toml)