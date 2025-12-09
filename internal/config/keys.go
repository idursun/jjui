package config

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

func Convert(m KeyMappings[keys]) KeyMappings[key.Binding] {
	return KeyMappings[key.Binding]{
		Up:                key.NewBinding(key.WithKeys(m.Up...), key.WithHelp(JoinKeys(m.Up), "up")),
		Down:              key.NewBinding(key.WithKeys(m.Down...), key.WithHelp(JoinKeys(m.Down), "down")),
		ScrollUp:          key.NewBinding(key.WithKeys(m.ScrollUp...), key.WithHelp(JoinKeys(m.ScrollUp), "scroll up")),
		ScrollDown:        key.NewBinding(key.WithKeys(m.ScrollDown...), key.WithHelp(JoinKeys(m.ScrollDown), "scroll down")),
		JumpToParent:      key.NewBinding(key.WithKeys(m.JumpToParent...), key.WithHelp(JoinKeys(m.JumpToParent), "jump to parent")),
		JumpToChildren:    key.NewBinding(key.WithKeys(m.JumpToChildren...), key.WithHelp(JoinKeys(m.JumpToChildren), "jump to children")),
		JumpToWorkingCopy: key.NewBinding(key.WithKeys(m.JumpToWorkingCopy...), key.WithHelp(JoinKeys(m.JumpToWorkingCopy), "jump to working copy")),
		Apply:             key.NewBinding(key.WithKeys(m.Apply...), key.WithHelp(JoinKeys(m.Apply), "apply")),
		ForceApply:        key.NewBinding(key.WithKeys(m.ForceApply...), key.WithHelp(JoinKeys(m.ForceApply), "force apply")),
		Cancel:            key.NewBinding(key.WithKeys(m.Cancel...), key.WithHelp(JoinKeys(m.Cancel), "cancel")),
		ToggleSelect:      key.NewBinding(key.WithKeys(m.ToggleSelect...), key.WithHelp(JoinKeys(m.ToggleSelect), "toggle selection")),
		New:               key.NewBinding(key.WithKeys(m.New...), key.WithHelp(JoinKeys(m.New), "new")),
		Commit:            key.NewBinding(key.WithKeys(m.Commit...), key.WithHelp(JoinKeys(m.Commit), "commit")),
		Refresh:           key.NewBinding(key.WithKeys(m.Refresh...), key.WithHelp(JoinKeys(m.Refresh), "refresh")),
		Quit:              key.NewBinding(key.WithKeys(m.Quit...), key.WithHelp(JoinKeys(m.Quit), "quit")),
		Diff:              key.NewBinding(key.WithKeys(m.Diff...), key.WithHelp(JoinKeys(m.Diff), "diff")),
		Describe:          key.NewBinding(key.WithKeys(m.Describe...), key.WithHelp(JoinKeys(m.Describe), "describe")),
		Undo:              key.NewBinding(key.WithKeys(m.Undo...), key.WithHelp(JoinKeys(m.Undo), "undo")),
		Redo:              key.NewBinding(key.WithKeys(m.Redo...), key.WithHelp(JoinKeys(m.Redo), "redo")),
		Abandon:           key.NewBinding(key.WithKeys(m.Abandon...), key.WithHelp(JoinKeys(m.Abandon), "abandon")),
		Edit:              key.NewBinding(key.WithKeys(m.Edit...), key.WithHelp(JoinKeys(m.Edit), "edit")),
		ForceEdit:         key.NewBinding(key.WithKeys(m.ForceEdit...), key.WithHelp(JoinKeys(m.ForceEdit), "force edit")),
		Diffedit:          key.NewBinding(key.WithKeys(m.Diffedit...), key.WithHelp(JoinKeys(m.Diffedit), "diff edit")),
		Absorb:            key.NewBinding(key.WithKeys(m.Absorb...), key.WithHelp(JoinKeys(m.Absorb), "absorb")),
		Split:             key.NewBinding(key.WithKeys(m.Split...), key.WithHelp(JoinKeys(m.Split), "split")),
		SplitParallel:     key.NewBinding(key.WithKeys(m.SplitParallel...), key.WithHelp(JoinKeys(m.SplitParallel), "split (parallel)")),
		Help:              key.NewBinding(key.WithKeys(m.Help...), key.WithHelp(JoinKeys(m.Help), "help")),
		Evolog: evologModeKeys[key.Binding]{
			Mode:    key.NewBinding(key.WithKeys(m.Evolog.Mode...), key.WithHelp(JoinKeys(m.Evolog.Mode), "evolog")),
			Diff:    key.NewBinding(key.WithKeys(m.Evolog.Diff...), key.WithHelp(JoinKeys(m.Evolog.Diff), "diff")),
			Restore: key.NewBinding(key.WithKeys(m.Evolog.Restore...), key.WithHelp(JoinKeys(m.Evolog.Restore), "restore")),
		},
		Revset:           key.NewBinding(key.WithKeys(m.Revset...), key.WithHelp(JoinKeys(m.Revset), "revset")),
		AceJump:          key.NewBinding(key.WithKeys(m.AceJump...), key.WithHelp(JoinKeys(m.AceJump), "ace jump")),
		QuickSearch:      key.NewBinding(key.WithKeys(m.QuickSearch...), key.WithHelp(JoinKeys(m.QuickSearch), "quick search")),
		QuickSearchCycle: key.NewBinding(key.WithKeys(m.QuickSearchCycle...), key.WithHelp(JoinKeys(m.QuickSearchCycle), "locate next match")),
		CustomCommands:   key.NewBinding(key.WithKeys(m.CustomCommands...), key.WithHelp(JoinKeys(m.CustomCommands), "custom commands menu")),
		Leader:           key.NewBinding(key.WithKeys(m.Leader...), key.WithHelp(JoinKeys(m.Leader), "leader")),
		Suspend:          key.NewBinding(key.WithKeys(m.Suspend...), key.WithHelp(JoinKeys(m.Suspend), "suspend")),
		SetParents:       key.NewBinding(key.WithKeys(m.SetParents...), key.WithHelp(JoinKeys(m.SetParents), "set parents")),
		ExecJJ:           key.NewBinding(key.WithKeys(m.ExecJJ...), key.WithHelp(JoinKeys(m.ExecJJ), "interactive jj")),
		ExecShell:        key.NewBinding(key.WithKeys(m.ExecShell...), key.WithHelp(JoinKeys(m.ExecShell), "interactive shell command")),
		Revert: revertModeKeys[key.Binding]{
			Mode:   key.NewBinding(key.WithKeys(m.Revert.Mode...), key.WithHelp(JoinKeys(m.Revert.Mode), "revert")),
			After:  key.NewBinding(key.WithKeys(m.Revert.After...), key.WithHelp(JoinKeys(m.Revert.After), "insert after")),
			Before: key.NewBinding(key.WithKeys(m.Revert.Before...), key.WithHelp(JoinKeys(m.Revert.Before), "insert before")),
			Onto:   key.NewBinding(key.WithKeys(m.Revert.Onto...), key.WithHelp(JoinKeys(m.Revert.Onto), "onto")),
			Insert: key.NewBinding(key.WithKeys(m.Revert.Insert...), key.WithHelp(JoinKeys(m.Revert.Insert), "insert between")),
		},
		Rebase: rebaseModeKeys[key.Binding]{
			Mode:        key.NewBinding(key.WithKeys(m.Rebase.Mode...), key.WithHelp(JoinKeys(m.Rebase.Mode), "rebase")),
			Revision:    key.NewBinding(key.WithKeys(m.Rebase.Revision...), key.WithHelp(JoinKeys(m.Rebase.Revision), "revision")),
			Source:      key.NewBinding(key.WithKeys(m.Rebase.Source...), key.WithHelp(JoinKeys(m.Rebase.Source), "source")),
			Branch:      key.NewBinding(key.WithKeys(m.Rebase.Branch...), key.WithHelp(JoinKeys(m.Rebase.Branch), "branch")),
			After:       key.NewBinding(key.WithKeys(m.Rebase.After...), key.WithHelp(JoinKeys(m.Rebase.After), "insert after")),
			Before:      key.NewBinding(key.WithKeys(m.Rebase.Before...), key.WithHelp(JoinKeys(m.Rebase.Before), "insert before")),
			Onto:        key.NewBinding(key.WithKeys(m.Rebase.Onto...), key.WithHelp(JoinKeys(m.Rebase.Onto), "onto")),
			Insert:      key.NewBinding(key.WithKeys(m.Rebase.Insert...), key.WithHelp(JoinKeys(m.Rebase.Insert), "insert between")),
			SkipEmptied: key.NewBinding(key.WithKeys(m.Rebase.SkipEmptied...), key.WithHelp(JoinKeys(m.Rebase.SkipEmptied), "skip emptied")),
		},
		Duplicate: duplicateModeKeys[key.Binding]{
			Mode:   key.NewBinding(key.WithKeys(m.Duplicate.Mode...), key.WithHelp(JoinKeys(m.Duplicate.Mode), "duplicate")),
			After:  key.NewBinding(key.WithKeys(m.Duplicate.After...), key.WithHelp(JoinKeys(m.Duplicate.After), "duplicate after")),
			Before: key.NewBinding(key.WithKeys(m.Duplicate.Before...), key.WithHelp(JoinKeys(m.Duplicate.Before), "duplicate before")),
			Onto:   key.NewBinding(key.WithKeys(m.Duplicate.Onto...), key.WithHelp(JoinKeys(m.Duplicate.Onto), "duplicate onto")),
		},
		Squash: squashModeKeys[key.Binding]{
			Mode:                  key.NewBinding(key.WithKeys(m.Squash.Mode...), key.WithHelp(JoinKeys(m.Squash.Mode), "squash")),
			KeepEmptied:           key.NewBinding(key.WithKeys(m.Squash.KeepEmptied...), key.WithHelp(JoinKeys(m.Squash.KeepEmptied), "keep emptied commits")),
			UseDestinationMessage: key.NewBinding(key.WithKeys(m.Squash.UseDestinationMessage...), key.WithHelp(JoinKeys(m.Squash.UseDestinationMessage), "use destination message")),
			Interactive:           key.NewBinding(key.WithKeys(m.Squash.Interactive...), key.WithHelp(JoinKeys(m.Squash.Interactive), "interactive")),
		},
		Details: detailsModeKeys[key.Binding]{
			Mode:                  key.NewBinding(key.WithKeys(m.Details.Mode...), key.WithHelp(JoinKeys(m.Details.Mode), "details")),
			Close:                 key.NewBinding(key.WithKeys(m.Details.Close...), key.WithHelp(JoinKeys(m.Details.Close), "close")),
			Split:                 key.NewBinding(key.WithKeys(m.Details.Split...), key.WithHelp(JoinKeys(m.Details.Split), "split")),
			SplitParallel:         key.NewBinding(key.WithKeys(m.Details.SplitParallel...), key.WithHelp(JoinKeys(m.Details.SplitParallel), "split (parallel)")),
			Squash:                key.NewBinding(key.WithKeys(m.Details.Squash...), key.WithHelp(JoinKeys(m.Details.Squash), "squash")),
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
			ToggleBottom: key.NewBinding(key.WithKeys(m.Preview.ToggleBottom...), key.WithHelp(JoinKeys(m.Preview.ToggleBottom), "toggle show at bottom")),
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
			Revert:  key.NewBinding(key.WithKeys(m.OpLog.Revert...), key.WithHelp(JoinKeys(m.OpLog.Revert), "revert")),
		},
		InlineDescribe: inlineDescribeModeKeys[key.Binding]{
			Mode:   key.NewBinding(key.WithKeys(m.InlineDescribe.Mode...), key.WithHelp(JoinKeys(m.InlineDescribe.Mode), "inline describe")),
			Accept: key.NewBinding(key.WithKeys(m.InlineDescribe.Accept...), key.WithHelp(JoinKeys(m.InlineDescribe.Accept), "accept")),
			Editor: key.NewBinding(key.WithKeys(m.InlineDescribe.Editor...), key.WithHelp(JoinKeys(m.InlineDescribe.Editor), "open in editor")),
		},
		FileSearch: fileSearchKeys[key.Binding]{
			Toggle: key.NewBinding(key.WithKeys(m.FileSearch.Toggle...), key.WithHelp(JoinKeys(m.FileSearch.Toggle), "fuzzy files search")),
			Up:     key.NewBinding(key.WithKeys(m.FileSearch.Up...), key.WithHelp(JoinKeys(m.FileSearch.Up), "up")),
			Down:   key.NewBinding(key.WithKeys(m.FileSearch.Down...), key.WithHelp(JoinKeys(m.FileSearch.Down), "down")),
			Accept: key.NewBinding(key.WithKeys(m.FileSearch.Accept...), key.WithHelp(JoinKeys(m.FileSearch.Accept), "file revset")),
			Edit:   key.NewBinding(key.WithKeys(m.FileSearch.Edit...), key.WithHelp(JoinKeys(m.FileSearch.Edit), "edit file")),
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
	ScrollUp          T                         `toml:"scroll_up"`
	ScrollDown        T                         `toml:"scroll_down"`
	JumpToParent      T                         `toml:"jump_to_parent"`
	JumpToChildren    T                         `toml:"jump_to_children"`
	JumpToWorkingCopy T                         `toml:"jump_to_working_copy"`
	Apply             T                         `toml:"apply"`
	Cancel            T                         `toml:"cancel"`
	ForceApply        T                         `toml:"force_apply"`
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
	ForceEdit         T                         `toml:"force_edit"`
	Diffedit          T                         `toml:"diffedit"`
	Absorb            T                         `toml:"absorb"`
	Split             T                         `toml:"split"`
	SplitParallel     T                         `toml:"split_parallel"`
	Undo              T                         `toml:"undo"`
	Redo              T                         `toml:"redo"`
	Revset            T                         `toml:"revset"`
	ExecJJ            T                         `toml:"exec_jj"`
	ExecShell         T                         `toml:"exec_shell"`
	AceJump           T                         `toml:"ace_jump"`
	QuickSearch       T                         `toml:"quick_search"`
	QuickSearchCycle  T                         `toml:"quick_search_cycle"`
	CustomCommands    T                         `toml:"custom_commands"`
	Leader            T                         `toml:"leader"`
	Suspend           T                         `toml:"suspend"`
	SetParents        T                         `toml:"set_parents"`
	Revert            revertModeKeys[T]         `toml:"revert"`
	Rebase            rebaseModeKeys[T]         `toml:"rebase"`
	Duplicate         duplicateModeKeys[T]      `toml:"duplicate"`
	Squash            squashModeKeys[T]         `toml:"squash"`
	Details           detailsModeKeys[T]        `toml:"details"`
	Evolog            evologModeKeys[T]         `toml:"evolog"`
	Preview           previewModeKeys[T]        `toml:"preview"`
	Bookmark          bookmarkModeKeys[T]       `toml:"bookmark"`
	InlineDescribe    inlineDescribeModeKeys[T] `toml:"inline_describe"`
	Git               gitModeKeys[T]            `toml:"git"`
	OpLog             opLogModeKeys[T]          `toml:"oplog"`
	FileSearch        fileSearchKeys[T]         `toml:"file_search"`
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
	Mode                  T `toml:"mode"`
	KeepEmptied           T `toml:"keep_emptied"`
	UseDestinationMessage T `toml:"use_destination_message"`
	Interactive           T `toml:"interactive"`
}

type revertModeKeys[T any] struct {
	Mode   T `toml:"mode"`
	After  T `toml:"after"`
	Before T `toml:"before"`
	Onto   T `toml:"onto"`
	Insert T `toml:"insert"`
}

type rebaseModeKeys[T any] struct {
	Mode        T `toml:"mode"`
	Revision    T `toml:"revision"`
	Source      T `toml:"source"`
	Branch      T `toml:"branch"`
	After       T `toml:"after"`
	Before      T `toml:"before"`
	Onto        T `toml:"onto"`
	Insert      T `toml:"insert"`
	SkipEmptied T `toml:"skip_emptied"`
}

type duplicateModeKeys[T any] struct {
	Mode   T `toml:"mode"`
	After  T `toml:"after"`
	Before T `toml:"before"`
	Onto   T `toml:"onto"`
}

type evologModeKeys[T any] struct {
	Mode    T `toml:"mode"`
	Diff    T `toml:"diff"`
	Restore T `toml:"restore"`
}

type detailsModeKeys[T any] struct {
	Mode                  T `toml:"mode"`
	Close                 T `toml:"close"`
	Split                 T `toml:"split"`
	SplitParallel         T `toml:"split_parallel"`
	Restore               T `toml:"restore"`
	Absorb                T `toml:"absorb"`
	Squash                T `toml:"squash"`
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
	ToggleBottom T `toml:"toggle_bottom"`
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
	Revert  T `toml:"revert"`
}

type inlineDescribeModeKeys[T any] struct {
	Mode   T `toml:"mode"`
	Accept T `toml:"accept"`
	Editor T `toml:"editor"`
}

type fileSearchKeys[T any] struct {
	Toggle T `toml:"toggle"`
	Up     T `toml:"up"`
	Down   T `toml:"down"`
	Accept T `toml:"accept"`
	Edit   T `toml:"edit"`
}
