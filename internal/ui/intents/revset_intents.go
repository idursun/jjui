package intents

type Edit struct {
	Clear bool
}

func (Edit) isIntent() {}

//jjui:bind scope=revisions.rebase action=cancel
//jjui:bind scope=revisions.squash action=cancel
//jjui:bind scope=revisions.revert action=cancel
//jjui:bind scope=revisions.duplicate action=cancel
//jjui:bind scope=revisions.details.confirmation action=cancel
//jjui:bind scope=revisions.evolog action=cancel
//jjui:bind scope=revisions.abandon action=cancel
//jjui:bind scope=revisions.set_parents action=cancel
//jjui:bind scope=revisions.set_bookmark action=cancel
//jjui:bind scope=revisions.inline_describe action=cancel
//jjui:bind scope=revisions.ace_jump action=cancel
//jjui:bind scope=ui action=cancel
//jjui:bind scope=ui.bookmarks action=cancel
//jjui:bind scope=ui.git action=cancel
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
//jjui:bind scope=revisions.set_parents action=apply
//jjui:bind scope=revisions.set_bookmark action=apply
//jjui:bind scope=revisions.ace_jump action=apply
//jjui:bind scope=ui.bookmarks action=apply
//jjui:bind scope=ui.git action=apply
//jjui:bind scope=revisions action=apply set=Force:$bool(force)
//jjui:bind scope=revisions action=force_apply set=Force:true
//jjui:bind scope=status.input action=apply
//jjui:bind scope=file_search action=apply
//jjui:bind scope=revisions.quick_search.input action=apply
//jjui:bind scope=revset action=apply
//jjui:bind scope=password action=apply
//jjui:bind scope=input action=apply
//jjui:bind scope=undo action=apply
//jjui:bind scope=redo action=apply
type Apply struct {
	Value string
	Force bool
}

func (Apply) isIntent() {}

type Set struct {
	Value string
}

func (Set) isIntent() {}

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
