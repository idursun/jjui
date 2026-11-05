package jj

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkspaces(t *testing.T) {
	assert.Equal(t, []Workspace{{Name: "default", Current: true}, {Name: "feature"}},
		ParseWorkspaces("default\ttrue\nfeature\tfalse\n\nmalformed\n"))
	assert.Empty(t, ParseWorkspaces(""))
}

func TestParseWorkspacePathsPreservesEscapedNamesAndDirectories(t *testing.T) {
	assert.Equal(t, []Workspace{
		{Name: "default", Current: true},
		{Name: "with\ttab", Path: "/directory with spaces/line\nbreak\\name"},
	}, ParseWorkspaces("\"default\"\ttrue\tnull\n\"with\\ttab\"\tfalse\t\"/directory with spaces/line\\nbreak\\\\name\"\n"))
}

func TestDiscoverWorkspacesFallsBackForOlderJJ(t *testing.T) {
	calls := 0
	workspaces, err := DiscoverWorkspaces(func(args []string) ([]byte, error) {
		calls++
		if calls == 1 {
			assert.Equal(t, WorkspaceListWithPaths(), args)
			return nil, errors.New("unknown template keyword: root")
		}
		assert.Equal(t, WorkspaceList(), args)
		return []byte("default\ttrue\nfeature\tfalse\n"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Equal(t, []Workspace{{Name: "default", Current: true}, {Name: "feature"}}, workspaces)
}

func TestDiscoverWorkspacesReportsCommandFailure(t *testing.T) {
	failure := errors.New("repository is unavailable")
	workspaces, err := DiscoverWorkspaces(func([]string) ([]byte, error) { return nil, failure })
	assert.ErrorIs(t, err, failure)
	assert.Nil(t, workspaces)
}

func TestLogDoesNotOverrideWorkspaceColorsOrGraphOrdering(t *testing.T) {
	assert.NotContains(t, Log("all()", 10, "description"), "--config")
}
