package view

type Scope string

const (
	ScopeNone        Scope = ""
	ScopeList        Scope = "list"
	ScopeRevisions   Scope = "revisions"
	ScopeOplog       Scope = "oplog"
	ScopeDiff        Scope = "diff"
	ScopeRevset      Scope = "revset"
	ScopePreview     Scope = "preview"
	ScopeUndo        Scope = "undo"
	ScopeBookmarks   Scope = "bookmarks"
	ScopeGit         Scope = "git"
	ScopeHelp        Scope = "help"
	ScopeStatus      Scope = "status"
	ScopeExecJJ      Scope = "exec_jj"
	ScopeExecSh      Scope = "exec_sh"
	ScopeQuickSearch Scope = "quick_search"
)
