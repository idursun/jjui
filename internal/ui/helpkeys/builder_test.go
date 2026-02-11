package helpkeys

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/stretchr/testify/assert"
)

func TestBuildFromBindings_RespectsScopeOrderAndActionTokenDedupe(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "ui.git.move_down", Scope: "ui", Key: config.StringList{"j"}},
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"J"}},
		{Action: "revisions.apply", Scope: "revisions", Key: config.StringList{"enter"}},
		{Action: "ui.git.apply", Scope: "ui", Key: config.StringList{"a"}},
	}
	scopes := []keybindings.Scope{"revisions", "ui"}

	entries := BuildFromBindings(scopes, bindings, nil)
	assert.Equal(t, []Entry{
		{Label: "J", Desc: "move down"},
		{Label: "enter", Desc: "apply"},
	}, entries)
}

func TestBuildFromBindings_UsesConfiguredDescription(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revisions.apply", Scope: "revisions", Key: config.StringList{"enter"}},
		{Action: "ui.cancel", Scope: "revisions", Key: config.StringList{"esc"}},
	}
	configured := map[keybindings.Action]config.ActionConfig{
		"revisions.apply": {Desc: "run operation"},
	}
	scopes := []keybindings.Scope{"revisions"}

	entries := BuildFromBindings(scopes, bindings, configured)
	assert.Equal(t, []Entry{
		{Label: "enter", Desc: "run operation"},
		{Label: "esc", Desc: "cancel"},
	}, entries)
}

func TestBuildFromContinuations_SortsAndAnnotatesNonLeaf(t *testing.T) {
	entries := BuildFromContinuations([]dispatch.Continuation{
		{Key: "g", Action: "ui.open_git", IsLeaf: false},
		{Key: "b", Action: "ui.open_bookmarks", IsLeaf: true},
	}, nil)

	assert.Equal(t, []Entry{
		{Label: "b", Desc: "open bookmarks"},
		{Label: "g", Desc: "open git ..."},
	}, entries)
}

func TestNormalizeDisplayKey_PrettyKeys(t *testing.T) {
	assert.Equal(t, "↑", NormalizeDisplayKey("up"))
	assert.Equal(t, "↓", NormalizeDisplayKey("down"))
	assert.Equal(t, "←", NormalizeDisplayKey("left"))
	assert.Equal(t, "→", NormalizeDisplayKey("right"))
	assert.Equal(t, "esc", NormalizeDisplayKey("esc"))
	assert.Equal(t, "enter", NormalizeDisplayKey("enter"))
}
