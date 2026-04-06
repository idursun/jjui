package intents

import "github.com/idursun/jjui/internal/jj"

//jjui:bind scope=revisions action=open_details
type OpenDetails struct{}

func (OpenDetails) isIntent() {}

//jjui:bind scope=revisions action=open_squash
type OpenSquash struct {
	Selected jj.SelectedRevisions
	Files    []string
}

func (OpenSquash) isIntent() {}

//jjui:bind scope=revisions action=open_rebase
type OpenRebase struct {
	Selected jj.SelectedRevisions
	Source   RebaseSource
	Target   ModeTarget
}

func (OpenRebase) isIntent() {}

//jjui:bind scope=revisions action=open_revert
type OpenRevert struct {
	Selected jj.SelectedRevisions
	Target   ModeTarget
}

func (OpenRevert) isIntent() {}

//jjui:bind scope=revisions action=describe
type Describe struct {
	Selected jj.SelectedRevisions
}

func (Describe) isIntent() {}

//jjui:bind scope=revisions action=open_inline_describe
type OpenInlineDescribe struct {
	Selected *jj.Commit
}

func (OpenInlineDescribe) isIntent() {}

//jjui:bind scope=revisions action=open_evolog
type OpenEvolog struct {
	Selected *jj.Commit
}

func (OpenEvolog) isIntent() {}

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

//jjui:bind scope=revisions.quick_search action=clear
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
type DiffEdit struct {
	Selected *jj.Commit
}

func (DiffEdit) isIntent() {}

//jjui:bind scope=revisions action=absorb
type Absorb struct {
	Selected *jj.Commit
}

func (Absorb) isIntent() {}

//jjui:bind scope=revisions action=open_abandon
type OpenAbandon struct {
	Selected jj.SelectedRevisions
}

func (OpenAbandon) isIntent() {}

//jjui:bind scope=revisions.abandon action=toggle_select
type AbandonToggleSelect struct{}

func (AbandonToggleSelect) isIntent() {}

//jjui:bind scope=revisions.abandon action=select_descendants
type AbandonSelectDescendants struct{}

func (AbandonSelectDescendants) isIntent() {}

//jjui:bind scope=revisions action=open_duplicate
type OpenDuplicate struct {
	Selected jj.SelectedRevisions
}

func (OpenDuplicate) isIntent() {}

//jjui:bind scope=revisions action=open_set_parents
type OpenSetParents struct {
	Selected *jj.Commit
}

func (OpenSetParents) isIntent() {}

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
