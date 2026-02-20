
# Duplicate Operation

The duplicate operation allows you to copy one or more revisions directly from the user interface. This feature is useful for creating new branches or experimenting with changes without affecting the original history.

To duplicate revisions, select the revision(s) you want to copy and press `y` (the default key) to enter duplicate mode.

Workflow:
1. Select the source revision(s) you want to duplicate.
2. Press `y` to start the duplicate operation.
3. Navigate to the target revision where you want to place the copy.
4. Fine-tune the placement using sub-keys:
   - `a`: Place the copy after the target revision
   - `b`: Place the copy before the target revision
   - `d`: Place the copy onto the target revision (as the destination)
5. Press `Enter` to confirm and execute the command.

The UI provides a live preview of the operation, allowing you to see the result before confirming. Duplicate works alongside other revision operations such as rebase, squash, and abandon, giving you flexible control over your repository history.
