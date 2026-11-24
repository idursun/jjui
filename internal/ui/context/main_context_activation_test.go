package context_test

import (
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivateWorkspaceValidatesDestinationWithMockedRunner(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	t.Cleanup(runner.Verify)
	ctx := test.NewTestContext(runner)
	ctx.Location, ctx.WorkspaceName = "/repo", "default"

	runner.Expect([]string{"-R", "/feature", "root", "--ignore-working-copy"}).SetOutput([]byte("/feature"))
	runner.Expect([]string{"git", "root", "--ignore-working-copy"}).SetOutput([]byte("/repo-store"))
	runner.Expect([]string{"-R", "/feature", "git", "root", "--ignore-working-copy"}).SetOutput([]byte("/repo-store"))
	runner.Expect(append([]string{"-R", "/feature"}, jj.WorkspaceList()...)).SetOutput([]byte("default\tfalse\nfeature\ttrue\n"))
	runner.Expect(append([]string{"-R", "/feature"}, jj.ConfigListAll()...)).SetOutput([]byte(""))

	require.NoError(t, ctx.ActivateWorkspace("feature", "/feature"))
	assert.Equal(t, "/feature", ctx.Location)
	assert.Equal(t, "feature", ctx.WorkspaceName)
	assert.Equal(t, "/feature", ctx.WorkingDirectory)
}

func TestActivateWorkspaceRejectsMissingRoot(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	t.Cleanup(runner.Verify)
	ctx := test.NewTestContext(runner)
	ctx.Location, ctx.WorkspaceName = "/repo", "default"

	runner.Expect([]string{"-R", "/feature", "root", "--ignore-working-copy"}).SetError(errors.New("no such directory"))

	require.Error(t, ctx.ActivateWorkspace("feature", "/feature"))
	assert.Equal(t, "/repo", ctx.Location)
	assert.Equal(t, "default", ctx.WorkspaceName)
}

func TestActivateWorkspaceRejectsForeignRepository(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	t.Cleanup(runner.Verify)
	ctx := test.NewTestContext(runner)
	ctx.Location, ctx.WorkspaceName = "/repo", "default"

	runner.Expect([]string{"-R", "/other", "root", "--ignore-working-copy"}).SetOutput([]byte("/other"))
	runner.Expect([]string{"git", "root", "--ignore-working-copy"}).SetOutput([]byte("/repo-store"))
	runner.Expect([]string{"-R", "/other", "git", "root", "--ignore-working-copy"}).SetOutput([]byte("/other-store"))

	require.Error(t, ctx.ActivateWorkspace("feature", "/other"))
	assert.Equal(t, "/repo", ctx.Location)
	assert.Equal(t, "default", ctx.WorkspaceName)
}

func TestActivateWorkspaceRejectsNameMismatch(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	t.Cleanup(runner.Verify)
	ctx := test.NewTestContext(runner)
	ctx.Location, ctx.WorkspaceName = "/repo", "default"

	runner.Expect([]string{"-R", "/feature", "root", "--ignore-working-copy"}).SetOutput([]byte("/feature"))
	runner.Expect([]string{"git", "root", "--ignore-working-copy"}).SetOutput([]byte("/repo-store"))
	runner.Expect([]string{"-R", "/feature", "git", "root", "--ignore-working-copy"}).SetOutput([]byte("/repo-store"))
	runner.Expect(append([]string{"-R", "/feature"}, jj.WorkspaceList()...)).SetOutput([]byte("default\ttrue\n"))

	err := ctx.ActivateWorkspace("feature", "/feature")
	require.ErrorContains(t, err, "is not workspace feature")
	assert.Equal(t, "/repo", ctx.Location)
	assert.Equal(t, "default", ctx.WorkspaceName)
}
