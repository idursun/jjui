# Preview
You can open the preview window by pressing `p`. 

Preview window displays output of the `jj show` (or `jj op show`) command of the selected revision/file/operation. 

You can specify commands to display the content of different item types in the preview window using the configuration options under the `preview` table in your [configuration file](../Configuration).

The default commands are:
```toml
[preview]
revision_command = ["show", "--color", "always", "-r", "$change_id"]
oplog_command = ["op", "show", "$operation_id", "--color", "always"]
file_command = ["diff", "--color", "always", "-r", "$change_id", "$file"]
```

While the preview window is showing, you can press; `ctrl+n` to scroll one line down, `ctrl+p` to scroll one line up, `ctrl+d` to scroll half page down, `ctrl+u` to scroll half page up. 

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_preview.gif)

### Configuration

By default preview window has the following configuration:

```toml
[keys.preview]
  mode = ["p"]
  scroll_up = ["ctrl+p"]
  scroll_down = ["ctrl+n"]
  half_page_down = ["ctrl+d"]
  half_page_up = ["ctrl+u"]
  expand = ["ctrl+h"]
  shrink = ["ctrl+l"]
[preview]
  show_at_start = false
  width_percentage = 50.0
  width_increment_percentage = 5.0
  revision_command = ["show", "--color", "always", "-r", "$change_id"]
  oplog_command = ["op", "show", "$operation_id", "--color", "always"]
  file_command = ["diff", "--color", "always", "-r", "$change_id", "$file"]
```

## Preview Window Position

Pressing `P` will move the preview window to the bottom of the screen, and pressing it again will move it back to the side. You can configure `jjui` to show the preview window at the bottom by default by setting:

```toml
[preview]
show_at_bottom = true
```