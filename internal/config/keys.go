package config

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

var DefaultKeyMappings = KeyMappings[keys]{
	Up:           []string{"up", "k"},
	Down:         []string{"down", "j"},
	Apply:        []string{"enter"},
	Cancel:       []string{"esc"},
	ToggleSelect: []string{" "},
	New:          []string{"n"},
	Refresh:      []string{"ctrl+r"},
	Quit:         []string{"q"},
	Undo:         []string{"u"},
	Describe:     []string{"D"},
	Abandon:      []string{"a"},
	Edit:         []string{"e"},
	Diff:         []string{"d"},
	Diffedit:     []string{"E"},
	Absorb:       []string{"A"},
	Split:        []string{"s"},
	Squash:       []string{"S"},
	Evolog:       []string{"v"},
	Help:         []string{"?"},
	Revset:       []string{"L"},
	QuickSearch:  []string{"/"},
	Rebase: rebaseModeKeys[keys]{
		Mode:     []string{"r"},
		Revision: []string{"r"},
		Source:   []string{"s"},
		Branch:   []string{"B"},
		After:    []string{"a"},
		Before:   []string{"b"},
		Onto:     []string{"d"},
	},
	Details: detailsModeKeys[keys]{
		Mode:         []string{"l"},
		Close:        []string{"h"},
		Split:        []string{"s"},
		Restore:      []string{"r"},
		Diff:         []string{"d"},
		ToggleSelect: []string{"m", " "},
	},
	Preview: previewModeKeys[keys]{
		Mode:         []string{"p"},
		ScrollUp:     []string{"ctrl+p"},
		ScrollDown:   []string{"ctrl+n"},
		HalfPageDown: []string{"ctrl+d"},
		HalfPageUp:   []string{"ctrl+u"},
	},
	Bookmark: bookmarkModeKeys[keys]{
		Mode:    []string{"b"},
		Set:     []string{"B"},
		Delete:  []string{"d"},
		Move:    []string{"m"},
		Forget:  []string{"f"},
		Track:   []string{"t"},
		Untrack: []string{"u"},
	},
	Git: gitModeKeys[keys]{
		Mode:  []string{"g"},
		Push:  []string{"p"},
		Fetch: []string{"f"},
	},
	OpLog: opLogModeKeys[keys]{
		Mode:    []string{"o"},
		Restore: []string{"r"},
	},
}

