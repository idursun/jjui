package intents

import "github.com/idursun/jjui/internal/jj"

//jjui:bind scope=revisions action=open_details
type OpenDetails struct{}

func (OpenDetails) isIntent() {}

//jjui:bind scope=revisions action=open_squash
type StartSquash struct {
	Selected jj.SelectedRevisions
	Files    []string
}

func (StartSquash) isIntent() {}

//jjui:bind scope=revisions action=open_rebase
type StartRebase struct {
	Selected jj.SelectedRevisions
	Source   RebaseSource
	Target   RebaseTarget
}

func (StartRebase) isIntent() {}

//jjui:bind scope=revisions action=open_revert
type StartRevert struct {
	Selected jj.SelectedRevisions
	Target   RevertTarget
}

func (StartRevert) isIntent() {}

//jjui:bind scope=revisions action=describe
type StartDescribe struct {
	Selected jj.SelectedRevisions
}

func (StartDescribe) isIntent() {}

//jjui:bind scope=revisions action=inline_describe
type StartInlineDescribe struct {
	Selected *jj.Commit
}

func (StartInlineDescribe) isIntent() {}

//jjui:bind scope=revisions action=open_evolog
type StartEvolog struct {
	Selected *jj.Commit
}

func (StartEvolog) isIntent() {}

//jjui:bind scope=revisions action=diff
type ShowDiff struct {
	Selected *jj.Commit
}

func (ShowDiff) isIntent() {}

//jjui:bind scope=revisions action=split
//jjui:bind scope=revisions action=split_parallel set=IsParallel:true
type StartSplit struct {
	Selected      *jj.Commit
	IsParallel    bool
	IsInteractive bool
	Files         []string
}

func (StartSplit) isIntent() {}

//jjui:bind scope=revisions action=toggle_select
type RevisionsToggleSelect struct{}

func (RevisionsToggleSelect) isIntent() {}

//jjui:bind scope=revisions action=quick_search_clear
type RevisionsQuickSearchClear struct{}

func (RevisionsQuickSearchClear) isIntent() {}

type NavigationTarget int

const (
	TargetNone NavigationTarget = iota
	TargetParent
	TargetChild
	TargetWorkingCopy
)

//jjui:bind scope=revisions action=move_up set=Delta:-1
//jjui:bind scope=revisions action=move_down set=Delta:1
//jjui:bind scope=revisions action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=revisions action=page_down set=Delta:1,IsPage:true
//jjui:bind scope=revisions action=jump_to_parent set=Target:TargetParent
//jjui:bind scope=revisions action=jump_to_children set=Target:TargetChild
//jjui:bind scope=revisions action=jump_to_working_copy set=Target:TargetWorkingCopy
//jjui:bind scope=revisions.rebase action=jump_to_working_copy set=Target:TargetWorkingCopy
//jjui:bind scope=revisions.squash action=jump_to_working_copy set=Target:TargetWorkingCopy
//jjui:bind scope=revisions.duplicate action=jump_to_working_copy set=Target:TargetWorkingCopy
//jjui:bind scope=revisions.abandon action=jump_to_working_copy set=Target:TargetWorkingCopy
//jjui:bind scope=revisions.set_parents action=jump_to_working_copy set=Target:TargetWorkingCopy
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

//jjui:bind scope=revisions action=new
type StartNew struct {
	Selected jj.SelectedRevisions
}

func (StartNew) isIntent() {}

//jjui:bind scope=revisions action=commit
type CommitWorkingCopy struct{}

func (CommitWorkingCopy) isIntent() {}

//jjui:bind scope=revisions action=edit
//jjui:bind scope=revisions action=force_edit set=IgnoreImmutable:true
type StartEdit struct {
	Selected        *jj.Commit
	IgnoreImmutable bool
}

func (StartEdit) isIntent() {}

//jjui:bind scope=revisions action=diff_edit
type StartDiffEdit struct {
	Selected *jj.Commit
}

func (StartDiffEdit) isIntent() {}

//jjui:bind scope=revisions action=absorb
type StartAbsorb struct {
	Selected *jj.Commit
}

func (StartAbsorb) isIntent() {}

//jjui:bind scope=revisions action=abandon
type StartAbandon struct {
	Selected jj.SelectedRevisions
}

func (StartAbandon) isIntent() {}

//jjui:bind scope=revisions.abandon action=toggle_select
type AbandonToggleSelect struct{}

func (AbandonToggleSelect) isIntent() {}

//jjui:bind scope=revisions action=open_duplicate
type StartDuplicate struct {
	Selected jj.SelectedRevisions
}

func (StartDuplicate) isIntent() {}

//jjui:bind scope=revisions action=set_parents
type SetParents struct {
	Selected *jj.Commit
}

func (SetParents) isIntent() {}

//jjui:bind scope=revisions.set_parents action=toggle_select
type SetParentsToggleSelect struct{}

func (SetParentsToggleSelect) isIntent() {}

//jjui:bind scope=revisions action=refresh
//jjui:bind scope=revisions.details action=refresh
type Refresh struct {
	KeepSelections   bool
	SelectedRevision string
}

func (Refresh) isIntent() {}
