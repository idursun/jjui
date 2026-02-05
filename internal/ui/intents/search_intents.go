package intents

//jjui:bind scope=ui action=quick_search
type QuickSearch struct{}

func (QuickSearch) isIntent() {}

//jjui:bind scope=revisions action=quick_search_next
//jjui:bind scope=revisions action=quick_search_prev set=Reverse:true
//jjui:bind scope=oplog.quick_search action=quick_search_next
//jjui:bind scope=oplog.quick_search action=quick_search_prev set=Reverse:true
type QuickSearchCycle struct {
	Reverse bool
}

func (QuickSearchCycle) isIntent() {}

//jjui:bind scope=ui action=file_search_toggle
type FileSearchToggle struct{}

func (FileSearchToggle) isIntent() {}

//jjui:bind scope=revisions.rebase action=ace_jump
//jjui:bind scope=revisions.squash action=ace_jump
//jjui:bind scope=revisions.duplicate action=ace_jump
//jjui:bind scope=revisions.abandon action=ace_jump
//jjui:bind scope=revisions.set_parents action=ace_jump
//jjui:bind scope=revisions action=ace_jump
type StartAceJump struct{}

func (StartAceJump) isIntent() {}

//jjui:bind scope=file_search action=move_up set=Delta:1
//jjui:bind scope=file_search action=move_down set=Delta:-1
type FileSearchNavigate struct {
	Delta int
}

func (FileSearchNavigate) isIntent() {}

type FileSearchCancel struct{}

func (FileSearchCancel) isIntent() {}

//jjui:bind scope=file_search action=edit
type FileSearchEdit struct{}

func (FileSearchEdit) isIntent() {}

//jjui:bind scope=file_search action=toggle
type FileSearchTogglePreview struct{}

func (FileSearchTogglePreview) isIntent() {}

type FileSearchAccept struct{}

func (FileSearchAccept) isIntent() {}

//jjui:bind scope=file_search action=page_up set=Kind:PreviewPageUp
//jjui:bind scope=file_search action=page_down set=Kind:PreviewPageDown
//jjui:bind scope=file_search action=preview_half_page_up set=Kind:PreviewHalfPageUp
//jjui:bind scope=file_search action=preview_half_page_down set=Kind:PreviewHalfPageDown
type FileSearchPreviewScroll struct {
	Kind PreviewScrollKind
}

func (FileSearchPreviewScroll) isIntent() {}

//jjui:bind scope=status.input action=autocomplete
type SuggestCycle struct{}

func (SuggestCycle) isIntent() {}

//jjui:bind scope=status.input action=page_up set=Delta:1
//jjui:bind scope=status.input action=page_down set=Delta:-1
//jjui:bind scope=status.input action=move_up set=Delta:1
//jjui:bind scope=status.input action=move_down set=Delta:-1
type SuggestNavigate struct {
	Delta int
}

func (SuggestNavigate) isIntent() {}
