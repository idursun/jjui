package help

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/stretchr/testify/assert"
)

func TestBuildFromBindings_FiltersToScope(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"J"}},
		{Action: "revisions.apply", Scope: "revisions", Key: config.StringList{"enter"}},
		{Action: "git.move_down", Scope: "ui", Key: config.StringList{"j"}},
	}

	entries := BuildFromBindings("revisions", bindings)
	assert.Equal(t, []Entry{
		{Label: "J", Desc: "move down"},
		{Label: "enter", Desc: "apply"},
	}, entries)
}

func TestBuildFromBindings_UsesConfiguredDescription(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revisions.apply", Desc: "run operation", Scope: "revisions", Key: config.StringList{"enter"}},
		{Action: "ui.cancel", Scope: "revisions", Key: config.StringList{"esc"}},
	}

	entries := BuildFromBindings("revisions", bindings)
	assert.Equal(t, []Entry{
		{Label: "enter", Desc: "run operation"},
		{Label: "esc", Desc: "cancel"},
	}, entries)
}

func TestBuildFromBindings_SameScopeLastBindingWins(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revisions.open_details", Scope: "revisions", Key: config.StringList{"l"}},
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"j"}},
		{Action: "revisions.open_details", Scope: "revisions", Key: config.StringList{"o"}},
	}

	entries := BuildFromBindings("revisions", bindings)
	assert.Equal(t, []Entry{
		{Label: "j", Desc: "move down"},
		{Label: "o", Desc: "open details"},
	}, entries)
}

func TestBuildFromBindings_SameScopeDifferentActionsWithSameLeaf(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revset.edit", Scope: "revisions", Key: config.StringList{"shift+l"}, Desc: "revset"},
		{Action: "revisions.edit", Scope: "revisions", Key: config.StringList{"e"}, Desc: "edit"},
	}

	entries := BuildFromBindings("revisions", bindings)
	assert.Equal(t, []Entry{
		{Label: "shift+l", Desc: "revset"},
		{Label: "e", Desc: "edit"},
	}, entries)
}

func TestBuildGroupedFromBindings_GroupsByScope(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"j"}},
		{Action: "ui.quit", Scope: "ui", Key: config.StringList{"q"}},
	}
	scopes := []keybindings.ScopeName{"revisions", "ui"}

	groups := BuildGroupedFromBindings(scopes, bindings)
	assert.Len(t, groups, 2)
	assert.Equal(t, "Revisions", groups[0].Name)
	assert.Equal(t, "Global", groups[1].Name)
}

func TestMarkOverriddenKeys_MarksOuterDuplicates(t *testing.T) {
	groups := []ScopeGroup{
		{Name: "Inner", Entries: []Entry{{Label: "j", Desc: "move down"}}},
		{Name: "Outer", Entries: []Entry{{Label: "j", Desc: "scroll down"}, {Label: "q", Desc: "quit"}}},
	}
	MarkOverriddenKeys(groups)
	assert.False(t, groups[0].Entries[0].Overridden)
	assert.True(t, groups[1].Entries[0].Overridden)
	assert.False(t, groups[1].Entries[1].Overridden)
}

func TestBuildFromContinuations_SortsAndAnnotatesNonLeaf(t *testing.T) {
	entries := BuildFromContinuations([]dispatch.Continuation{
		{Key: "g", Action: "ui.open_git", IsLeaf: false},
		{Key: "b", Action: "ui.open_bookmarks", IsLeaf: true},
	})

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
