package context_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/jj"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/ui/common"
)

// blankModel discards every message; it exists only so drainForCompleted can
// hand cmd to test.SimulateModel, which needs an Update(tea.Msg) tea.Cmd to
// feed follow-up commands into (there are none here, we only care about the
// observed messages).
type blankModel struct{}

func (blankModel) Update(tea.Msg) tea.Cmd { return nil }

// drainForCompleted runs cmd (and any tea.Batch/tea.Sequence commands it in
// turn produces) via test.SimulateModel until a common.CommandCompletedMsg is
// observed.
func drainForCompleted(t *testing.T, cmd tea.Cmd) common.CommandCompletedMsg {
	t.Helper()
	var found *common.CommandCompletedMsg
	test.SimulateModel(blankModel{}, cmd, func(msg tea.Msg) {
		if completed, ok := msg.(common.CommandCompletedMsg); ok && found == nil {
			found = &completed
		}
	})
	require.NotNil(t, found, "no CommandCompletedMsg produced")
	return *found
}

// initRepoWithImmutableParent creates a temp jj repo with two commits and
// returns its directory and the commit id of the parent (non-working-copy)
// commit.
func initRepoWithImmutableParent(t *testing.T) (dir string, parentCommitID string) {
	t.Helper()
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj binary not available in PATH")
	}

	dir = t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("jj", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "jj %s: %s", strings.Join(args, " "), out)
		return string(out)
	}

	run("git", "init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644))
	run("commit", "-m", "first commit")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\nworld\n"), 0o644))
	run("commit", "-m", "second commit")

	parentCommitID = strings.TrimSpace(run("log", "-r", "@-", "--no-graph", "--color", "never", "-T", "commit_id"))
	require.NotEmpty(t, parentCommitID)
	return dir, parentCommitID
}

// withImmutableConfig returns a --config argument pair that marks commitID as
// the sole immutable head, so operations against it fail exactly like they
// would against a real immutable (e.g. pushed) commit.
func withImmutableConfig(commitID string) []string {
	return []string{"--config", `revset-aliases."immutable_heads()"=` + strconv.Quote(commitID)}
}

// newTestRunner builds a MainCommandRunner backed by an unstarted askpass
// server, matching production wiring (see cmd/jjui/main.go) without opening
// a real socket.
func newTestRunner(dir string) *appContext.MainCommandRunner {
	return &appContext.MainCommandRunner{Location: dir, Askpass: askpass.NewUnstartedServer("JJUI")}
}

func TestMainCommandRunner_RunCommand_RetriesWithIgnoreImmutable(t *testing.T) {
	dir, parent := initRepoWithImmutableParent(t)
	runner := newTestRunner(dir)

	args := append(withImmutableConfig(parent), "edit", "-r", parent)
	completed := drainForCompleted(t, runner.RunCommand(args))

	require.Error(t, completed.Err)
	require.True(t, jj.IsImmutableError(completed.Err), "expected immutable error, got: %v", completed.Err)
	require.NotNil(t, completed.Retry)

	retryCompleted := drainForCompleted(t, completed.Retry)
	require.NoError(t, retryCompleted.Err)
}

func TestMainCommandRunner_RunCommand_NoRetryForUnrelatedError(t *testing.T) {
	dir, _ := initRepoWithImmutableParent(t)
	runner := newTestRunner(dir)

	completed := drainForCompleted(t, runner.RunCommand([]string{"edit", "-r", "nonexistent-revision-id"}))

	require.Error(t, completed.Err)
	require.False(t, jj.IsImmutableError(completed.Err))
	require.Nil(t, completed.Retry)
}

func TestMainCommandRunner_RunCommandWithInput_RetriesWithIgnoreImmutable(t *testing.T) {
	dir, parent := initRepoWithImmutableParent(t)
	runner := newTestRunner(dir)

	args := append(withImmutableConfig(parent), "describe", "-r", parent, "--stdin")
	completed := drainForCompleted(t, runner.RunCommandWithInput(args, "new description"))

	require.Error(t, completed.Err)
	require.True(t, jj.IsImmutableError(completed.Err), "expected immutable error, got: %v", completed.Err)
	require.NotNil(t, completed.Retry)

	retryCompleted := drainForCompleted(t, completed.Retry)
	require.NoError(t, retryCompleted.Err)
}
