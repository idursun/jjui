package source

// RevisionSource wraps a list of revision IDs as completion items.
type RevisionSource struct {
	Entries []string
}

func (s RevisionSource) Fetch(_ Runner) ([]Item, error) {
	items := make([]Item, len(s.Entries))
	for i, revision := range s.Entries {
		items[i] = Item{Name: revision, Kind: KindRevision}
	}
	return items, nil
}
