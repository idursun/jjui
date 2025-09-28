package view

type Scope string

const (
	ScopeNone      Scope = ""
	ScopeList      Scope = "list"
	ScopeRevisions Scope = "revisions"
	ScopeOplog     Scope = "oplog"
	ScopeDiff      Scope = "diff"
	ScopeRevset    Scope = "revset"
	ScopePreview   Scope = "preview"
	ScopeUndo      Scope = "undo"
	ScopeBookmarks Scope = "bookmarks"
	ScopeGit       Scope = "git"
	ScopeHelp      Scope = "help"
)
