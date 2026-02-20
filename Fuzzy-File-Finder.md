jjui features a fuzzy file finder that can be activated using `ctrl+t` shortcut by default.


## Selecting a file

When activated, you will see the list of all files that are present on the current revision.
Press `esc` to cancel the file search.

When you select a file (via `enter`) the current `revset` will be changed to show all revisions
that have touched the selected file.

You can navigate the candidates list using `up`/`down` and `ctrl+n`/`ctrl+p` as you'd expect on
shell prompts.

Using `tab` will accept the selected file, updating the revset, but will not close the file search
like `enter` does.

## Refined search

You can filter by typing parts of the path name. Space (` `) is used to refine search, that is,
to search again but only on the currently matching elements.

We use the excellent [`sahilm/fuzzy`](https://github.com/sahilm/fuzzy) library to perform search,
which is also used in some charmbracelet UI widgets.

Refined search is useful particularly for finding files, because `sahilm/fuzzy` ranks results
higher if they have a better match closer to the beginning of the string. However when finding
files, you most likely remember the file name (being the furthest part of the whole path), so
if you type the file name first and then space you can then filter by directory path.

## Live mode

Upon entering fuzzy file search with `ctrl+t`, if you press `ctrl+t` again, live preview will be
activated.

Live mode will show the revset for the file _as you type_ it and also the show the revision preview.

When live mode is active, `up` and `down` are used to move on the revision list, allowing you to
see the diff of the selected revision for the file being explored. This is useful when exploring
the changes made to a file and quickly moving to explore other files if needed.

Also, during live mode, `ctrl+n` and `ctrl+p` are used to scroll the diff preview.



