package exec_process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	shellwords "github.com/mattn/go-shellwords"
)

func ExecMsgFromLine(prompt string, line string) common.ExecMsg {
	line = strings.TrimSpace(line)
	switch prompt {
	case common.ExecShell.Prompt:
		return common.ExecMsg{
			Line: line,
			Mode: common.ExecShell,
		}
	default:
		return common.ExecMsg{
			Line: line,
			Mode: common.ExecJJ,
		}
	}
}

func ExecLine(ctx *context.MainContext, msg common.ExecMsg) tea.Cmd {
	replacements := ctx.CreateReplacements()
	switch msg.Mode {
	case common.ExecJJ:
		args, err := shellwords.Parse(msg.Line)
		if err != nil {
			return func() tea.Msg {
				return common.ExecProcessCompletedMsg{
					Err: fmt.Errorf("parsing command line: %w", err),
					Msg: msg,
				}
			}
		}
		args = jj.TemplatedArgs(args, replacements)
		return execProgram("jj", args, ctx.Location, nil, msg)
	case common.ExecShell:
		// user input is run via `$SHELL -c` to support user specifying command lines
		// that have pipes (eg, to a pager) or redirection.
		program := os.Getenv("SHELL")
		if len(program) == 0 {
			program = "sh"
		}
		args := []string{"-c", msg.Line}
		return execProgram(program, args, ctx.Location, replacements, msg)
	}
	return nil
}

// This is different from command_runner.RunInteractiveCommand.
// This function does not capture any IO. We want all IO to be given to the program.
//
// If we detect tty mode changes while the child is running we treat it as interactive.
// Otherwise, if the program terminates in less than 5 seconds, we ask the user to
// press a key so output does not flash away immediately.
//
// Since programs are run interactively (without capturing stdio) users have
// already seen output on the terminal, and we don't use the usual CommandRunning or
// CommandCompleted machinery we use for background jj processes.
// However, if the program fails we ask the user for confirmation before closing
// and returning stdio back to jjui.
func execProgram(program string, args []string, location string, env map[string]string, msg common.ExecMsg) tea.Cmd {
	p := &process{program: program, args: args, env: env, location: location}
	return tea.Exec(p, func(err error) tea.Msg {
		return common.ExecProcessCompletedMsg{
			Err: err,
			Msg: msg,
		}
	})
}

type process struct {
	program  string
	args     []string
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	env      map[string]string
	location string
}

type runResult struct {
	err               error
	stdinIsTTY        bool
	rawModeChanged    bool
	exitedBeforeTimer bool
}

// Run This is a blocking call.
func (p *process) Run() error {
	cmd := exec.Command(p.program, p.args...)
	cmd.Dir = p.location
	cmd.Stdin = p.stdin
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr
	var env []string
	for k, v := range p.env {
		name := strings.TrimPrefix(k, "$")
		env = append(env, name+"="+v)
	}
	// extend the current environment with context replacements.
	// this is useful for sub-programs to access context vars.
	cmd.Env = append(os.Environ(), env...)

	stdinFile, stdinIsTTY := ttyFile(p.stdin)
	stopObserve := func() bool { return false }
	if stdinIsTTY {
		if stop, ok := observeTTYChanges(stdinFile); ok {
			stopObserve = stop
		}
	}

	startedAt := time.Now()
	err := cmd.Run()
	result := runResult{
		err:               err,
		stdinIsTTY:        stdinIsTTY,
		rawModeChanged:    stopObserve(),
		exitedBeforeTimer: time.Since(startedAt) < 5*time.Second,
	}

	if shouldPrompt(result) {
		_, _ = io.WriteString(p.stderr, "\njjui: press enter to continue... ")
		_ = waitForEnter(stdinFile)
	}
	return err
}

func shouldPrompt(result runResult) bool {
	if !result.stdinIsTTY {
		return false
	}
	if result.err != nil {
		return true
	}
	if result.rawModeChanged {
		return false
	}
	return result.exitedBeforeTimer
}

func waitForEnter(r io.Reader) error {
	reader := bufio.NewReader(r)
	_, err := reader.ReadByte()
	return err
}

func ttyFile(r io.Reader) (*os.File, bool) {
	f, ok := r.(*os.File)
	if !ok || !term.IsTerminal(f.Fd()) {
		return nil, false
	}
	return f, true
}

func (p *process) SetStdin(stdin io.Reader) {
	p.stdin = stdin

}
func (p *process) SetStdout(stdout io.Writer) {
	p.stdout = stdout

}
func (p *process) SetStderr(stderr io.Writer) {
	p.stderr = stderr
}
