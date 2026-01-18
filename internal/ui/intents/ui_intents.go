package intents

type Undo struct{}

func (Undo) isIntent() {}

type Redo struct{}

func (Redo) isIntent() {}

type ExecJJ struct{}

func (ExecJJ) isIntent() {}

type ExecShell struct{}

func (ExecShell) isIntent() {}

type Quit struct{}

func (Quit) isIntent() {}

type Suspend struct{}

func (Suspend) isIntent() {}

type HelpToggle struct{}

func (HelpToggle) isIntent() {}

type OpenBookmarks struct{}

func (OpenBookmarks) isIntent() {}

type OpenGit struct{}

func (OpenGit) isIntent() {}

type BookmarksSet struct{}

func (BookmarksSet) isIntent() {}

type BookmarksFilterKind string

const (
	BookmarksFilterMove    BookmarksFilterKind = "move"
	BookmarksFilterDelete  BookmarksFilterKind = "delete"
	BookmarksFilterForget  BookmarksFilterKind = "forget"
	BookmarksFilterTrack   BookmarksFilterKind = "track"
	BookmarksFilterUntrack BookmarksFilterKind = "untrack"
)

type BookmarksFilter struct {
	Kind BookmarksFilterKind
}

func (BookmarksFilter) isIntent() {}

type BookmarksCycleRemotes struct {
	Delta int
}

func (BookmarksCycleRemotes) isIntent() {}

type BookmarksApplyShortcut struct {
	Key string
}

func (BookmarksApplyShortcut) isIntent() {}

type GitFilterKind string

const (
	GitFilterPush  GitFilterKind = "push"
	GitFilterFetch GitFilterKind = "fetch"
)

type GitFilter struct {
	Kind GitFilterKind
}

func (GitFilter) isIntent() {}

type GitCycleRemotes struct {
	Delta int
}

func (GitCycleRemotes) isIntent() {}

type GitApplyShortcut struct {
	Key string
}

func (GitApplyShortcut) isIntent() {}

type OpenCustomCommands struct{}

func (OpenCustomCommands) isIntent() {}

type OpenLeader struct{}

func (OpenLeader) isIntent() {}
