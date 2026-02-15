package intents

//jjui:bind scope=revisions.details action=move_up set=Delta:-1
//jjui:bind scope=revisions.details action=move_down set=Delta:1
type DetailsNavigate struct {
	Delta int
}

func (DetailsNavigate) isIntent() {}

//jjui:bind scope=revisions.details action=cancel
type DetailsClose struct{}

func (DetailsClose) isIntent() {}

//jjui:bind scope=revisions.details action=diff
type DetailsDiff struct{}

func (DetailsDiff) isIntent() {}

//jjui:bind scope=revisions.details action=split
//jjui:bind scope=revisions.details action=split_parallel set=IsParallel:true
type DetailsSplit struct {
	IsParallel    bool
	IsInteractive bool
}

func (DetailsSplit) isIntent() {}

//jjui:bind scope=revisions.details action=squash
type DetailsSquash struct{}

func (DetailsSquash) isIntent() {}

//jjui:bind scope=revisions.details action=restore
type DetailsRestore struct{}

func (DetailsRestore) isIntent() {}

//jjui:bind scope=revisions.details action=absorb
type DetailsAbsorb struct{}

func (DetailsAbsorb) isIntent() {}

//jjui:bind scope=revisions.details action=toggle_select
type DetailsToggleSelect struct{}

func (DetailsToggleSelect) isIntent() {}

//jjui:bind scope=revisions.details action=revisions_changing_file
type DetailsRevisionsChangingFile struct{}

func (DetailsRevisionsChangingFile) isIntent() {}
