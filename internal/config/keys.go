package config

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

var DefaultKeyMappings = KeyMappings[keys]{
	Up:                []string{"up", "k"},
	Down:              []string{"down", "j"},
	JumpToParent:      []string{"J"},
	JumpToWorkingCopy: []string{"@"},
	Apply:             []string{"enter"},
	Cancel:            []string{"esc"},
	ToggleSelect:      []string{" "},
	New:               []string{"n"},
	Commit:            []string{"c"},
	Refresh:           []string{"ctrl+r"},
	Quit:              []string{"q"},
	Undo:              []string{"u"},
	Describe:          []string{"D"},
	Abandon:           []string{"a"},
	Edit:              []string{"e"},
	Diff:              []string{"d"},
	Diffedit:          []string{"E"},
	Absorb:            []string{"A"},
	Split:             []string{"s"},
	Evolog:            []string{"v"},
	Help:              []string{"?"},
	Revset:            []string{"L"},
	QuickSearch:       []string{"/"},
	QuickSearchCycle:  []string{"'"},
	CustomCommands:    []string{"x"},
	Suspend:           []string{"ctrl+z"},
	Rebase: rebaseModeKeys[keys]{
		Mode:     []string{"r"},
		Revision: []string{"r"},
		Source:   []string{"s"},
		Branch:   []string{"B"},
		After:    []string{"a"},
		Before:   []string{"b"},
		Onto:     []string{"d"},
		Insert:   []string{"i"},
	},
	Squash: squashModeKeys[keys]{
		Mode:        []string{"S"},
		KeepEmptied: []string{"e"},
		Interactive: []string{"i"},
	},
	Details: detailsModeKeys[keys]{
		Mode:                  []string{"l"},
		Close:                 []string{"h"},
		Split:                 []string{"s"},
		Restore:               []string{"r"},
		Absorb:                []string{"A"},
		Diff:                  []string{"d"},
		ToggleSelect:          []string{"m", " "},
		RevisionsChangingFile: []string{"*"},
	},
	Preview: previewModeKeys[keys]{
		Mode:         []string{"p"},
		ScrollUp:     []string{"ctrl+p"},
		ScrollDown:   []string{"ctrl+n"},
		HalfPageDown: []string{"ctrl+d"},
		HalfPageUp:   []string{"ctrl+u"},
		Expand:       []string{"ctrl+h"},
		Shrink:       []string{"ctrl+l"},
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
	InlineDescribe: inlineDescribeModeKeys[keys]{
		Mode:   []string{"enter"},
		Accept: []string{"alt+enter", "ctrl+s"},
	},
}

func Convert(m KeyMappings[keys]) KeyMappings[key.Binding] {
	return KeyMappings[key.Binding]{
		Up:                key.NewBinding(key.WithKeys(m.Up...), key.WithHelp(JoinKeys(m.Up), "up")),
		Down:              key.NewBinding(key.WithKeys(m.Down...), key.WithHelp(JoinKeys(m.Down), "down")),
		JumpToParent:      key.NewBinding(key.WithKeys(m.JumpToParent...), key.WithHelp(JoinKeys(m.JumpToParent), "jump to parent")),
		JumpToWorkingCopy: key.NewBinding(key.WithKeys(m.JumpToWorkingCopy...), key.WithHelp(JoinKeys(m.JumpToWorkingCopy), "jump to working copy")),
		Apply:             key.NewBinding(key.WithKeys(m.Apply...), key.WithHelp(JoinKeys(m.Apply), "apply")),
		Cancel:            key.NewBinding(key.WithKeys(m.Cancel...), key.WithHelp(JoinKeys(m.Cancel), "cancel")),
		ToggleSelect:      key.NewBinding(key.WithKeys(m.ToggleSelect...), key.WithHelp(JoinKeys(m.ToggleSelect), "toggle selection")),
		New:               key.NewBinding(key.WithKeys(m.New...), key.WithHelp(JoinKeys(m.New), "new")),
		Commit:            key.NewBinding(key.WithKeys(m.Commit...), key.WithHelp(JoinKeys(m.Commit), "commit")),
		Refresh:           key.NewBinding(key.WithKeys(m.Refresh...), key.WithHelp(JoinKeys(m.Refresh), "refresh")),
		Quit:              key.NewBinding(key.WithKeys(m.Quit...), key.WithHelp(JoinKeys(m.Quit), "quit")),
		Diff:              key.NewBinding(key.WithKeys(m.Diff...), key.WithHelp(JoinKeys(m.Diff), "diff")),
		Describe:          key.NewBinding(key.WithKeys(m.Describe...), key.WithHelp(JoinKeys(m.Describe), "describe")),
		Undo:              key.NewBinding(key.WithKeys(m.Undo...), key.WithHelp(JoinKeys(m.Undo), "undo")),
		Abandon:           key.NewBinding(key.WithKeys(m.Abandon...), key.WithHelp(JoinKeys(m.Abandon), "abandon")),
		Edit:              key.NewBinding(key.WithKeys(m.Edit...), key.WithHelp(JoinKeys(m.Edit), "edit")),
		Diffedit:          key.NewBinding(key.WithKeys(m.Diffedit...), key.WithHelp(JoinKeys(m.Diffedit), "diff edit")),
		Absorb:            key.NewBinding(key.WithKeys(m.Absorb...), key.WithHelp(JoinKeys(m.Absorb), "absorb")),
		Split:             key.NewBinding(key.WithKeys(m.Split...), key.WithHelp(JoinKeys(m.Split), "split")),
		Help:              key.NewBinding(key.WithKeys(m.Help...), key.WithHelp(JoinKeys(m.Help), "help")),
		Evolog:            key.NewBinding(key.WithKeys(m.Evolog...), key.WithHelp(JoinKeys(m.Evolog), "evolog")),
		Revset:            key.NewBinding(key.WithKeys(m.Revset...), key.WithHelp(JoinKeys(m.Revset), "revset")),
		QuickSearch:       key.NewBinding(key.WithKeys(m.QuickSearch...), key.WithHelp(JoinKeys(m.QuickSearch), "quick search")),
		QuickSearchCycle:  key.NewBinding(key.WithKeys(m.QuickSearchCycle...), key.WithHelp(JoinKeys(m.QuickSearchCycle), "locate next match")),
		CustomCommands:    key.NewBinding(key.WithKeys(m.CustomCommands...), key.WithHelp(JoinKeys(m.CustomCommands), "custom commands menu")),
		Suspend:           key.NewBinding(key.WithKeys(m.Suspend...), key.WithHelp(JoinKeys(m.Suspend), "suspend")),
		Rebase: rebaseModeKeys[key.Binding]{
			Mode:     key.NewBinding(key.WithKeys(m.Rebase.Mode...), key.WithHelp(JoinKeys(m.Rebase.Mode), "rebase")),
			Revision: key.NewBinding(key.WithKeys(m.Rebase.Revision...), key.WithHelp(JoinKeys(m.Rebase.Revision), "revision")),
			Source:   key.NewBinding(key.WithKeys(m.Rebase.Source...), key.WithHelp(JoinKeys(m.Rebase.Source), "source")),
			Branch:   key.NewBinding(key.WithKeys(m.Rebase.Branch...), key.WithHelp(JoinKeys(m.Rebase.Branch), "branch")),
			After:    key.NewBinding(key.WithKeys(m.Rebase.After...), key.WithHelp(JoinKeys(m.Rebase.After), "insert after")),
			Before:   key.NewBinding(key.WithKeys(m.Rebase.Before...), key.WithHelp(JoinKeys(m.Rebase.Before), "insert before")),
			Onto:     key.NewBinding(key.WithKeys(m.Rebase.Onto...), key.WithHelp(JoinKeys(m.Rebase.Onto), "onto")),
			Insert:   key.NewBinding(key.WithKeys(m.Rebase.Insert...), key.WithHelp(JoinKeys(m.Rebase.Insert), "insert between")),
		},
		Squash: squashModeKeys[key.Binding]{
			Mode:        key.NewBinding(key.WithKeys(m.Squash.Mode...), key.WithHelp(JoinKeys(m.Squash.Mode), "squash")),
			KeepEmptied: key.NewBinding(key.WithKeys(m.Squash.KeepEmptied...), key.WithHelp(JoinKeys(m.Squash.KeepEmptied), "keep emptied commits")),
			Interactive: key.NewBinding(key.WithKeys(m.Squash.Interactive...), key.WithHelp(JoinKeys(m.Squash.Interactive), "interactive")),
		},
		Details: detailsModeKeys[key.Binding]{
			Mode:                  key.NewBinding(key.WithKeys(m.Details.Mode...), key.WithHelp(JoinKeys(m.Details.Mode), "details")),
			Close:                 key.NewBinding(key.WithKeys(m.Details.Close...), key.WithHelp(JoinKeys(m.Details.Close), "close")),
			Split:                 key.NewBinding(key.WithKeys(m.Details.Split...), key.WithHelp(JoinKeys(m.Details.Split), "split")),
			Restore:               key.NewBinding(key.WithKeys(m.Details.Restore...), key.WithHelp(JoinKeys(m.Details.Restore), "restore")),
			Absorb:                key.NewBinding(key.WithKeys(m.Details.Absorb...), key.WithHelp(JoinKeys(m.Details.Absorb), "absorb")),
			Diff:                  key.NewBinding(key.WithKeys(m.Details.Diff...), key.WithHelp(JoinKeys(m.Details.Diff), "diff")),
			ToggleSelect:          key.NewBinding(key.WithKeys(m.Details.ToggleSelect...), key.WithHelp(JoinKeys(m.Details.ToggleSelect), "details toggle select")),
			RevisionsChangingFile: key.NewBinding(key.WithKeys(m.Details.RevisionsChangingFile...), key.WithHelp(JoinKeys(m.Details.RevisionsChangingFile), "show revisions changing file")),
		},
		Bookmark: bookmarkModeKeys[key.Binding]{
			Mode:    key.NewBinding(key.WithKeys(m.Bookmark.Mode...), key.WithHelp(JoinKeys(m.Bookmark.Mode), "bookmarks")),
			Set:     key.NewBinding(key.WithKeys(m.Bookmark.Set...), key.WithHelp(JoinKeys(m.Bookmark.Set), "set bookmark")),
			Delete:  key.NewBinding(key.WithKeys(m.Bookmark.Delete...), key.WithHelp(JoinKeys(m.Bookmark.Delete), "delete")),
			Move:    key.NewBinding(key.WithKeys(m.Bookmark.Move...), key.WithHelp(JoinKeys(m.Bookmark.Move), "move")),
			Forget:  key.NewBinding(key.WithKeys(m.Bookmark.Forget...), key.WithHelp(JoinKeys(m.Bookmark.Forget), "forget")),
			Track:   key.NewBinding(key.WithKeys(m.Bookmark.Track...), key.WithHelp(JoinKeys(m.Bookmark.Track), "track")),
			Untrack: key.NewBinding(key.WithKeys(m.Bookmark.Untrack...), key.WithHelp(JoinKeys(m.Bookmark.Untrack), "untrack")),
		},
		Preview: previewModeKeys[key.Binding]{
			Mode:         key.NewBinding(key.WithKeys(m.Preview.Mode...), key.WithHelp(JoinKeys(m.Preview.Mode), "preview")),
			ScrollUp:     key.NewBinding(key.WithKeys(m.Preview.ScrollUp...), key.WithHelp(JoinKeys(m.Preview.ScrollUp), "preview scroll up")),
			ScrollDown:   key.NewBinding(key.WithKeys(m.Preview.ScrollDown...), key.WithHelp(JoinKeys(m.Preview.ScrollDown), "preview scroll down")),
			HalfPageDown: key.NewBinding(key.WithKeys(m.Preview.HalfPageDown...), key.WithHelp(JoinKeys(m.Preview.HalfPageDown), "preview half page down")),
			HalfPageUp:   key.NewBinding(key.WithKeys(m.Preview.HalfPageUp...), key.WithHelp(JoinKeys(m.Preview.HalfPageUp), "preview half page up")),
			Expand:       key.NewBinding(key.WithKeys(m.Preview.Expand...), key.WithHelp(JoinKeys(m.Preview.Expand), "expand width")),
			Shrink:       key.NewBinding(key.WithKeys(m.Preview.Shrink...), key.WithHelp(JoinKeys(m.Preview.Shrink), "shrink width")),
		},
		Git: gitModeKeys[key.Binding]{
			Mode:  key.NewBinding(key.WithKeys(m.Git.Mode...), key.WithHelp(JoinKeys(m.Git.Mode), "git")),
			Push:  key.NewBinding(key.WithKeys(m.Git.Push...), key.WithHelp(JoinKeys(m.Git.Push), "git push")),
			Fetch: key.NewBinding(key.WithKeys(m.Git.Fetch...), key.WithHelp(JoinKeys(m.Git.Fetch), "git fetch")),
		},
		OpLog: opLogModeKeys[key.Binding]{
			Mode:    key.NewBinding(key.WithKeys(m.OpLog.Mode...), key.WithHelp(JoinKeys(m.OpLog.Mode), "oplog")),
			Restore: key.NewBinding(key.WithKeys(m.OpLog.Restore...), key.WithHelp(JoinKeys(m.OpLog.Restore), "restore")),
		},
		InlineDescribe: inlineDescribeModeKeys[key.Binding]{
			Mode:   key.NewBinding(key.WithKeys(m.InlineDescribe.Mode...), key.WithHelp(JoinKeys(m.InlineDescribe.Mode), "inline describe")),
			Accept: key.NewBinding(key.WithKeys(m.InlineDescribe.Accept...), key.WithHelp(JoinKeys(m.InlineDescribe.Accept), "accept")),
		},
	}
}

func (c *Config) GetKeyMap() KeyMappings[key.Binding] {
	return Convert(c.Keys)
}

func JoinKeys(keys []string) string {
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
	Up                T                         `toml:"up"`
	Down              T                         `toml:"down"`
	JumpToParent      T                         `toml:"jump_to_parent"`
	JumpToWorkingCopy T                         `toml:"jump_to_working_copy"`
	Apply             T                         `toml:"apply"`
	Cancel            T                         `toml:"cancel"`
	ToggleSelect      T                         `toml:"toggle_select"`
	New               T                         `toml:"new"`
	Commit            T                         `toml:"commit"`
	Refresh           T                         `toml:"refresh"`
	Abandon           T                         `toml:"abandon"`
	Diff              T                         `toml:"diff"`
	Quit              T                         `toml:"quit"`
	Help              T                         `toml:"help"`
	Describe          T                         `toml:"describe"`
	Edit              T                         `toml:"edit"`
	Diffedit          T                         `toml:"diffedit"`
	Absorb            T                         `toml:"absorb"`
	Split             T                         `toml:"split"`
	Undo              T                         `toml:"undo"`
	Evolog            T                         `toml:"evolog"`
	Revset            T                         `toml:"revset"`
	QuickSearch       T                         `toml:"quick_search"`
	QuickSearchCycle  T                         `toml:"quick_search_cycle"`
	CustomCommands    T                         `toml:"custom_commands"`
	Suspend           T                         `toml:"suspend"`
	Rebase            rebaseModeKeys[T]         `toml:"rebase"`
	Squash            squashModeKeys[T]         `toml:"squash"`
	Details           detailsModeKeys[T]        `toml:"details"`
	Preview           previewModeKeys[T]        `toml:"preview"`
	Bookmark          bookmarkModeKeys[T]       `toml:"bookmark"`
	InlineDescribe    inlineDescribeModeKeys[T] `toml:"inline_describe"`
	Git               gitModeKeys[T]            `toml:"git"`
	OpLog             opLogModeKeys[T]          `toml:"oplog"`
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

type squashModeKeys[T any] struct {
	Mode        T `toml:"mode"`
	KeepEmptied T `toml:"keep_emptied"`
	Interactive T `toml:"interactive"`
}

type rebaseModeKeys[T any] struct {
	Mode     T `toml:"mode"`
	Revision T `toml:"revision"`
	Source   T `toml:"source"`
	Branch   T `toml:"branch"`
	After    T `toml:"after"`
	Before   T `toml:"before"`
	Onto     T `toml:"onto"`
	Insert   T `toml:"insert"`
}

type detailsModeKeys[T any] struct {
	Mode                  T `toml:"mode"`
	Close                 T `toml:"close"`
	Split                 T `toml:"split"`
	Restore               T `toml:"restore"`
	Absorb                T `toml:"absorb"`
	Diff                  T `toml:"diff"`
	ToggleSelect          T `toml:"select"`
	RevisionsChangingFile T `toml:"revisions_changing_file"`
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
	Expand       T `toml:"expand"`
	Shrink       T `toml:"shrink"`
}

type opLogModeKeys[T any] struct {
	Mode    T `toml:"mode"`
	Restore T `toml:"restore"`
}

type inlineDescribeModeKeys[T any] struct {
	Mode   T `toml:"mode"`
	Accept T `toml:"accept"`
}