func Convert(m KeyMappings[keys]) KeyMappings[key.Binding] {
	return KeyMappings[key.Binding]{
		Up:           key.NewBinding(key.WithKeys(m.Up...), key.WithHelp(join(m.Up), "up")),
		Down:         key.NewBinding(key.WithKeys(m.Down...), key.WithHelp(join(m.Down), "down")),
		Apply:        key.NewBinding(key.WithKeys(m.Apply...), key.WithHelp(join(m.Apply), "apply")),
		Cancel:       key.NewBinding(key.WithKeys(m.Cancel...), key.WithHelp(join(m.Cancel), "cancel")),
		ToggleSelect: key.NewBinding(key.WithKeys(m.ToggleSelect...), key.WithHelp(join(m.ToggleSelect), "toggle selection")),
		New:          key.NewBinding(key.WithKeys(m.New...), key.WithHelp(join(m.New), "new")),
		Refresh:      key.NewBinding(key.WithKeys(m.Refresh...), key.WithHelp(join(m.Refresh), "refresh")),
		Quit:         key.NewBinding(key.WithKeys(m.Quit...), key.WithHelp(join(m.Quit), "quit")),
		Diff:         key.NewBinding(key.WithKeys(m.Diff...), key.WithHelp(join(m.Diff), "diff")),
		Describe:     key.NewBinding(key.WithKeys(m.Describe...), key.WithHelp(join(m.Describe), "describe")),
		Undo:         key.NewBinding(key.WithKeys(m.Undo...), key.WithHelp(join(m.Undo), "undo")),
		Abandon:      key.NewBinding(key.WithKeys(m.Abandon...), key.WithHelp(join(m.Abandon), "abandon")),
		Edit:         key.NewBinding(key.WithKeys(m.Edit...), key.WithHelp(join(m.Edit), "edit")),
		Diffedit:     key.NewBinding(key.WithKeys(m.Diffedit...), key.WithHelp(join(m.Diffedit), "diff edit")),
		Absorb:       key.NewBinding(key.WithKeys(m.Absorb...), key.WithHelp(join(m.Absorb), "absorb")),
		Split:        key.NewBinding(key.WithKeys(m.Split...), key.WithHelp(join(m.Split), "split")),
		Squash:       key.NewBinding(key.WithKeys(m.Squash...), key.WithHelp(join(m.Squash), "squash")),
		Help:         key.NewBinding(key.WithKeys(m.Help...), key.WithHelp(join(m.Help), "help")),
		Evolog:       key.NewBinding(key.WithKeys(m.Evolog...), key.WithHelp(join(m.Evolog), "evolog")),
		Revset:       key.NewBinding(key.WithKeys(m.Revset...), key.WithHelp(join(m.Revset), "revset")),
		QuickSearch:  key.NewBinding(key.WithKeys(m.QuickSearch...), key.WithHelp(join(m.QuickSearch), "quick search")),
		Rebase: rebaseModeKeys[key.Binding]{
			Mode:     key.NewBinding(key.WithKeys(m.Rebase.Mode...), key.WithHelp(join(m.Rebase.Mode), "rebase")),
			Revision: key.NewBinding(key.WithKeys(m.Rebase.Revision...), key.WithHelp(join(m.Rebase.Revision), "change source to revision")),
			Source:   key.NewBinding(key.WithKeys(m.Rebase.Source...), key.WithHelp(join(m.Rebase.Source), "change source to descendants")),
			Branch:   key.NewBinding(key.WithKeys(m.Rebase.Branch...), key.WithHelp(join(m.Rebase.Branch), "change source to branch")),
			After:    key.NewBinding(key.WithKeys(m.Rebase.After...), key.WithHelp(join(m.Rebase.After), "change target to after")),
			Before:   key.NewBinding(key.WithKeys(m.Rebase.Before...), key.WithHelp(join(m.Rebase.Before), "change target to before")),
			Onto:     key.NewBinding(key.WithKeys(m.Rebase.Onto...), key.WithHelp(join(m.Rebase.Onto), "change target to onto")),
		},
		Details: detailsModeKeys[key.Binding]{
			Mode:         key.NewBinding(key.WithKeys(m.Details.Mode...), key.WithHelp(join(m.Details.Mode), "details")),
			Close:        key.NewBinding(key.WithKeys(m.Details.Close...), key.WithHelp(join(m.Details.Close), "close")),
			Split:        key.NewBinding(key.WithKeys(m.Details.Split...), key.WithHelp(join(m.Details.Split), "details split")),
			Restore:      key.NewBinding(key.WithKeys(m.Details.Restore...), key.WithHelp(join(m.Details.Restore), "details restore")),
			Diff:         key.NewBinding(key.WithKeys(m.Details.Diff...), key.WithHelp(join(m.Details.Diff), "details diff")),
			ToggleSelect: key.NewBinding(key.WithKeys(m.Details.ToggleSelect...), key.WithHelp(join(m.Details.ToggleSelect), "details toggle select")),
		},
		Bookmark: bookmarkModeKeys[key.Binding]{
			Mode:    key.NewBinding(key.WithKeys(m.Bookmark.Mode...), key.WithHelp(join(m.Bookmark.Mode), "bookmarks")),
			Set:     key.NewBinding(key.WithKeys(m.Bookmark.Set...), key.WithHelp(join(m.Bookmark.Set), "set bookmark")),
			Delete:  key.NewBinding(key.WithKeys(m.Bookmark.Delete...), key.WithHelp(join(m.Bookmark.Delete), "delete")),
			Move:    key.NewBinding(key.WithKeys(m.Bookmark.Move...), key.WithHelp(join(m.Bookmark.Move), "move")),
			Forget:  key.NewBinding(key.WithKeys(m.Bookmark.Forget...), key.WithHelp(join(m.Bookmark.Forget), "forget")),
			Track:   key.NewBinding(key.WithKeys(m.Bookmark.Track...), key.WithHelp(join(m.Bookmark.Track), "track")),
			Untrack: key.NewBinding(key.WithKeys(m.Bookmark.Untrack...), key.WithHelp(join(m.Bookmark.Untrack), "untrack")),
		},
		Preview: previewModeKeys[key.Binding]{
			Mode:         key.NewBinding(key.WithKeys(m.Preview.Mode...), key.WithHelp(join(m.Preview.Mode), "preview")),
			ScrollUp:     key.NewBinding(key.WithKeys(m.Preview.ScrollUp...), key.WithHelp(join(m.Preview.ScrollUp), "preview scroll up")),
			ScrollDown:   key.NewBinding(key.WithKeys(m.Preview.ScrollDown...), key.WithHelp(join(m.Preview.ScrollDown), "preview scroll down")),
			HalfPageDown: key.NewBinding(key.WithKeys(m.Preview.HalfPageDown...), key.WithHelp(join(m.Preview.HalfPageDown), "preview half page down")),
			HalfPageUp:   key.NewBinding(key.WithKeys(m.Preview.HalfPageUp...), key.WithHelp(join(m.Preview.HalfPageUp), "preview half page up")),
		},
		Git: gitModeKeys[key.Binding]{
			Mode:  key.NewBinding(key.WithKeys(m.Git.Mode...), key.WithHelp(join(m.Git.Mode), "git")),
			Push:  key.NewBinding(key.WithKeys(m.Git.Push...), key.WithHelp(join(m.Git.Push), "git push")),
			Fetch: key.NewBinding(key.WithKeys(m.Git.Fetch...), key.WithHelp(join(m.Git.Fetch), "git fetch")),
		},
		OpLog: opLogModeKeys[key.Binding]{
			Mode:    key.NewBinding(key.WithKeys(m.OpLog.Mode...), key.WithHelp(join(m.OpLog.Mode), "oplog")),
			Restore: key.NewBinding(key.WithKeys(m.OpLog.Restore...), key.WithHelp(join(m.OpLog.Restore), "restore")),
		},
	}
}

