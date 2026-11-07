package revset

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBookmarksRevset(t *testing.T) {
	tests := []struct {
		filter, remote, want string
	}{
		{"", remoteAll, `bookmarks() | remote_bookmarks()`},
		{"fix", remoteAll, `bookmarks(substring-i:"fix") | remote_bookmarks(substring-i:"fix")`},
		{"", remoteLocal, `bookmarks()`},
		{"fix", remoteLocal, `bookmarks(substring-i:"fix")`},
		{"", "origin", `remote_bookmarks(remote=exact:"origin")`},
		{"fix", "origin", `remote_bookmarks(substring-i:"fix", remote=exact:"origin")`},
		{`a"b\c`, remoteLocal, `bookmarks(substring-i:"a\"b\\c")`},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, bookmarksRevset(tt.filter, tt.remote), "filter=%q remote=%q", tt.filter, tt.remote)
	}
}

// openBookmarks starts the view with origin and upstream remotes and returns
// every revset the model asked for, in order.
func openBookmarks(t *testing.T) (*Model, *[]string) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin https://a\nupstream https://b\n"))

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "::@"
	model := New(ctx)
	revsets := &[]string{}
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarks{}))
	return model, revsets
}

func simulate(model *Model, revsets *[]string, cmd tea.Cmd) {
	test.SimulateModel(model, cmd, func(msg tea.Msg) {
		if r, ok := msg.(common.UpdateRevSetMsg); ok {
			*revsets = append(*revsets, string(r))
			model.context.CurrentRevset = string(r) // ui does this in the app
		}
	})
}

func TestBookmarks_CyclesThroughRemotes(t *testing.T) {
	model, revsets := openBookmarks(t)
	for range 4 {
		simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksCycleRemote{Delta: 1}))
	}
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksCycleRemote{Delta: -1}))

	assert.Equal(t, []string{
		`bookmarks() | remote_bookmarks()`,
		`bookmarks()`,
		`remote_bookmarks(remote=exact:"origin")`,
		`remote_bookmarks(remote=exact:"upstream")`,
		`bookmarks() | remote_bookmarks()`,
		`remote_bookmarks(remote=exact:"upstream")`,
	}, *revsets)
	assert.True(t, model.InBookmarksView(), "its own revsets must not end the view")
}

func TestBookmarks_FilterAppliesWhileTyping(t *testing.T) {
	model, revsets := openBookmarks(t)
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksCycleRemote{Delta: 2}))
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilter{}))
	require.True(t, model.IsEditing())
	simulate(model, revsets, test.Type("fi"))
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilterAccept{}))

	assert.False(t, model.IsEditing())
	assert.Equal(t, []string{
		`remote_bookmarks(substring-i:"f", remote=exact:"origin")`,
		`remote_bookmarks(substring-i:"fi", remote=exact:"origin")`,
	}, (*revsets)[2:])
	assert.Empty(t, model.History, "generated revsets stay out of the history")
}

func TestBookmarks_FilterCancelRestoresPreviousFilter(t *testing.T) {
	model, revsets := openBookmarks(t)
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilter{}))
	simulate(model, revsets, test.Type("a"))
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilterAccept{}))
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilter{}))
	simulate(model, revsets, test.Type("b"))
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilterCancel{}))

	assert.Equal(t, `bookmarks(substring-i:"a") | remote_bookmarks(substring-i:"a")`, (*revsets)[len(*revsets)-1])
}

func TestBookmarks_ExitRestoresRevset(t *testing.T) {
	model, revsets := openBookmarks(t)
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksCycleRemote{Delta: 1}))
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksExit{}))

	assert.Equal(t, "::@", (*revsets)[len(*revsets)-1])
	assert.False(t, model.InBookmarksView())
}

func TestBookmarks_OtherRevsetEndsView(t *testing.T) {
	model, _ := openBookmarks(t)
	model.Update(common.UpdateRevSetMsg("trunk()"))
	assert.False(t, model.InBookmarksView())
}

func TestBookmarks_ViewShowsRemotesAndFilter(t *testing.T) {
	model, revsets := openBookmarks(t)
	simulate(model, revsets, intents.Invoke(intents.RevsetBookmarksFilter{}))
	simulate(model, revsets, test.Type("fix"))

	lines := strings.Split(test.RenderImmediate(model, 160, 2), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `revset: bookmarks(substring-i:"fix") | remote_bookmarks(substring-i:"fix")`)
	for _, s := range []string{"remote:", "all", "local", "origin", "upstream", "filter: fix"} {
		assert.Contains(t, lines[1], s)
	}
	assert.Equal(t, 2, model.Height())
}

func TestBookmarks_ClickSelectsRemote(t *testing.T) {
	model, revsets := openBookmarks(t)
	simulate(model, revsets, func() tea.Msg { return selectRemoteMsg{index: 3} })
	assert.Equal(t, `remote_bookmarks(remote=exact:"upstream")`, (*revsets)[len(*revsets)-1])
}
