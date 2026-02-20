# How do I .... ?

Welcome to our `How do I ...?` questions and answers program.

The intention of this page is to document some use-cases that are totally possible   
but maybe are not (as of today) directly implemented as key-bindings on jjui.

It would be very difficult to have key-bindings for every imaginable workflow,  
so instead, we have tried for jjui to be flexible enough to allow you do stuff  
even if they are not directly implemented as jjui features.

We are of course, open to implementing features that are valuable to most of jjui users,   
so if something on this page seems like can be made into jjui's codebase, please open a  
[Feature-Request](https://github.com/idursun/jjui/issues/new?template=feature_request.md),
an [Implementation-Proposal](https://github.com/idursun/jjui/issues/new?template=impl_proposal.md)
or Pull-Request.  
Feel free to open an [Q&A discussion](https://github.com/idursun/jjui/discussions/categories/q-a) for anything not covered here.

Also, remember that jjui's UI is scriptable, and anything that is possible via the UI  
can be assigned key-bindings using [Leader-Key](./Leader-Key) or [Custom-Commands](./Custom-Commands)

### How do I edit files with conflicts?

Move to the revision marked as having conflicts,

- Use `d` to enter the details view, you will be presented with a list of files changed  
  in that revision and will see the files containing conflicts.
- Use ` ` (space) to check (✓) the files you want to edit.
- Use `$ vim $checked_files` to edit all of them in vim.

### How do I create a new change directly upon the current revision?

- Use `: new -A $commit_id`

### How do I create a mega merge?

- Use ` ` (space) to check (✓) the revisions you want to merge.
- Use `: new all:$checked_commit_ids` to create a merge commit having multiple parents.

### How do I squash together specific files from multiple revisions?

- Use ` ` (space) to check (✓) the revisions you want to squash from.
- Open `d` (details) on one of these revisions and  
  use ` ` (space) to check (✓) the files from details view. 
- Use `: squash --from $checked_commit_ids --into @ $checked_files `

This will squash the content of checked files from the checked revisions into the working copy.