func (c *Config) GetKeyMap() KeyMappings[key.Binding] {
	return Convert(c.Keys)
}

func join(keys []string) string {
	var joined []string
	for _, key := range keys {
		k := key
		switch key {
		case "up":
			k = "↑"
		case "down":
			k = "↓"
		case " ":
			k = "space"
		}
		joined = append(joined, k)
	}
	return strings.Join(joined, "/")
}

type keys []string

type KeyMappings[T any] struct {
	Up           T                   `toml:"up"`
	Down         T                   `toml:"down"`
	Apply        T                   `toml:"apply"`
	Cancel       T                   `toml:"cancel"`
	ToggleSelect T                   `toml:"toggle_select"`
	New          T                   `toml:"new"`
	Refresh      T                   `toml:"refresh"`
	Abandon      T                   `toml:"abandon"`
	Diff         T                   `toml:"diff"`
	Quit         T                   `toml:"quit"`
	Help         T                   `toml:"help"`
	Describe     T                   `toml:"describe"`
	Edit         T                   `toml:"edit"`
	Diffedit     T                   `toml:"diffedit"`
	Absorb       T                   `toml:"absorb"`
	Split        T                   `toml:"split"`
	Squash       T                   `toml:"squash"`
	Undo         T                   `toml:"undo"`
	Evolog       T                   `toml:"evolog"`
	Revset       T                   `toml:"revset"`
	QuickSearch  T                   `toml:"quick_search"`
	Rebase       rebaseModeKeys[T]   `toml:"rebase"`
	Details      detailsModeKeys[T]  `toml:"details"`
	Preview      previewModeKeys[T]  `toml:"preview"`
	Bookmark     bookmarkModeKeys[T] `toml:"bookmark"`
	Git          gitModeKeys[T]      `toml:"git"`
	OpLog        opLogModeKeys[T]    `toml:"oplog"`
}

type bookmarkModeKeys[T any] struct {
	Mode    T `toml:"mode"`
	Set     T `toml:"set"`
	Delete  T `toml:"delete"`
	Move    T `toml:"move"`
	Forget  T `toml:"forget"`
	Track   T `toml:"track"`
	Untrack T `toml:"untrack"`
}

type rebaseModeKeys[T any] struct {
	Mode     T `toml:"mode"`
	Revision T `toml:"revision"`
	Source   T `toml:"source"`
	Branch   T `toml:"branch"`
	After    T `toml:"after"`
	Before   T `toml:"before"`
	Onto     T `toml:"onto"`
}

type detailsModeKeys[T any] struct {
	Mode         T `toml:"mode"`
	Close        T `toml:"close"`
	Split        T `toml:"split"`
	Restore      T `toml:"restore"`
	Diff         T `toml:"diff"`
	ToggleSelect T `toml:"select"`
}

type gitModeKeys[T any] struct {
	Mode  T `toml:"mode"`
	Push  T `toml:"push"`
	Fetch T `toml:"fetch"`
}

type previewModeKeys[T any] struct {
	Mode         T `toml:"mode"`
	ScrollUp     T `toml:"scroll_up"`
	ScrollDown   T `toml:"scroll_down"`
	HalfPageDown T `toml:"half_page_down"`
	HalfPageUp   T `toml:"half_page_up"`
}

type opLogModeKeys[T any] struct {
	Mode    T `toml:"mode"`
	Restore T `toml:"restore"`
}
