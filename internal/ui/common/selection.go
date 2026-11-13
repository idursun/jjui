package common

import "github.com/idursun/jjui/internal/jj"

type SelectedItem interface {
	Equal(other SelectedItem) bool
}

type SelectionSnapshot struct {
	Highlighted SelectedItem
	Checked     []SelectedItem
}

type SelectionProvider interface {
	Selection() SelectionSnapshot
}

// StateProvider exposes explicitly supported, live model state to consumers
// such as Lua. Implementations should return false for unknown or unavailable
// properties; callers must not infer state by reflecting over model fields.
// Implementations currently expose values that map to Lua strings, booleans,
// and numbers (Go string, bool, int, int64, and float64); richer values need an
// explicit Lua representation before they can be added.
type StateProvider interface {
	QueryState(name string) (any, bool)
}

// ApplicationStateProvider is the root-level bridge used by MainContext. A
// model may implement StateProvider without also implementing selection; only
// the application root needs to satisfy this composite interface.
type ApplicationStateProvider interface {
	StateProvider
	SelectionProvider
}

type SelectedRevision struct {
	ChangeId string
	CommitId string
}

func (s SelectedRevision) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedRevision); ok {
		return s.ChangeId == o.ChangeId && s.CommitId == o.CommitId
	}
	return false
}

type SelectedFile struct {
	ChangeId string
	CommitId string
	File     jj.FileName
}

func (s SelectedFile) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedFile); ok {
		return s.ChangeId == o.ChangeId && s.CommitId == o.CommitId && s.File == o.File
	}
	return false
}

type SelectedOperation struct {
	OperationId string
}

func (s SelectedOperation) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedOperation); ok {
		return s.OperationId == o.OperationId
	}
	return false
}

type SelectedCommit struct {
	CommitId string
}

func (s SelectedCommit) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedCommit); ok {
		return s.CommitId == o.CommitId
	}
	return false
}
