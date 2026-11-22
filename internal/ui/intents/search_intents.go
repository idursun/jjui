package intents

//jjui:bind scope=ui action=quick_search
type QuickSearch struct{}

func (QuickSearch) isIntent() {}

//jjui:bind scope=revisions.quick_search action=next
//jjui:bind scope=revisions.quick_search action=prev set=Reverse:true
//jjui:bind scope=oplog.quick_search action=next
//jjui:bind scope=oplog.quick_search action=prev set=Reverse:true
//jjui:bind scope=file_view action=next_match
//jjui:bind scope=file_view action=prev_match set=Reverse:true
type QuickSearchCycle struct {
	Reverse bool
}

func (QuickSearchCycle) isIntent() {}

//jjui:bind scope=revisions action=ace_jump
type StartAceJump struct{}

func (StartAceJump) isIntent() {}

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
