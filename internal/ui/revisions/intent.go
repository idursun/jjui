package revisions

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/revert"
)

// Intent represents a high-level action the revisions view can perform.
// It decouples inputs (keyboard/mouse/macros) from the actual capability.
type Intent interface {
	apply(*Model) tea.Cmd
}

type OpenDetails struct{}

func (OpenDetails) apply(m *Model) tea.Cmd {
	return m.openDetails(OpenDetails{})
}

type StartSquash struct {
	Selected jj.SelectedRevisions
	Files    []string
}

func (s StartSquash) apply(m *Model) tea.Cmd {
	return m.startSquash(s)
}

type StartRebase struct {
	Selected jj.SelectedRevisions
	Source   rebase.Source
	Target   rebase.Target
}

func (s StartRebase) apply(m *Model) tea.Cmd {
	return m.startRebase(s)
}

type StartRevert struct {
	Selected jj.SelectedRevisions
	Target   revert.Target
}

func (s StartRevert) apply(m *Model) tea.Cmd {
	return m.startRevert(s)
}

type StartDescribe struct {
	Selected jj.SelectedRevisions
}

func (s StartDescribe) apply(m *Model) tea.Cmd {
	return m.startDescribe(s)
}

type StartInlineDescribe struct {
	Selected *jj.Commit
}

func (s StartInlineDescribe) apply(m *Model) tea.Cmd {
	return m.startInlineDescribe(s)
}

type StartEvolog struct {
	Selected *jj.Commit
}

func (s StartEvolog) apply(m *Model) tea.Cmd {
	return m.startEvolog(s)
}

type ShowDiff struct {
	Selected *jj.Commit
}

func (s ShowDiff) apply(m *Model) tea.Cmd {
	return m.showDiff(s)
}

type StartSplit struct {
	Selected   *jj.Commit
	IsParallel bool
	Files      []string
}

func (s StartSplit) apply(m *Model) tea.Cmd {
	return m.startSplit(s)
}

type NavigationTarget int

const (
	TargetNone NavigationTarget = iota
	TargetParent
	TargetChild
	TargetWorkingCopy
)

type Navigate struct {
	Delta       int              // +N down, -N up
	Page        bool             // use page-sized step when true
	Target      NavigationTarget // logical destination (parent/child/working)
	ChangeID    string           // explicit change/commit id to select
	FallbackID  string           // optional fallback change/commit id
	EnsureView  *bool            // defaults to true when nil
	AllowStream *bool            // defaults to true when nil
}

func (n Navigate) apply(m *Model) tea.Cmd {
	return m.navigate(n)
}

type StartNew struct {
	Selected jj.SelectedRevisions
}

func (s StartNew) apply(m *Model) tea.Cmd {
	return m.startNew(s)
}

type CommitWorkingCopy struct{}

func (CommitWorkingCopy) apply(m *Model) tea.Cmd {
	return m.commitWorkingCopy()
}

type StartEdit struct {
	Selected        *jj.Commit
	IgnoreImmutable bool
}

func (s StartEdit) apply(m *Model) tea.Cmd {
	return m.startEdit(s)
}

type StartDiffEdit struct {
	Selected *jj.Commit
}

func (s StartDiffEdit) apply(m *Model) tea.Cmd {
	return m.startDiffEdit(s)
}

type StartAbsorb struct {
	Selected *jj.Commit
}

func (s StartAbsorb) apply(m *Model) tea.Cmd {
	return m.startAbsorb(s)
}

type StartAbandon struct {
	Selected jj.SelectedRevisions
}

func (s StartAbandon) apply(m *Model) tea.Cmd {
	return m.startAbandon(s)
}

type StartDuplicate struct {
	Selected jj.SelectedRevisions
}

func (s StartDuplicate) apply(m *Model) tea.Cmd {
	return m.startDuplicate(s)
}

type SetParents struct {
	Selected *jj.Commit
}

func (s SetParents) apply(m *Model) tea.Cmd {
	return m.startSetParents(s)
}

type Refresh struct {
	KeepSelections   bool
	SelectedRevision string
}

func (r Refresh) apply(m *Model) tea.Cmd {
	return m.refresh(r)
}
