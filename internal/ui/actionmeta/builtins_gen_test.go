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

func TestActionArgSchemaAndRequiredArgs(t *testing.T) {
	schema := ActionArgSchema("revisions.squash.apply")
	require.NotNil(t, schema)
	require.Contains(t, schema, "force")

	require.Nil(t, ActionArgSchema("not_real"))

	required := ActionRequiredArgs("revisions.revert.set_target")
	require.Contains(t, required, "target")

	require.Nil(t, ActionRequiredArgs("not_real"))
}

func TestBuiltInActions(t *testing.T) {
	actions := BuiltInActions()
	require.NotEmpty(t, actions)
	require.Contains(t, actions, "revisions.squash.apply")
}

func TestActionWhen(t *testing.T) {
	require.Equal(t, "revisions.has_revision && !revisions.is_empty", ActionWhen("revisions.diff"))
	require.Empty(t, ActionWhen("revisions.new"))
}
