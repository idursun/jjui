package exec_process

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
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
		args := strings.Fields(msg.Line)
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
// If program terminates in less than 5-secs we ask to press a key to return to JJUI.
// This is useful for programs that would otherwise terminate quickly and just flash.
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

	// If program terminates quickly (most likely non-interactive commands),
	// we ask the user to press a key, so they can at least see the output.
	askUserClose := true
	go func() {
		time.Sleep(5 * time.Second)
		askUserClose = false
	}()

	err := cmd.Run()
	// Dont auto-close on error.
	if askUserClose || err != nil {
		p.stderr.Write([]byte("\njjui: press enter to continue... "))
		reader := bufio.NewReader(p.stdin)
		reader.ReadByte()
	}
	return err
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
