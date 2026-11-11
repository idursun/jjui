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
type Quit struct{}

func (Quit) isIntent() {}

//jjui:bind scope=ui action=suspend
type Suspend struct{}

func (Suspend) isIntent() {}

//jjui:bind scope=ui action=change_theme set=Name:$string(name)
type ChangeTheme struct {
	Name string
}

func (ChangeTheme) isIntent() {}

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

//jjui:bind scope=revisions action=open_set_bookmark set=Value:$string?(value)
type OpenSetBookmark struct {
	Value    string
	Revision string
}

func (OpenSetBookmark) isIntent() {}

//jjui:bind scope=choose action=filter
type ChooseOpenFilter struct{}

func (ChooseOpenFilter) isIntent() {}

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
//jjui:bind scope=revisions.inline_describe.confirmation action=cancel
//jjui:bind scope=annotation.confirmation action=cancel
//jjui:bind scope=revset.bookmarks.remotes.editor action=cancel
//jjui:bind scope=revset.bookmarks.remotes.confirmation action=cancel
//jjui:bind scope=revisions.evolog action=cancel
//jjui:bind scope=revisions.abandon action=cancel
//jjui:bind scope=revisions.absorb action=cancel
//jjui:bind scope=revisions.set_parents action=cancel
//jjui:bind scope=revisions.set_bookmark action=cancel
//jjui:bind scope=revisions.rename_workspace action=cancel
//jjui:bind scope=revisions.diff_range action=cancel
//jjui:bind scope=revisions.new_between action=cancel
//jjui:bind scope=revisions.bookmark_target action=cancel
//jjui:bind scope=revisions.inline_describe action=cancel
//jjui:bind scope=revisions.ace_jump action=cancel
//jjui:bind scope=ui action=cancel
//jjui:bind scope=help action=cancel
//jjui:bind scope=status.input action=cancel
//jjui:bind scope=file_search action=cancel
//jjui:bind scope=revisions.quick_search.input action=cancel
//jjui:bind scope=revset action=cancel
//jjui:bind scope=password action=cancel
//jjui:bind scope=input action=cancel
//jjui:bind scope=undo action=cancel
//jjui:bind scope=redo action=cancel
type Cancel struct{}

func (Cancel) isIntent() {}

//jjui:bind scope=ui.overlay action=dismiss
type DismissOverlay struct{}

func (DismissOverlay) isIntent() {}

//jjui:bind scope=revisions.rebase action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.rebase action=force_apply set=Force:true
//jjui:bind scope=revisions.squash action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.squash action=force_apply set=Force:true
//jjui:bind scope=revisions.revert action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.revert action=force_apply set=Force:true
//jjui:bind scope=revisions.duplicate action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.duplicate action=force_apply set=Force:true
//jjui:bind scope=revisions.details.confirmation action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.inline_describe.confirmation action=apply set=Force:$bool(force)
//jjui:bind scope=annotation.confirmation action=apply set=Force:$bool(force)
//jjui:bind scope=revset.bookmarks.remotes.editor action=apply
//jjui:bind scope=revset.bookmarks.remotes.confirmation action=apply
//jjui:bind scope=revisions.details.confirmation action=force_apply set=Force:true
//jjui:bind scope=revisions.evolog action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.abandon action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.abandon action=force_apply set=Force:true
//jjui:bind scope=revisions.absorb action=apply
//jjui:bind scope=revisions.set_parents action=apply
//jjui:bind scope=revisions.set_bookmark action=apply
//jjui:bind scope=revisions.rename_workspace action=apply
//jjui:bind scope=revisions.diff_range action=apply
//jjui:bind scope=revisions.new_between action=apply
//jjui:bind scope=revisions.bookmark_target action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.bookmark_target action=force_apply set=Force:true
//jjui:bind scope=revisions.ace_jump action=apply
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
//jjui:bind scope=revisions.inline_describe.confirmation action=prev set=Delta:-1
//jjui:bind scope=annotation.confirmation action=prev set=Delta:-1
//jjui:bind scope=revisions.inline_describe.confirmation action=next set=Delta:1
//jjui:bind scope=annotation.confirmation action=next set=Delta:1
//jjui:bind scope=revset.bookmarks.remotes.confirmation action=prev set=Delta:-1
//jjui:bind scope=revset.bookmarks.remotes.confirmation action=next set=Delta:1
type OptionSelect struct {
	Delta int
}

func (OptionSelect) isIntent() {}
