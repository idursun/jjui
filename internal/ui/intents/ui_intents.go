package intents

//jjui:bind scope=ui action=open_undo
type Undo struct{}

func (Undo) isIntent() {}

//jjui:bind scope=ui action=open_redo
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
//jjui:bind scope=oplog action=quit
//jjui:bind scope=bookmarks action=quit
//jjui:bind scope=git action=quit
type Quit struct{}

func (Quit) isIntent() {}

//jjui:bind scope=ui action=suspend
type Suspend struct{}

func (Suspend) isIntent() {}

//jjui:bind scope=ui action=expand_status
type ExpandStatusToggle struct{}

func (ExpandStatusToggle) isIntent() {}

//jjui:bind scope=ui action=open_help
type OpenHelp struct{}

func (OpenHelp) isIntent() {}

//jjui:bind scope=help action=close
type HelpClose struct{}

func (HelpClose) isIntent() {}

//jjui:bind scope=help action=filter
type HelpFilter struct{}

func (HelpFilter) isIntent() {}

//jjui:bind scope=help action=scroll_up set=Delta:-1
//jjui:bind scope=help action=scroll_down set=Delta:1
//jjui:bind scope=help action=page_up set=Delta:-10
//jjui:bind scope=help action=page_down set=Delta:10
//jjui:bind scope=help action=move_top set=Delta:0
//jjui:bind scope=help action=move_bottom set=Delta:999999
type HelpScroll struct {
	Delta int
}

func (HelpScroll) isIntent() {}

//jjui:bind scope=ui action=open_bookmarks
type OpenBookmarks struct{}

func (OpenBookmarks) isIntent() {}

//jjui:bind scope=ui action=toggle_bookmark_view
type ToggleBookmarkView struct{}

func (ToggleBookmarkView) isIntent() {}

//jjui:bind scope=ui action=focus_next_pane
type FocusNextPane struct{}

func (FocusNextPane) isIntent() {}

//jjui:bind scope=ui action=open_git
type OpenGit struct{}

func (OpenGit) isIntent() {}

//jjui:bind scope=revisions action=open_set_bookmark
type OpenSetBookmark struct{}

func (OpenSetBookmark) isIntent() {}

type BookmarksFilterKind string

const (
	BookmarksFilterMove    BookmarksFilterKind = "move"
	BookmarksFilterDelete  BookmarksFilterKind = "delete"
	BookmarksFilterForget  BookmarksFilterKind = "forget"
	BookmarksFilterTrack   BookmarksFilterKind = "track"
	BookmarksFilterUntrack BookmarksFilterKind = "untrack"
)

//jjui:bind scope=bookmarks action=bookmark_move set=Kind:BookmarksFilterMove
//jjui:bind scope=bookmarks action=bookmark_delete set=Kind:BookmarksFilterDelete
//jjui:bind scope=bookmarks action=bookmark_forget set=Kind:BookmarksFilterForget
//jjui:bind scope=bookmarks action=bookmark_track set=Kind:BookmarksFilterTrack
//jjui:bind scope=bookmarks action=bookmark_untrack set=Kind:BookmarksFilterUntrack
type BookmarksFilter struct {
	Kind BookmarksFilterKind
}

func (BookmarksFilter) isIntent() {}

//jjui:bind scope=bookmarks action=cycle_remotes set=Delta:1
//jjui:bind scope=bookmarks action=cycle_remotes_back set=Delta:-1
type BookmarksCycleRemotes struct {
	Delta int
}

func (BookmarksCycleRemotes) isIntent() {}

//jjui:bind scope=bookmarks action=filter
type BookmarksOpenFilter struct{}

func (BookmarksOpenFilter) isIntent() {}

//jjui:bind scope=bookmarks action=move_up set=Delta:-1
//jjui:bind scope=bookmarks action=move_down set=Delta:1
//jjui:bind scope=bookmarks action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=bookmarks action=page_down set=Delta:1,IsPage:true
type BookmarksNavigate struct {
	Delta  int
	IsPage bool
}

