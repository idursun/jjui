package jj

import "slices"

type SelectedRevisions struct {
	Revisions []*Commit
}

func NewSelectedRevisions(revisions ...*Commit) SelectedRevisions {
	filtered := make([]*Commit, 0, len(revisions))
	for _, revision := range revisions {
		if revision != nil {
			filtered = append(filtered, revision)
		}
	}

	return SelectedRevisions{
		Revisions: filtered,
	}
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

func (s SelectedRevisions) Toggle(revision *Commit) SelectedRevisions {
	if revision == nil {
		return s
	}
	if s.Contains(revision) {
		return s.Remove(revision)
	}
	return s.Add(revision)
}

func (s SelectedRevisions) Remove(revision *Commit) SelectedRevisions {
	if revision == nil {
		return s
	}
	index := slices.IndexFunc(s.Revisions, func(candidate *Commit) bool { return candidate.GetChangeId() == revision.GetChangeId() })
	if index != -1 {
		s.Revisions = append(s.Revisions[:index], s.Revisions[index+1:]...)
	}
	return s
}

func (s SelectedRevisions) Add(revision *Commit) SelectedRevisions {
	if revision == nil {
		return s
	}
	return SelectedRevisions{
		Revisions: append(s.Revisions, revision),
	}
}

func (s SelectedRevisions) GetIds() []string {
	var ret []string
	for _, revision := range s.Revisions {
		ret = append(ret, revision.GetChangeId())
	}
	return ret
}

func (s SelectedRevisions) AsPrefixedArgs(prefix string) []string {
	var ret []string
	for _, revision := range s.Revisions {
		ret = append(ret, prefix, revision.GetChangeId())
	}
	return ret
}

func (s SelectedRevisions) AsArgs() []string {
	return s.AsPrefixedArgs("-r")
}

func (s SelectedRevisions) Last() string {
	if len(s.Revisions) == 0 {
		return ""
	}
	last := s.Revisions[len(s.Revisions)-1]
	return last.GetChangeId()
}
