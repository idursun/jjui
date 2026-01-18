package intents

type EvologNavigate struct {
	Delta int
}

func (EvologNavigate) isIntent() {}

type EvologDiff struct{}

func (EvologDiff) isIntent() {}

type EvologRestore struct{}

func (EvologRestore) isIntent() {}
