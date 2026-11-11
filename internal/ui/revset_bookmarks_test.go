package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevsetBookmarks_KeysWinOverRevisions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin /tmp/remote.git\n"))
	defer commandRunner.Verify()
	model := NewUI(test.NewTestContext(commandRunner))
	model.revsetModel.Update(intents.RevsetBookmarks{})
	require.True(t, model.revsetModel.InBookmarksView())

	resolve := func(key tea.KeyPressMsg) intents.Intent {
		return model.resolver.ResolveKey(key, model.dispatchScopes()).Intent
	}
	assert.Equal(t, intents.RevsetBookmarksCycleRemote{Delta: 1}, resolve(tea.KeyPressMsg{Code: tea.KeyTab}))
	assert.Equal(t, intents.RevsetBookmarksFilter{}, resolve(tea.KeyPressMsg{Code: '/', Text: "/"}))

	model.revsetModel.Update(intents.RevsetBookmarksFilter{})
	assert.Equal(t, "revset.bookmarks.filter", string(model.dispatchScopes()[0].Name))
	assert.Equal(t, intents.RevsetBookmarksFilterAccept{}, resolve(tea.KeyPressMsg{Code: tea.KeyEnter}))
	assert.Nil(t, resolve(tea.KeyPressMsg{Code: 'j', Text: "j"}), "j is typed into the filter")
}

func TestRevsetBookmarks_KAtTopFocusesRemotes(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin /tmp/remote.git\n"))
	defer commandRunner.Verify()
	model := NewUI(test.NewTestContext(commandRunner))
	model.revsetModel.Update(intents.RevsetBookmarks{})
	resolve := func(key tea.KeyPressMsg) intents.Intent {
		return model.resolver.ResolveKey(key, model.dispatchScopes()).Intent
	}
	k := tea.KeyPressMsg{Code: 'k', Text: "k"}

	assert.Equal(t, intents.RevsetBookmarksFocusRemotes{}, resolve(k), "k on the top row")

	model.revsetModel.Update(intents.RevsetBookmarksFocusRemotes{})
	assert.Equal(t, "revset.bookmarks.remotes", string(model.dispatchScopes()[0].Name))
	assert.Equal(t, intents.RevsetBookmarksCycleRemote{Delta: 1}, resolve(tea.KeyPressMsg{Code: 'l', Text: "l"}))
	assert.Equal(t, intents.RevsetBookmarksCycleRemote{Delta: -1}, resolve(tea.KeyPressMsg{Code: 'h', Text: "h"}))
	assert.Equal(t, intents.RevsetBookmarksRemotesLeave{}, resolve(tea.KeyPressMsg{Code: 'j', Text: "j"}))
	assert.Equal(t, intents.RevsetBookmarksRemotesFetch{}, resolve(tea.KeyPressMsg{Code: 'f', Text: "f"}))
}

// Each revset change of the bookmarks view is delivered by its own command,
// so two quick changes can arrive out of order. The log must end on the
// revset the view shows.
func TestRevsetBookmarks_LateRevsetFromAnEarlierChangeIsIgnored(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin /tmp/remote.git\n"))
	defer commandRunner.Verify()
	model := NewUI(test.NewTestContext(commandRunner))
	model.Update(model.revsetModel.Update(intents.RevsetBookmarks{})())
	model.revsetModel.Update(intents.RevsetBookmarksFocusRemotes{})

	toLocal := model.revsetModel.Update(intents.RevsetBookmarksCycleRemote{Delta: 1})()
	toOrigin := model.revsetModel.Update(intents.RevsetBookmarksCycleRemote{Delta: 1})()
	model.Update(toOrigin)
	model.Update(toLocal)

	assert.Equal(t, string(toOrigin.(common.UpdateRevSetMsg)), model.context.CurrentRevset)
	assert.True(t, model.revsetModel.InBookmarksView())
}

func TestRevsetBookmarks_RenameAndRemoveKeys(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin /tmp/remote.git\n"))
	defer commandRunner.Verify()
	model := NewUI(test.NewTestContext(commandRunner))
	model.revsetModel.Update(intents.RevsetBookmarks{FocusRemotes: true})
	model.revsetModel.Update(intents.RevsetBookmarksCycleRemote{Delta: 2})
	resolve := func(key tea.KeyPressMsg) intents.Intent {
		return model.resolver.ResolveKey(key, model.dispatchScopes()).Intent
	}

	assert.Equal(t, intents.RevsetBookmarksRemotesRename{}, resolve(tea.KeyPressMsg{Code: 'r', Text: "r"}))
	assert.Equal(t, intents.RevsetBookmarksRemotesRemove{}, resolve(tea.KeyPressMsg{Code: 'x', Text: "x"}))

	model.revsetModel.Update(intents.RevsetBookmarksRemotesRename{})
	assert.Equal(t, "revset.bookmarks.remotes.editor", string(model.dispatchScopes()[0].Name))
	assert.Equal(t, intents.Apply{}, resolve(tea.KeyPressMsg{Code: tea.KeyEnter}))
	assert.Nil(t, resolve(tea.KeyPressMsg{Code: 'x', Text: "x"}), "x is typed into the name")
	model.revsetModel.Update(intents.Cancel{})

	model.revsetModel.Update(intents.RevsetBookmarksRemotesRemove{})
	assert.Equal(t, "revset.bookmarks.remotes.confirmation", string(model.dispatchScopes()[0].Name))
	assert.Equal(t, intents.Apply{}, resolve(tea.KeyPressMsg{Code: tea.KeyEnter}))
	assert.Nil(t, resolve(tea.KeyPressMsg{Code: 'y', Text: "y"}), "y reaches the confirmation's own keys")
}
