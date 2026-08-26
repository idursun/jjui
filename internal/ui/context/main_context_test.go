package context

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppContextCapturesWorkingDirectoryAndWorkspaceChangePreservesIt(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	ctx := NewAppContext("/repo", nil)
	assert.Equal(t, workingDirectory, ctx.WorkingDirectory)

	ctx.ChangeWorkspace("/other/repo")
	assert.Equal(t, "/other/repo", ctx.Location)
	assert.Equal(t, workingDirectory, ctx.WorkingDirectory)
}

func TestChangeWorkspace(t *testing.T) {
	runner := &MainCommandRunner{Location: "/old/path"}
	ctx := &MainContext{
		Location:      "/old/path",
		CommandRunner: runner,
	}

	ctx.ChangeWorkspace("/new/path")

	assert.Equal(t, "/new/path", ctx.Location)
	assert.Equal(t, "/new/path", runner.Location)
}

func TestChangeWorkspace_UpdatesBothLocations(t *testing.T) {
	runner := &MainCommandRunner{Location: "/a"}
	ctx := &MainContext{
		Location:      "/a",
		CommandRunner: runner,
	}

	ctx.ChangeWorkspace("/b")
	assert.Equal(t, "/b", ctx.Location)
	assert.Equal(t, "/b", runner.Location)

	ctx.ChangeWorkspace("/c")
	assert.Equal(t, "/c", ctx.Location)
	assert.Equal(t, "/c", runner.Location)
}

func TestChangeWorkspace_NonMainCommandRunner(t *testing.T) {
	// If the runner is not a *MainCommandRunner, ctx.Location still updates
	// but the runner's location is unaffected (no panic).
	ctx := &MainContext{
		Location:      "/old",
		CommandRunner: nil,
	}

	require.NotPanics(t, func() {
		ctx.ChangeWorkspace("/new")
	})
	assert.Equal(t, "/new", ctx.Location)
}
