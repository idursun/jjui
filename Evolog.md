
# Evolog Operation

The evolog operation provides a specialized view for exploring and restoring changes from the evolution log of your repository. This feature is accessible from the Revisions view and offers interactive controls for inspecting and restoring historical changes.

To activate evolog, press `v` in the Revisions view to open the evolog output for the selected revision(s).

Available actions include:
- Restore: Press `r` to restore the selected evolog item. This is equivalent to running `jj restore --from <selected evolog commit id> --into <selected target revision change id> --restore-descendants`.
- Show Diff: Press `d` to display the diff for the selected evolog item.

Evolog is often used together with other revision operations such as squash, duplicate, and abandon, to help you manage and recover changes in your repository history.
