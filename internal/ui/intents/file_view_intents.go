package intents

//jjui:bind scope=ui action=open_file_view
type FileViewOpen struct{}

func (FileViewOpen) isIntent() {}

//jjui:bind scope=file_view action=move_up set=Delta:-1
//jjui:bind scope=file_view action=move_down set=Delta:1
//jjui:bind scope=file_view action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=file_view action=page_down set=Delta:1,IsPage:true
//jjui:bind scope=file_view action=half_page_up set=Delta:-1,IsHalfPage:true
//jjui:bind scope=file_view action=half_page_down set=Delta:1,IsHalfPage:true
//jjui:bind scope=file_view action=move_top set=Delta:-1,ToEnd:true
//jjui:bind scope=file_view action=move_bottom set=Delta:1,ToEnd:true
//jjui:bind scope=file_view.path action=move_up set=Delta:-1
//jjui:bind scope=file_view.path action=move_down set=Delta:1
type FileViewNavigate struct {
	Delta      int
	IsPage     bool
	IsHalfPage bool
	ToEnd      bool
}

func (FileViewNavigate) isIntent() {}

//jjui:bind scope=file_view action=next_change set=Delta:1
//jjui:bind scope=file_view action=prev_change set=Delta:-1
type FileViewMoveChange struct {
	Delta int
}

func (FileViewMoveChange) isIntent() {}

// FileViewOlder annotates the file as it was before the change under the cursor.
//
//jjui:bind scope=file_view action=older
type FileViewOlder struct{}

func (FileViewOlder) isIntent() {}

// FileViewNewer returns to the annotation FileViewOlder came from.
//
//jjui:bind scope=file_view action=newer
type FileViewNewer struct{}

func (FileViewNewer) isIntent() {}

//jjui:bind scope=file_view action=left set=Delta:-1
//jjui:bind scope=file_view action=right set=Delta:1
type FileViewScrollHorizontal struct {
	Delta int
}

func (FileViewScrollHorizontal) isIntent() {}

//jjui:bind scope=file_view action=toggle_wrap
type FileViewToggleWrap struct{}

func (FileViewToggleWrap) isIntent() {}

//jjui:bind scope=file_view action=search
type FileViewSearch struct{}

func (FileViewSearch) isIntent() {}

//jjui:bind scope=file_view.search_input action=apply
type FileViewSearchApply struct{}

func (FileViewSearchApply) isIntent() {}

//jjui:bind scope=file_view.search_input action=cancel
type FileViewSearchCancel struct{}

func (FileViewSearchCancel) isIntent() {}

//jjui:bind scope=file_view action=jump_to_change
type FileViewJumpToChange struct{}

func (FileViewJumpToChange) isIntent() {}

//jjui:bind scope=file_view action=filter_revset
type FileViewFilterRevset struct{}

func (FileViewFilterRevset) isIntent() {}

//jjui:bind scope=file_view action=edit_path
type FileViewEditPath struct{}

func (FileViewEditPath) isIntent() {}

// FileViewClose clears the search first, when there is one.
//
//jjui:bind scope=file_view action=close
type FileViewClose struct{}

func (FileViewClose) isIntent() {}

//jjui:bind scope=file_view.path action=accept
type FileViewPathAccept struct{}

func (FileViewPathAccept) isIntent() {}

//jjui:bind scope=file_view.path action=cancel
type FileViewPathCancel struct{}

func (FileViewPathCancel) isIntent() {}
