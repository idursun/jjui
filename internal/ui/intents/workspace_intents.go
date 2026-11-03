package intents

//jjui:bind scope=ui action=open_workspaces
type OpenWorkspaces struct{}

func (OpenWorkspaces) isIntent() {}

//jjui:bind scope=workspace_pane action=move_up set=Delta:-1
//jjui:bind scope=workspace_pane action=move_down set=Delta:1
type WorkspaceNavigate struct{ Delta int }

func (WorkspaceNavigate) isIntent() {}

//jjui:bind scope=workspace_pane action=add
type WorkspaceAdd struct{}

func (WorkspaceAdd) isIntent() {}

//jjui:bind scope=workspace_pane action=rename
type WorkspaceRename struct{}

func (WorkspaceRename) isIntent() {}

//jjui:bind scope=workspace_pane action=forget
type WorkspaceForget struct{}

func (WorkspaceForget) isIntent() {}

//jjui:bind scope=workspace_pane action=remove
type WorkspaceRemove struct{}

func (WorkspaceRemove) isIntent() {}

//jjui:bind scope=workspace_pane action=root
type WorkspaceRoot struct{}

func (WorkspaceRoot) isIntent() {}

//jjui:bind scope=workspace_pane action=update_stale
type WorkspaceUpdateStale struct{}

func (WorkspaceUpdateStale) isIntent() {}

//jjui:bind scope=workspace_pane action=show_in_revision
type WorkspaceReveal struct{}

func (WorkspaceReveal) isIntent() {}

//jjui:bind scope=workspace_pane action=refresh
type WorkspaceRefresh struct{}

func (WorkspaceRefresh) isIntent() {}
