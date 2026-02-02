package source

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/idursun/jjui/internal/jj"
)

type Kind int

const (
	KindFunction Kind = iota
	KindAlias
	KindHistory
	KindBookmark
	KindTag
)

// Item represents a completion/picker item from any source.
type Item struct {
	Name          string
	Kind          Kind
	SignatureHelp string
}

// Runner executes a jj command and returns its output.
type Runner = func([]string) ([]byte, error)

// Source provides items for completion or selection.
type Source interface {
	Fetch(runner Runner) ([]Item, error)
}

// FetchAll collects items from multiple sources, skipping failures.
func FetchAll(runner Runner, sources ...Source) []Item {
	var all []Item
	for _, s := range sources {
		items, err := s.Fetch(runner)
		if err != nil {
			continue
		}
		all = append(all, items...)
	}
	return all
}

// BookmarkSource loads all bookmarks (local + remote).
type BookmarkSource struct{}

func (s BookmarkSource) Fetch(runner Runner) ([]Item, error) {
	output, err := runner(jj.BookmarkListAll())
	if err != nil {
		return nil, err
	}
	bookmarks := jj.ParseBookmarkListOutput(string(output))
	var items []Item
	for _, b := range bookmarks {
		if b.Name == "" {
			continue
		}
		if b.Local != nil {
			items = append(items, Item{Name: b.Name, Kind: KindBookmark})
		}
		for _, remote := range b.Remotes {
			items = append(items, Item{Name: fmt.Sprintf("%s@%s", b.Name, remote.Remote), Kind: KindBookmark})
		}
	}
	return items, nil
}

// TagSource loads all tags.
type TagSource struct{}

func (s TagSource) Fetch(runner Runner) ([]Item, error) {
	output, err := runner(jj.TagList())
	if err != nil {
		return nil, err
	}
	names := ParseTagListOutput(string(output))
	items := make([]Item, len(names))
	for i, name := range names {
		items[i] = Item{Name: name, Kind: KindTag}
	}
	return items, nil
}

// ParseTagListOutput parses the output of `jj tag list` into tag names.
func ParseTagListOutput(output string) []string {
	var names []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
