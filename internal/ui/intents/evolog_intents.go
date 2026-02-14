package intents

//jjui:bind scope=revisions.evolog action=move_up set=Delta:-1
//jjui:bind scope=revisions.evolog action=move_down set=Delta:1
type EvologNavigate struct {
	Delta int
}

func (EvologNavigate) isIntent() {}

//jjui:bind scope=revisions.evolog action=diff
type EvologDiff struct{}

func (EvologDiff) isIntent() {}

//jjui:bind scope=revisions.evolog action=restore
type EvologRestore struct{}

func (EvologRestore) isIntent() {}
