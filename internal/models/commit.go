package models

import "strings"

const (
	RootChangeId = "zzzzzzzz"
)

type Commit struct {
	ChangeId      string
	IsWorkingCopy bool
	Hidden        bool
	CommitId      string
}

func (c Commit) IsRoot() bool {
	return c.ChangeId == RootChangeId
}

func (c Commit) IsConflicting() bool {
	return strings.HasSuffix(c.ChangeId, "??")
}

func (c Commit) GetChangeId() string {
	if c.Hidden || c.IsConflicting() {
		return c.CommitId
	}
	return c.ChangeId
}
