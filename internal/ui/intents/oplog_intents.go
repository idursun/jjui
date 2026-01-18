package intents

type OpLogNavigate struct {
	Delta  int
	IsPage bool
}

func (OpLogNavigate) isIntent() {}

type OpLogOpen struct{}

func (OpLogOpen) isIntent() {}

type OpLogClose struct{}

func (OpLogClose) isIntent() {}

type OpLogShowDiff struct {
	OperationId string
}

func (OpLogShowDiff) isIntent() {}

type OpLogRestore struct {
	OperationId string
}

func (OpLogRestore) isIntent() {}

type OpLogRevert struct {
	OperationId string
}

func (OpLogRevert) isIntent() {}
