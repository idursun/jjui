package jj

type SelectedRevisions struct {
	Revisions []*Commit
}

func NewSelectedRevisions(revisions ...*Commit) SelectedRevisions {
	return SelectedRevisions{
		Revisions: revisions,
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
