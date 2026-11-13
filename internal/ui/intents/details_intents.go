package intents

//jjui:bind scope=revisions.details action=move_up set=Delta:-1
//jjui:bind scope=revisions.details action=move_down set=Delta:1
//jjui:bind scope=revisions.details action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=revisions.details action=page_down set=Delta:1,IsPage:true
type DetailsNavigate struct {
	Delta  int
	IsPage bool
}

func (DetailsNavigate) isIntent() {}

//jjui:bind scope=revisions.details action=cancel
type DetailsClose struct{}

func (DetailsClose) isIntent() {}

//jjui:bind scope=revisions.details action=filter
type DetailsOpenFilter struct{}

func (DetailsOpenFilter) isIntent() {}

//jjui:bind scope=revisions.details action=filter_apply
type DetailsApplyFilter struct{}

func (DetailsApplyFilter) isIntent() {}

//jjui:bind scope=revisions.details action=filter_cancel
type DetailsCancelFilter struct{}

func (DetailsCancelFilter) isIntent() {}

//jjui:bind scope=revisions.details action=diff when="revisions.details.has_file"
type DetailsDiff struct{}

func (DetailsDiff) isIntent() {}

//jjui:bind scope=revisions.details action=split when="revisions.details.has_selection"
//jjui:bind scope=revisions.details action=split_parallel set=IsParallel:true when="revisions.details.has_selection"
type DetailsSplit struct {
	IsParallel    bool
	IsInteractive bool
}

func (DetailsSplit) isIntent() {}

//jjui:bind scope=revisions.details action=squash when="revisions.details.has_selection"
type DetailsSquash struct{}

func (DetailsSquash) isIntent() {}

//jjui:bind scope=revisions.details action=restore when="revisions.details.has_selection"
type DetailsRestore struct{}

func (DetailsRestore) isIntent() {}

//jjui:bind scope=revisions.details action=absorb when="revisions.details.has_selection"
type DetailsAbsorb struct{}

func (DetailsAbsorb) isIntent() {}

//jjui:bind scope=revisions.details action=toggle_select when="revisions.details.has_file"
type DetailsToggleSelect struct{}

func (DetailsToggleSelect) isIntent() {}

//jjui:bind scope=revisions.details action=invert_selection
type DetailsInvertSelection struct{}

func (DetailsInvertSelection) isIntent() {}

//jjui:bind scope=revisions.details action=revisions_changing_file when="revisions.details.has_file"
type DetailsRevisionsChangingFile struct{}

func (DetailsRevisionsChangingFile) isIntent() {}

//jjui:bind scope=revisions.details action=select_file set=File:$string(file)
type DetailsSelectFile struct {
	File string
}

func (DetailsSelectFile) isIntent() {}