func (BookmarksNavigate) isIntent() {}

type BookmarksApplyShortcut struct {
	Key string
}

func (BookmarksApplyShortcut) isIntent() {}

//jjui:bind scope=bookmark_view action=filter
type BookmarkViewOpenFilter struct{}

func (BookmarkViewOpenFilter) isIntent() {}

//jjui:bind scope=bookmark_view action=toggle_expand
type BookmarkViewToggleExpand struct{}

func (BookmarkViewToggleExpand) isIntent() {}

//jjui:bind scope=bookmark_view action=move_up set=Delta:-1
//jjui:bind scope=bookmark_view action=move_down set=Delta:1
//jjui:bind scope=bookmark_view action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=bookmark_view action=page_down set=Delta:1,IsPage:true
type BookmarkViewNavigate struct {
	Delta  int
	IsPage bool
}

func (BookmarkViewNavigate) isIntent() {}

//jjui:bind scope=bookmark_view action=reveal
type BookmarkViewReveal struct{}

func (BookmarkViewReveal) isIntent() {}

//jjui:bind scope=bookmark_view action=reveal_in_revisions
type BookmarkViewRevealInRevisions struct{}

func (BookmarkViewRevealInRevisions) isIntent() {}

//jjui:bind scope=bookmark_view action=edit
type BookmarkViewEdit struct{}

func (BookmarkViewEdit) isIntent() {}

//jjui:bind scope=bookmark_view action=new
type BookmarkViewNew struct{}

func (BookmarkViewNew) isIntent() {}

//jjui:bind scope=bookmark_view action=rename
type BookmarkViewRename struct{}

func (BookmarkViewRename) isIntent() {}

//jjui:bind scope=bookmark_view action=delete
type BookmarkViewDelete struct{}

func (BookmarkViewDelete) isIntent() {}

//jjui:bind scope=bookmark_view action=forget
type BookmarkViewForget struct{}

func (BookmarkViewForget) isIntent() {}

//jjui:bind scope=bookmark_view action=track
type BookmarkViewTrack struct{}

func (BookmarkViewTrack) isIntent() {}

//jjui:bind scope=bookmark_view action=untrack
type BookmarkViewUntrack struct{}

func (BookmarkViewUntrack) isIntent() {}

//jjui:bind scope=bookmark_view action=move
type BookmarkViewMove struct{}

func (BookmarkViewMove) isIntent() {}

//jjui:bind scope=bookmark_view action=toggle_select
type BookmarkViewToggleSelect struct{}

func (BookmarkViewToggleSelect) isIntent() {}

type GitFilterKind string

const (
	GitFilterPush  GitFilterKind = "push"
	GitFilterFetch GitFilterKind = "fetch"
)

//jjui:bind scope=git action=push set=Kind:GitFilterPush
//jjui:bind scope=git action=fetch set=Kind:GitFilterFetch
type GitFilter struct {
	Kind GitFilterKind
}

func (GitFilter) isIntent() {}

//jjui:bind scope=git action=cycle_remotes set=Delta:1
//jjui:bind scope=git action=cycle_remotes_back set=Delta:-1
type GitCycleRemotes struct {
	Delta int
}

func (GitCycleRemotes) isIntent() {}

//jjui:bind scope=git action=filter
type GitOpenFilter struct{}

func (GitOpenFilter) isIntent() {}

//jjui:bind scope=git action=move_up set=Delta:-1
//jjui:bind scope=git action=move_down set=Delta:1
//jjui:bind scope=git action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=git action=page_down set=Delta:1,IsPage:true
type GitNavigate struct {
	Delta  int
	IsPage bool
}

func (GitNavigate) isIntent() {}

type GitApplyShortcut struct {
	Key string
}

func (GitApplyShortcut) isIntent() {}

//jjui:bind scope=choose action=move_up set=Delta:-1
//jjui:bind scope=choose action=move_down set=Delta:1
type ChooseNavigate struct {
	Delta int
}

func (ChooseNavigate) isIntent() {}

//jjui:bind scope=choose action=apply
type ChooseApply struct{}

