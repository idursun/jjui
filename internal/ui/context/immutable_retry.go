package context

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
)

// BuildBufferedRetry returns a command that re-runs args with --ignore-immutable
// appended, using either RunCommand or RunCommandWithInput depending on whether
// input is non-nil. Nothing runs until the returned Cmd is invoked, same as any
// other tea.Cmd. It returns nil when err is not jj's "commit is immutable" error,
// or when args already requests --ignore-immutable.
func BuildBufferedRetry(runner CommandRunner, args []string, input *string, continuations []tea.Cmd, err error) tea.Cmd {
	if !canRetryWithIgnoreImmutable(args, err) {
		return nil
	}
	retryArgs := withIgnoreImmutable(args)
	return func() tea.Msg {
		var cmd tea.Cmd
		if input != nil {
			cmd = runner.RunCommandWithInput(retryArgs, *input, continuations...)
		} else {
			cmd = runner.RunCommand(retryArgs, continuations...)
		}
		if cmd == nil {
			return nil
		}
		return cmd()
	}
}

// BuildInteractiveRetry is the RunInteractiveCommand counterpart of BuildBufferedRetry.
func BuildInteractiveRetry(runner CommandRunner, args []string, continuation tea.Cmd, err error) tea.Cmd {
	if !canRetryWithIgnoreImmutable(args, err) {
		return nil
	}
	retryArgs := withIgnoreImmutable(args)
	return func() tea.Msg {
		cmd := runner.RunInteractiveCommand(retryArgs, continuation)
		if cmd == nil {
			return nil
		}
		return cmd()
	}
}

func canRetryWithIgnoreImmutable(args []string, err error) bool {
	if !jj.IsImmutableError(err) || slices.Contains(args, "--ignore-immutable") {
		return false
	}
	// args can come from user-authored lua actions (see scripting.RunScript),
	// which may use "--" to end option parsing early. Appending the flag
	// after that would make jj parse it as a positional, so decline rather
	// than risk a confusing retry failure.
	return !slices.Contains(args, "--")
}

func withIgnoreImmutable(args []string) []string {
	retryArgs := make([]string, 0, len(args)+1)
	retryArgs = append(retryArgs, args...)
	retryArgs = append(retryArgs, "--ignore-immutable")
	return retryArgs
}
