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

//jjui:bind scope=revisions.rebase action=set_source set=Source:$enum(source)
type RebaseSetSource struct {
	Source RebaseSource
}

func (RebaseSetSource) isIntent() {}

//jjui:bind scope=revisions.rebase action=set_target set=Target:$enum(target)
type RebaseSetTarget struct {
	Target RebaseTarget
}

func (RebaseSetTarget) isIntent() {}

//jjui:bind scope=revisions.rebase action=skip_emptied
type RebaseToggleSkipEmptied struct{}

func (RebaseToggleSkipEmptied) isIntent() {}

//jjui:bind scope=revisions.rebase action=target
type RebaseOpenTargetPicker struct{}

func (RebaseOpenTargetPicker) isIntent() {}

//jjui:bind scope=revisions.revert action=set_target set=Target:$enum(target)
type RevertSetTarget struct {
	Target RevertTarget
}

func (RevertSetTarget) isIntent() {}

//jjui:bind scope=revisions.revert action=target
type RevertOpenTargetPicker struct{}

func (RevertOpenTargetPicker) isIntent() {}

//jjui:bind scope=revisions.duplicate action=set_target set=Target:$enum(target)
type DuplicateSetTarget struct {
	Target DuplicateTarget
}

func (DuplicateSetTarget) isIntent() {}

//jjui:bind scope=revisions.duplicate action=target
type DuplicateOpenTargetPicker struct{}

func (DuplicateOpenTargetPicker) isIntent() {}

type SquashOption int

const (
	SquashOptionKeepEmptied SquashOption = iota
	SquashOptionUseDestinationMessage
	SquashOptionInteractive
)

//jjui:bind scope=revisions.squash action=keep_emptied set=Option:SquashOptionKeepEmptied
//jjui:bind scope=revisions.squash action=use_destination_msg set=Option:SquashOptionUseDestinationMessage
//jjui:bind scope=revisions.squash action=interactive set=Option:SquashOptionInteractive
type SquashToggleOption struct {
	Option SquashOption
}

func (SquashToggleOption) isIntent() {}

//jjui:bind scope=revisions.squash action=target
type SquashOpenTargetPicker struct{}

func (SquashOpenTargetPicker) isIntent() {}

//jjui:bind scope=revisions.inline_describe action=accept
type InlineDescribeAccept struct{}

func (InlineDescribeAccept) isIntent() {}

//jjui:bind scope=revisions.inline_describe action=editor
type InlineDescribeEditor struct{}

func (InlineDescribeEditor) isIntent() {}

//jjui:bind scope=revisions.target_picker action=move_up set=Delta:-1
//jjui:bind scope=revisions.target_picker action=move_down set=Delta:1
type TargetPickerNavigate struct {
	Delta int
}

func (TargetPickerNavigate) isIntent() {}

//jjui:bind scope=revisions.target_picker action=apply set=Force:$bool(force)
//jjui:bind scope=revisions.target_picker action=force_apply set=Force:true
type TargetPickerApply struct {
	Force bool
}

func (TargetPickerApply) isIntent() {}

//jjui:bind scope=revisions.target_picker action=cancel
type TargetPickerCancel struct{}

func (TargetPickerCancel) isIntent() {}
