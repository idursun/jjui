package jj

import (
	"fmt"
	"strings"
)

const (
	RootChangeId = "zzzzzzzz"
)

type Commit struct {
	ChangeIdShort string
	ChangeId      string
	IsWorkingCopy bool
	Author        string
	Timestamp     string
	Bookmarks     []string
	Description   string
	Immutable     bool
	Conflict      bool
	Empty         bool
	Hidden        bool
	CommitIdShort string
	CommitId      string
}

func (c Commit) IsRoot() bool {
	return c.ChangeId == RootChangeId
}

func (c Commit) GetChangeId() string {
	if c.Hidden {
		return c.CommitId
	}
	return c.ChangeId
}

func SelectRelativeBranch(from string, to string) []string {
	const template = "separate(';', change_id.shortest(1)) ++ '\n'"
	return []string{"log", "-r", fmt.Sprintf("(%s..%s)::", to, from), "--color", "never", "--no-graph", "--template", template}
}

func ParseRevisions(output string) []string {
	var revisions []string
	lines := strings.Split(output, "\n")
	for _, rev := range lines {
		revisions = append(revisions, rev)
	}
	return revisions
}
