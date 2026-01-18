package intents

type RebaseSource int

const (
	RebaseSourceRevision RebaseSource = iota
	RebaseSourceBranch
	RebaseSourceDescendants
)

type RebaseTarget int

const (
	RebaseTargetDestination RebaseTarget = iota
	RebaseTargetAfter
	RebaseTargetBefore
	RebaseTargetInsert
)

type RevertTarget int

const (
	RevertTargetDestination RevertTarget = iota
	RevertTargetAfter
	RevertTargetBefore
	RevertTargetInsert
)

type DuplicateTarget int

const (
	DuplicateTargetDestination DuplicateTarget = iota
	DuplicateTargetAfter
	DuplicateTargetBefore
)

type RebaseSetSource struct {
	Source RebaseSource
}

func (RebaseSetSource) isIntent() {}

type RebaseSetTarget struct {
	Target RebaseTarget
}

func (RebaseSetTarget) isIntent() {}

type RebaseToggleSkipEmptied struct{}

func (RebaseToggleSkipEmptied) isIntent() {}

type RevertSetTarget struct {
	Target RevertTarget
}

func (RevertSetTarget) isIntent() {}

type DuplicateSetTarget struct {
	Target DuplicateTarget
}

func (DuplicateSetTarget) isIntent() {}

type SquashToggleKeepEmptied struct{}

func (SquashToggleKeepEmptied) isIntent() {}

type SquashToggleUseDestinationMessage struct{}

func (SquashToggleUseDestinationMessage) isIntent() {}

type SquashToggleInteractive struct{}

func (SquashToggleInteractive) isIntent() {}

type InlineDescribeAccept struct{}

func (InlineDescribeAccept) isIntent() {}

type InlineDescribeEditor struct{}

func (InlineDescribeEditor) isIntent() {}
