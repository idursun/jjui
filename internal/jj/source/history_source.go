package source

// HistorySource wraps a list of history entries as completion items.
type HistorySource struct {
	Entries []string
}

func (s HistorySource) Fetch(_ Runner) ([]Item, error) {
	items := make([]Item, len(s.Entries))
	for i, h := range s.Entries {
		items[i] = Item{Name: h, Kind: KindHistory}
	}
	return items, nil
}
