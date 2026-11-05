package jj

import (
	"errors"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/idursun/jjui/internal/config"
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
	previous := config.Current.Workspace
	config.Current.Workspace = config.WorkspaceConfig{}
	t.Cleanup(func() { config.Current.Workspace = previous })
	assert.NotContains(t, Log("all()", 10, "description"), "--config")
}

func TestLogWorkspaceColorIsOptionalAndRequiresPanelOptIn(t *testing.T) {
	previous := config.Current.Workspace
	t.Cleanup(func() { config.Current.Workspace = previous })
	for _, tc := range []struct {
		name    string
		enabled bool
		color   string
	}{
		{name: "unset"},
		{name: "enabled without color", enabled: true},
		{name: "disabled with color", color: "red"},
		{name: "enabled with named color", enabled: true, color: "red"},
		{name: "enabled with hex color", enabled: true, color: "#ff0000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.Current.Workspace = config.WorkspaceConfig{InteractiveWorkspacePane: tc.enabled, ActiveWorkspaceColor: tc.color}
			args := Log("all()", 10, "description")
			if !tc.enabled || tc.color == "" {
				assert.NotContains(t, args, "--config")
				return
			}
			require.Equal(t, "--config", args[4])
			var override struct {
				Colors map[string]struct {
					Fg string `toml:"fg"`
				} `toml:"colors"`
			}
			_, err := toml.Decode(args[5], &override)
			require.NoError(t, err)
			assert.Equal(t, tc.color, override.Colors["working_copy working_copies"].Fg)
			assert.Len(t, override.Colors, 1)
		})
	}
}
