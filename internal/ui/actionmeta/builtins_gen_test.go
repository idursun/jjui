package actionmeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBuiltInActionArgs(t *testing.T) {
	require.NoError(t, ValidateBuiltInActionArgs("revisions.squash.apply", map[string]any{"force": true}))
	require.NoError(t, ValidateBuiltInActionArgs("revisions.revert.set_target", map[string]any{"target": "before"}))

	err := ValidateBuiltInActionArgs("revisions.squash.apply", map[string]any{"force": "true"})
	require.Error(t, err)

	err = ValidateBuiltInActionArgs("revisions.revert.set_target", map[string]any{"target": "bad"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "accepted")

	err = ValidateBuiltInActionArgs("revisions.revert.set_target", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires arg")

	err = ValidateBuiltInActionArgs("revisions.squash.apply", map[string]any{"unknown": true})
	require.Error(t, err)

	err = ValidateBuiltInActionArgs("not_real", nil)
	require.Error(t, err)
}

func TestActionMetadataFor(t *testing.T) {
	meta, ok := ActionMetadataFor("revisions.squash.apply")
	require.True(t, ok)
	require.Equal(t, "revisions.squash.apply", meta.Action)
	require.Contains(t, meta.Args, "force")
	require.NotEmpty(t, meta.Owners)

	_, ok = ActionMetadataFor("not_real")
	require.False(t, ok)
}

func TestBuiltInActions(t *testing.T) {
	actions := BuiltInActions()
	require.NotEmpty(t, actions)
	require.Contains(t, actions, "revisions.squash.apply")
}
