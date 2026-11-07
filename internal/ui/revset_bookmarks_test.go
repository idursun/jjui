package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
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
