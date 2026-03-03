package intents

//jjui:bind scope=revisions.evolog action=move_up set=Delta:-1
//jjui:bind scope=revisions.evolog action=move_down set=Delta:1
//jjui:bind scope=revisions.evolog action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=revisions.evolog action=page_down set=Delta:1,IsPage:true
type EvologNavigate struct {
	Delta  int
	IsPage bool
}

func (EvologNavigate) isIntent() {}

//jjui:bind scope=revisions.evolog action=diff
type EvologDiff struct{}

func (EvologDiff) isIntent() {}

//jjui:bind scope=revisions.evolog action=restore
type EvologRestore struct{}

func (EvologRestore) isIntent() {}
