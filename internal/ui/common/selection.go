package common

type SelectedItem interface {
	Equal(other SelectedItem) bool
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
	File     string
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

type FocusedObject struct {
	Kind     string
	Value    string
	ChangeId string
	CommitId string
	Index    int
}
