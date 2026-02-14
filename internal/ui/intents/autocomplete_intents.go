package intents

//jjui:bind scope=revisions.target_picker action=autocomplete
//jjui:bind scope=revisions.target_picker action=autocomplete_back set=Reverse:true
//jjui:bind scope=revisions.set_bookmark action=autocomplete
//jjui:bind scope=revisions.set_bookmark action=autocomplete_back set=Reverse:true
type AutocompleteCycle struct {
	Reverse bool
}

func (AutocompleteCycle) isIntent() {}
