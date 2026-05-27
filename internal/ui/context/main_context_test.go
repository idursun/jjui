package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeDirectory(t *testing.T) {
	runner := &MainCommandRunner{Location: "/old/path"}
	ctx := &MainContext{
		Location:      "/old/path",
		CommandRunner: runner,
	}

	ctx.ChangeDirectory("/new/path")

	assert.Equal(t, "/new/path", ctx.Location)
	assert.Equal(t, "/new/path", runner.Location)
}

func TestChangeDirectory_UpdatesBothLocations(t *testing.T) {
	runner := &MainCommandRunner{Location: "/a"}
	ctx := &MainContext{
		Location:      "/a",
		CommandRunner: runner,
	}

	ctx.ChangeDirectory("/b")
	assert.Equal(t, "/b", ctx.Location)
	assert.Equal(t, "/b", runner.Location)

	ctx.ChangeDirectory("/c")
	assert.Equal(t, "/c", ctx.Location)
	assert.Equal(t, "/c", runner.Location)
}

func TestChangeDirectory_NonMainCommandRunner(t *testing.T) {
	// If the runner is not a *MainCommandRunner, ctx.Location still updates
	// but the runner's location is unaffected (no panic).
	ctx := &MainContext{
		Location:      "/old",
		CommandRunner: nil,
	}

	require.NotPanics(t, func() {
		ctx.ChangeDirectory("/new")
	})
	assert.Equal(t, "/new", ctx.Location)
}
