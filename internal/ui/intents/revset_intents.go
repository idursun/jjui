package intents

type Edit struct {
	Clear bool
}

func (Edit) isIntent() {}

type Cancel struct{}

func (Cancel) isIntent() {}

type Apply struct {
	Value string
}

func (Apply) isIntent() {}

type Set struct {
	Value string
}

func (Set) isIntent() {}

type Reset struct{}

func (Reset) isIntent() {}
