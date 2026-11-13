package jj

const (
	RootChangeId = "zzzzzzzz"
)

type Commit struct {
	ChangeId          string
	IsWorkingCopy     bool
	Hidden            bool
	HasBookmarks      bool
	HasLocalBookmarks bool
	HasWorkspace      bool
	Root              bool
	IsEmpty           bool
	CommitId          string
}

func (c Commit) IsRoot() bool {
	// The log template displays shortened change IDs, so the root ID in a
	// parsed row is not necessarily the canonical RootChangeId.
	return c.Root || c.ChangeId == RootChangeId
}

func (c Commit) GetChangeId() string {
	if c.Hidden {
		return c.CommitId
	}
	return c.ChangeId
}

func (c *Commit) Equal(other *Commit) bool {
	if c == nil || other == nil {
		return c == nil && other == nil
	}
	if c.GetChangeId() != other.GetChangeId() {
		return false
	}
	if c.CommitId == "" || other.CommitId == "" {
		return true
	}
	return c.CommitId == other.CommitId
}
