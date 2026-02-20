
# Command Execution

jjui allows you to execute both shell and jj commands directly from the user interface, making it easy to perform advanced operations without leaving the application.

To run interactive jj commands, press `:`. jjui will suspend and the command will run in your terminal (for example, `: restore -i`). You will return to jjui when the command finishes.

To run shell commands, press `$`. This lets you execute any shell command (such as `$ man jj` or `$ htop`) from within jjui.

Both command types support context-aware placeholders, which are replaced with values from your current selection:
- `$file`: The currently selected file
- `$change_id`: The ID of the selected revision
- `$operation_id`: The ID of the selected operation
- `$revset`: The current revset query
- `$checked_commit_ids`: Commit ids of all the selected revisions
- `$checked_files`: File names of all the selected files

These placeholders make it easy to create powerful commands that operate on your current selection and context.

Command history for both `:` and `$` commands is stored in the `$XDG_CACHE_HOME/jjui/history/` directory. Separate history files are used for jj and shell commands (`exec_jj` and `exec_sh` respectively). Each line in the history files represents a command entry.

To import your shell history and make it available in jjui's shell prompt, you can use:

```bash
mkdir -p ~/.cache/jjui/history
history >> ~/.cache/jjui/history/exec_sh
```