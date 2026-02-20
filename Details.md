
# Details

The details view provides an in-depth look at the files and changes within a selected revision. To open the details view, press `l` while a revision is selected.

In details mode, you can:
- Leave the details mode using `h`
- Split selected files using `s`
- Restore selected files using `r`
- View the diff of the highlighted file by pressing `d`
- Change the revset to show all revisions that touched the selected file by pressing `*`

You can toggle file selection using `m` or `space` keys. These actions allow you to manage files within a revision, split changes, and restore previous states efficiently.

## Showing Diff
Press `d` to show the diff of the selected file. This helps you review changes before making further modifications.

## Splitting Files in a Revision
Press `s` to split the selected files into two revisions:
- Selected files remain in the current revision.
- Unselected files move to a new revision.
If no file is selected, the currently highlighted file will be split.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_details_split.gif)

## Restoring Files in a Revision
Press `r` to restore the selected files to their previous state, discarding changes as needed.

## Show Revisions That Touched the File
Press `*` to change the revset to show all revisions that have affected the highlighted file. This is useful for tracking the history of specific files.

## Conflict Markers
The details view displays conflict markers next to files with conflicts, making it easy to identify and resolve issues directly from the panel.

Details mode works together with other revision operations such as rebase, squash, and abandon, providing a comprehensive workflow for managing changes in your repository.

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_details_restore.gif)

![GIF](https://github.com/idursun/jjui/wiki/gifs/jjui_details_diff.gif)
<img width="912" alt="image" src="https://github.com/user-attachments/assets/ab33ae31-a9cb-4721-bf1c-ff72f1319bb7" />
