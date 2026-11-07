package intents

//jjui:bind scope=ui action=open_revset set=Clear:true
//jjui:bind scope=revset action=edit set=Clear:$bool(clear)
type Edit struct {
	Clear bool
}

func (Edit) isIntent() {}

//jjui:bind scope=revset action=set set=Value:$string(value)
type Set struct {
	Value string
}

func (Set) isIntent() {}

//jjui:bind scope=revset action=reset
type Reset struct{}

func (Reset) isIntent() {}

//jjui:bind scope=revset action=autocomplete
//jjui:bind scope=revset action=autocomplete_back set=Reverse:true
type CompletionCycle struct {
	Reverse bool
}

func (CompletionCycle) isIntent() {}

//jjui:bind scope=revset action=move_up set=Delta:-1
//jjui:bind scope=revset action=move_down set=Delta:1
type CompletionMove struct {
	Delta int
}

func (CompletionMove) isIntent() {}

//jjui:bind scope=revset action=show_bookmarks
type RevsetBookmarks struct{}

func (RevsetBookmarks) isIntent() {}

//jjui:bind scope=revset.bookmarks action=next_remote set=Delta:1
//jjui:bind scope=revset.bookmarks action=prev_remote set=Delta:-1
//jjui:bind scope=revset.bookmarks.remotes action=next set=Delta:1
//jjui:bind scope=revset.bookmarks.remotes action=prev set=Delta:-1
type RevsetBookmarksCycleRemote struct {
	Delta int
}

func (RevsetBookmarksCycleRemote) isIntent() {}

//jjui:bind scope=revset.bookmarks action=edit_filter
type RevsetBookmarksFilter struct{}

func (RevsetBookmarksFilter) isIntent() {}

//jjui:bind scope=revset.bookmarks action=exit
type RevsetBookmarksExit struct{}

func (RevsetBookmarksExit) isIntent() {}

//jjui:bind scope=revset.bookmarks.filter action=accept
type RevsetBookmarksFilterAccept struct{}

func (RevsetBookmarksFilterAccept) isIntent() {}

//jjui:bind scope=revset.bookmarks.filter action=cancel
type RevsetBookmarksFilterCancel struct{}

func (RevsetBookmarksFilterCancel) isIntent() {}

//jjui:bind scope=revset.bookmarks action=focus_remotes
type RevsetBookmarksFocusRemotes struct{}

func (RevsetBookmarksFocusRemotes) isIntent() {}

//jjui:bind scope=revset.bookmarks.remotes action=leave
type RevsetBookmarksRemotesLeave struct{}

func (RevsetBookmarksRemotesLeave) isIntent() {}

//jjui:bind scope=revset.bookmarks.remotes action=fetch when="revset.bookmarks.can_fetch"
type RevsetBookmarksRemotesFetch struct{}

func (RevsetBookmarksRemotesFetch) isIntent() {}
