package intents

type Edit struct {
	Clear bool
}

func (Edit) isIntent() {}

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
