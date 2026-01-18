package intents

import "github.com/idursun/jjui/internal/jj"

type OpenDetails struct{}

func (OpenDetails) isIntent() {}

type StartSquash struct {
	Selected jj.SelectedRevisions
	Files    []string
}

func (StartSquash) isIntent() {}

type StartRebase struct {
	Selected jj.SelectedRevisions
	Source   RebaseSource
	Target   RebaseTarget
}

func (StartRebase) isIntent() {}

type StartRevert struct {
	Selected jj.SelectedRevisions
	Target   RevertTarget
}

func (StartRevert) isIntent() {}

type StartDescribe struct {
	Selected jj.SelectedRevisions
}

func (StartDescribe) isIntent() {}

type StartInlineDescribe struct {
	Selected *jj.Commit
}

func (StartInlineDescribe) isIntent() {}

type StartEvolog struct {
	Selected *jj.Commit
}

func (StartEvolog) isIntent() {}

type ShowDiff struct {
	Selected *jj.Commit
}

func (ShowDiff) isIntent() {}

type StartSplit struct {
	Selected   *jj.Commit
	IsParallel bool
	Files      []string
}

func (StartSplit) isIntent() {}

type RevisionsToggleSelect struct{}

func (RevisionsToggleSelect) isIntent() {}

type RevisionsQuickSearchClear struct{}

func (RevisionsQuickSearchClear) isIntent() {}

type NavigationTarget int

const (
	TargetNone NavigationTarget = iota
	TargetParent
	TargetChild
	TargetWorkingCopy
)

type Navigate struct {
	Delta       int              // +N down, -N up
	IsPage      bool             // use page-sized step when true
	Target      NavigationTarget // logical destination (parent/child/working)
	ChangeID    string           // explicit change/commit id to select
	FallbackID  string           // optional fallback change/commit id
	EnsureView  *bool            // defaults to true when nil
	AllowStream *bool            // defaults to true when nil
}

func (Navigate) isIntent() {}

type StartNew struct {
	Selected jj.SelectedRevisions
}

func (StartNew) isIntent() {}

type CommitWorkingCopy struct{}

func (CommitWorkingCopy) isIntent() {}

type StartEdit struct {
	Selected        *jj.Commit
	IgnoreImmutable bool
}

func (StartEdit) isIntent() {}

type StartDiffEdit struct {
	Selected *jj.Commit
}

func (StartDiffEdit) isIntent() {}

type StartAbsorb struct {
	Selected *jj.Commit
}

func (StartAbsorb) isIntent() {}

type StartAbandon struct {
	Selected jj.SelectedRevisions
}

func (StartAbandon) isIntent() {}

type AbandonToggleSelect struct{}

func (AbandonToggleSelect) isIntent() {}

type StartDuplicate struct {
	Selected jj.SelectedRevisions
}

func (StartDuplicate) isIntent() {}

type SetParents struct {
	Selected *jj.Commit
}

func (SetParents) isIntent() {}

type SetParentsToggleSelect struct{}

func (SetParentsToggleSelect) isIntent() {}

type Refresh struct {
	KeepSelections   bool
	SelectedRevision string
}

func (Refresh) isIntent() {}
