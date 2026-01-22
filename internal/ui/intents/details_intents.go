package intents

type DetailsNavigate struct {
	Delta int
}

func (DetailsNavigate) isIntent() {}

type DetailsClose struct{}

func (DetailsClose) isIntent() {}

type DetailsDiff struct{}

func (DetailsDiff) isIntent() {}

type DetailsSplit struct {
	IsParallel    bool
	IsInteractive bool
}

func (DetailsSplit) isIntent() {}

type DetailsSquash struct{}

func (DetailsSquash) isIntent() {}

type DetailsRestore struct{}

func (DetailsRestore) isIntent() {}

type DetailsAbsorb struct{}

func (DetailsAbsorb) isIntent() {}

type DetailsToggleSelect struct{}

func (DetailsToggleSelect) isIntent() {}

type DetailsRevisionsChangingFile struct{}

func (DetailsRevisionsChangingFile) isIntent() {}