func (ChooseApply) isIntent() {}

//jjui:bind scope=choose action=cancel
type ChooseCancel struct{}

func (ChooseCancel) isIntent() {}

//jjui:bind scope=revisions.rebase action=cancel
//jjui:bind scope=revisions.squash action=cancel
//jjui:bind scope=revisions.revert action=cancel
//jjui:bind scope=revisions.duplicate action=cancel
//jjui:bind scope=revisions action=cancel
//jjui:bind scope=revisions.details.confirmation action=cancel
//jjui:bind scope=revisions.evolog action=cancel
//jjui:bind scope=revisions.abandon action=cancel
//jjui:bind scope=revisions.absorb action=cancel
//jjui:bind scope=revisions.set_parents action=cancel
//jjui:bind scope=revisions.set_bookmark action=cancel
//jjui:bind scope=revisions.bookmark_move action=cancel
//jjui:bind scope=revisions.inline_describe action=cancel
//jjui:bind scope=revisions.ace_jump action=cancel
//jjui:bind scope=ui action=cancel
//jjui:bind scope=help action=cancel
//jjui:bind scope=bookmarks action=cancel
//jjui:bind scope=git action=cancel
//jjui:bind scope=status.input action=cancel
//jjui:bind scope=file_search action=cancel
//jjui:bind scope=revisions.quick_search.input action=cancel
//jjui:bind scope=revset action=cancel
//jjui:bind scope=password action=cancel
//jjui:bind scope=input action=cancel
//jjui:bind scope=undo action=cancel
//jjui:bind scope=redo action=cancel
//jjui:bind scope=bookmark_view action=cancel
//jjui:bind scope=bookmark_view.filter action=cancel
type Cancel struct{}

func (Cancel) isIntent() {}

//jjui:bind scope=revisions.rebase action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.rebase action=force_apply set=Force:true
//jjui:bind scope=revisions.squash action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.squash action=force_apply set=Force:true
//jjui:bind scope=revisions.revert action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.revert action=force_apply set=Force:true
//jjui:bind scope=revisions.duplicate action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.duplicate action=force_apply set=Force:true
//jjui:bind scope=revisions.details.confirmation action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.details.confirmation action=force_apply set=Force:true
//jjui:bind scope=revisions.evolog action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.abandon action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.abandon action=force_apply set=Force:true
//jjui:bind scope=revisions.absorb action=apply
//jjui:bind scope=revisions.set_parents action=apply
//jjui:bind scope=revisions.set_bookmark action=apply
//jjui:bind scope=revisions.bookmark_move action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.bookmark_move action=force_apply set=Force:true
//jjui:bind scope=revisions.ace_jump action=apply
//jjui:bind scope=bookmarks action=apply
//jjui:bind scope=git action=apply
//jjui:bind scope=revisions action=apply set=Force:$bool(force)
//jjui:bind scope=revisions action=force_apply set=Force:true
//jjui:bind scope=status.input action=apply
//jjui:bind scope=file_search action=apply
//jjui:bind scope=revisions.quick_search.input action=apply
//jjui:bind scope=revset action=apply
//jjui:bind scope=password action=apply
//jjui:bind scope=input action=apply
//jjui:bind scope=help action=apply
//jjui:bind scope=undo action=apply
//jjui:bind scope=redo action=apply
//jjui:bind scope=bookmark_view action=apply
//jjui:bind scope=bookmark_view.filter action=apply
type Apply struct {
	Value string
	Force bool
}

func (Apply) isIntent() {}

//jjui:bind scope=undo action=prev set=Delta:-1
//jjui:bind scope=undo action=next set=Delta:1
//jjui:bind scope=redo action=prev set=Delta:-1
//jjui:bind scope=redo action=next set=Delta:1
//jjui:bind scope=revisions.details.confirmation action=prev set=Delta:-1
//jjui:bind scope=revisions.details.confirmation action=next set=Delta:1
type OptionSelect struct {
	Delta int
}

func (OptionSelect) isIntent() {}
