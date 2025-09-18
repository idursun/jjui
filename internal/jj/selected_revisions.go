package jj

import "github.com/idursun/jjui/internal/models"

type SelectedRevisions []*models.RevisionItem

func NewSelectedRevisions(revision *models.RevisionItem) SelectedRevisions {
	return []*models.RevisionItem{revision}
}

func (s SelectedRevisions) Contains(revision *Commit) bool {
	if revision == nil {
		return false
	}
	for _, r := range s.Revisions {
		if r.GetChangeId() == revision.GetChangeId() {
			return true
		}
	}
	return false
}

func (s SelectedRevisions) GetIds() []string {
	var ret []string
	for _, revision := range s {
		ret = append(ret, revision.Commit.GetChangeId())
	}
	return ret
}

func (s SelectedRevisions) AsPrefixedArgs(prefix string) []string {
	var ret []string
	for _, revision := range s {
		ret = append(ret, prefix, revision.Commit.GetChangeId())
	}
	return ret
}

func (s SelectedRevisions) AsArgs() []string {
	return s.AsPrefixedArgs("-r")
}

func (s SelectedRevisions) Last() string {
	if len(s) == 0 {
		return ""
	}
	last := s[len(s)-1]
	return last.Commit.GetChangeId()
}
