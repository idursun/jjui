package intents

//jjui:bind scope=ui action=undo
type Undo struct{}

func (Undo) isIntent() {}

//jjui:bind scope=ui action=redo
type Redo struct{}

func (Redo) isIntent() {}

//jjui:bind scope=ui action=exec_jj
type ExecJJ struct{}

func (ExecJJ) isIntent() {}

//jjui:bind scope=ui action=exec_shell
type ExecShell struct{}

func (ExecShell) isIntent() {}

//jjui:bind scope=revisions.evolog action=quit
//jjui:bind scope=revisions.details action=quit
//jjui:bind scope=ui action=quit
//jjui:bind scope=ui.oplog action=quit
//jjui:bind scope=ui.bookmarks action=quit
//jjui:bind scope=ui.git action=quit
type Quit struct{}

func (Quit) isIntent() {}

//jjui:bind scope=ui action=suspend
type Suspend struct{}

func (Suspend) isIntent() {}

//jjui:bind scope=ui action=expand_status
type ExpandStatusToggle struct{}

func (ExpandStatusToggle) isIntent() {}

//jjui:bind scope=ui action=open_bookmarks
type OpenBookmarks struct{}

func (OpenBookmarks) isIntent() {}

//jjui:bind scope=ui action=open_git
type OpenGit struct{}

func (OpenGit) isIntent() {}

//jjui:bind scope=ui action=open_revset
type OpenRevset struct{}

func (OpenRevset) isIntent() {}

//jjui:bind scope=revisions action=bookmark_set
type BookmarksSet struct{}

func (BookmarksSet) isIntent() {}

type BookmarksFilterKind string

const (
	BookmarksFilterMove    BookmarksFilterKind = "move"
	BookmarksFilterDelete  BookmarksFilterKind = "delete"
	BookmarksFilterForget  BookmarksFilterKind = "forget"
	BookmarksFilterTrack   BookmarksFilterKind = "track"
	BookmarksFilterUntrack BookmarksFilterKind = "untrack"
)

//jjui:bind scope=ui.bookmarks action=bookmark_move set=Kind:BookmarksFilterMove
//jjui:bind scope=ui.bookmarks action=bookmark_delete set=Kind:BookmarksFilterDelete
//jjui:bind scope=ui.bookmarks action=bookmark_forget set=Kind:BookmarksFilterForget
//jjui:bind scope=ui.bookmarks action=bookmark_track set=Kind:BookmarksFilterTrack
//jjui:bind scope=ui.bookmarks action=bookmark_untrack set=Kind:BookmarksFilterUntrack
type BookmarksFilter struct {
	Kind BookmarksFilterKind
}

func (BookmarksFilter) isIntent() {}

//jjui:bind scope=ui.bookmarks action=cycle_remotes set=Delta:1
//jjui:bind scope=ui.bookmarks action=cycle_remotes_back set=Delta:-1
type BookmarksCycleRemotes struct {
	Delta int
}

func (BookmarksCycleRemotes) isIntent() {}

//jjui:bind scope=ui.bookmarks action=filter
type BookmarksOpenFilter struct{}

func (BookmarksOpenFilter) isIntent() {}

//jjui:bind scope=ui.bookmarks action=move_up set=Delta:-1
//jjui:bind scope=ui.bookmarks action=move_down set=Delta:1
//jjui:bind scope=ui.bookmarks action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=ui.bookmarks action=page_down set=Delta:1,IsPage:true
type BookmarksNavigate struct {
	Delta  int
	IsPage bool
}

func (BookmarksNavigate) isIntent() {}

type BookmarksApplyShortcut struct {
	Key string
}

func (BookmarksApplyShortcut) isIntent() {}

type GitFilterKind string

const (
	GitFilterPush  GitFilterKind = "push"
	GitFilterFetch GitFilterKind = "fetch"
)

//jjui:bind scope=ui.git action=push set=Kind:GitFilterPush
//jjui:bind scope=ui.git action=fetch set=Kind:GitFilterFetch
type GitFilter struct {
	Kind GitFilterKind
}

func (GitFilter) isIntent() {}

//jjui:bind scope=ui.git action=cycle_remotes set=Delta:1
//jjui:bind scope=ui.git action=cycle_remotes_back set=Delta:-1
type GitCycleRemotes struct {
	Delta int
}

func (GitCycleRemotes) isIntent() {}

//jjui:bind scope=ui.git action=filter
type GitOpenFilter struct{}

func (GitOpenFilter) isIntent() {}

//jjui:bind scope=ui.git action=move_up set=Delta:-1
//jjui:bind scope=ui.git action=move_down set=Delta:1
//jjui:bind scope=ui.git action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=ui.git action=page_down set=Delta:1,IsPage:true
type GitNavigate struct {
	Delta  int
	IsPage bool
}

func (GitNavigate) isIntent() {}

type GitApplyShortcut struct {
	Key string
}

func (GitApplyShortcut) isIntent() {}

//jjui:bind scope=ui.choose action=move_up set=Delta:-1
//jjui:bind scope=ui.choose action=move_down set=Delta:1
type ChooseNavigate struct {
	Delta int
}

func (ChooseNavigate) isIntent() {}

//jjui:bind scope=ui.choose action=apply
type ChooseApply struct{}

func (ChooseApply) isIntent() {}

//jjui:bind scope=ui.choose action=cancel
type ChooseCancel struct{}

func (ChooseCancel) isIntent() {}
