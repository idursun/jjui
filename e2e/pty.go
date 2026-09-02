package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"

	ghostty "go.mitchellh.com/libghostty"
)

// ptySession connects a child process to a libghostty terminal model. The
// mutex is important: libghostty terminals are stateful and not safe for
// concurrent use, while PTY output arrives on a reader goroutine.
type ptySession struct {
	term     *ghostty.Terminal
	master   *os.File
	cmd      *exec.Cmd
	readDone chan struct{}
	exitDone chan struct{}

	mu        sync.Mutex
	raw       bytes.Buffer
	waitErr   error
	exitCode  int
	startedAt time.Time
	trace     []traceEvent
}

const perWaitTimeout = 5 * time.Second

type traceEvent struct {
	at      time.Duration
	message string
}

type cursorPosition struct {
	X          uint16
	Y          uint16
	Visible    bool
	InViewport bool
	WideTail   bool
}

func startPTY(term *ghostty.Terminal, command []string, dir string, env []string, cols, rows uint16) (*ptySession, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("empty PTY command")
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Env = mergeEnvironment(os.Environ(), env)

	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}

	session := &ptySession{
		term:      term,
		master:    master,
		cmd:       cmd,
		readDone:  make(chan struct{}),
		exitDone:  make(chan struct{}),
		startedAt: time.Now(),
	}
	session.addTrace("started " + strings.Join(command, " "))
	go session.readOutput()
	go session.waitProcess()
	return session, nil
}

func (s *ptySession) readOutput() {
	defer close(s.readDone)

	buf := make([]byte, 32*1024)
	for {
		n, err := s.master.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.raw.Write(buf[:n])
			s.term.VTWrite(buf[:n])
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *ptySession) SendKey(key ghostty.Key, text string, mods ghostty.Mods) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := sendKey(s.master, s.term, key, text, mods); err != nil {
		return err
	}
	s.addTraceLocked(fmt.Sprintf("sent key=%s text=%q mods=%s", key, text, mods))
	return nil
}

func (s *ptySession) SendText(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := []byte(text)
	n, err := s.master.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	s.addTraceLocked(fmt.Sprintf("sent text %q", text))
	return nil
}

// Resize changes both the child PTY and the terminal model used for screen
// snapshots. Updating both while holding the session mutex keeps output
// processing from observing mismatched dimensions.
func (s *ptySession) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := pty.Setsize(s.master, &pty.Winsize{Cols: cols, Rows: rows}); err != nil {
		return err
	}
	if err := s.term.Resize(cols, rows, 8, 16); err != nil {
		return err
	}
	s.addTraceLocked(fmt.Sprintf("resized terminal to %dx%d", cols, rows))
	return nil
}

func (s *ptySession) Snapshot() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return screenSnapshot(s.term)
}

func (s *ptySession) StyledSnapshot() (styledScreen, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return styledScreenSnapshot(s.term)
}

func (s *ptySession) Cursor() (cursorPosition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cursorPositionSnapshot(s.term)
}

func (s *ptySession) RawOutput() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.raw.String()
}

func (s *ptySession) Wait() error {
	<-s.readDone
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waitErr
}

func (s *ptySession) WaitContext(ctx context.Context) error {
	select {
	case <-s.readDone:
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.waitErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForExit waits until the child has exited and the PTY reader has drained.
// A non-zero child status is returned as an error; use ExpectExit when that is
// the expected outcome.
func (s *ptySession) WaitForExit(ctx context.Context) error {
	select {
	case <-s.exitDone:
		<-s.readDone
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.waitErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ptySession) ExpectExit(ctx context.Context, code int) error {
	err := s.WaitForExit(ctx)
	if err != nil {
		select {
		case <-s.exitDone:
		default:
			return err
		}
	}
	got := s.processExitCode()
	if got != code {
		return fmt.Errorf("process exited with code %d, want %d: %v", got, code, err)
	}
	return nil
}

func (s *ptySession) processExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}

func (s *ptySession) Close() error {
	if s.cmd.Process != nil && s.cmd.ProcessState == nil {
		_ = s.cmd.Process.Kill()
	}
	return s.Wait()
}

func (s *ptySession) WaitForScreen(ctx context.Context, predicate func([]string) bool) ([]string, error) {
	return s.waitForScreen(ctx, 1, predicate)
}

func (s *ptySession) WaitForCursor(ctx context.Context, predicate func(cursorPosition) bool) (cursorPosition, error) {
	waitCtx, cancel := context.WithTimeout(ctx, perWaitTimeout)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	var last cursorPosition
	for {
		select {
		case <-waitCtx.Done():
			return last, fmt.Errorf("waiting for cursor: %w (last position: %d,%d visible=%t in_viewport=%t)", waitCtx.Err(), last.X, last.Y, last.Visible, last.InViewport)
		case <-s.exitDone:
			return last, s.screenWaitExitError(fmt.Sprintf("cursor position: %d,%d visible=%t in_viewport=%t", last.X, last.Y, last.Visible, last.InViewport))
		case <-ticker.C:
			cursor, err := s.Cursor()
			if err != nil {
				return last, err
			}
			last = cursor
			if predicate(cursor) {
				return cursor, nil
			}
		}
	}
}

func (s *ptySession) WaitForStyledScreen(ctx context.Context, predicate func(styledScreen) bool) (styledScreen, error) {
	waitCtx, cancel := context.WithTimeout(ctx, perWaitTimeout)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	var last styledScreen
	for {
		select {
		case <-waitCtx.Done():
			return last, fmt.Errorf("waiting for styled screen: %w\nlast screen:\n%s", waitCtx.Err(), formatStyledScreen(last))
		case <-s.exitDone:
			return last, s.screenWaitExitError(formatStyledScreen(last))
		case <-ticker.C:
			screen, err := s.StyledSnapshot()
			if err != nil {
				return nil, err
			}
			last = screen
			if predicate(screen) {
				return screen, nil
			}
		}
	}
}

// WaitForStableScreen waits for matching snapshots whose full styled terminal
// state is unchanged across the requested number of samples.
func (s *ptySession) WaitForStableScreen(ctx context.Context, samples int, predicate func([]string) bool) ([]string, error) {
	return s.waitForScreen(ctx, samples, predicate)
}

func (s *ptySession) waitForScreen(ctx context.Context, samples int, predicate func([]string) bool) ([]string, error) {
	if samples < 1 {
		return nil, fmt.Errorf("screen stability sample count must be positive")
	}
	waitCtx, cancel := context.WithTimeout(ctx, perWaitTimeout)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	var last []string
	var previous styledScreen
	stable := 0
	for {
		select {
		case <-waitCtx.Done():
			return last, fmt.Errorf("waiting for screen: %w\nlast screen:\n%s", waitCtx.Err(), formatScreen(last))
		case <-s.exitDone:
			return last, s.screenWaitExitError(formatScreen(last))
		case <-ticker.C:
			styled, err := s.StyledSnapshot()
			if err != nil {
				return nil, err
			}
			screen := styledTextSnapshot(styled)
			last = screen
			if predicate(screen) {
				if stable > 0 && reflect.DeepEqual(previous, styled) {
					stable++
				} else {
					stable = 1
				}
				if stable >= samples {
					return screen, nil
				}
			} else {
				stable = 0
			}
			previous = styled
		}
	}
}

func (s *ptySession) waitProcess() {
	err := s.cmd.Wait()
	s.mu.Lock()
	s.waitErr = err
	if s.cmd.ProcessState != nil {
		s.exitCode = s.cmd.ProcessState.ExitCode()
	}
	s.addTraceLocked(fmt.Sprintf("process exited code=%d err=%v", s.exitCode, err))
	s.mu.Unlock()
	close(s.exitDone)
	_ = s.master.Close()
	<-s.readDone
}

func (s *ptySession) screenWaitExitError(last string) error {
	s.mu.Lock()
	err := s.waitErr
	code := s.exitCode
	s.mu.Unlock()
	return fmt.Errorf("process exited with code %d: %v\nlast screen:\n%s\nraw output:\n%s", code, err, last, tailString(s.RawOutput(), 64*1024))
}

func (s *ptySession) addTrace(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addTraceLocked(message)
}

func (s *ptySession) addTraceLocked(message string) {
	s.trace = append(s.trace, traceEvent{at: time.Since(s.startedAt), message: message})
}

func (s *ptySession) Trace() []traceEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]traceEvent(nil), s.trace...)
}

func tailString(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return fmt.Sprintf("<output truncated; showing last %d bytes>\n%s", maxBytes, value[len(value)-maxBytes:])
}

func mergeEnvironment(base, overrides []string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))
	for _, value := range append(base, overrides...) {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if _, exists := values[parts[0]]; !exists {
			order = append(order, parts[0])
		}
		values[parts[0]] = parts[1]
	}

	result := make([]string, 0, len(order))
	for _, key := range order {
		result = append(result, key+"="+values[key])
	}
	return result
}

func formatScreen(screen []string) string {
	if len(screen) == 0 {
		return "<no screen captured>"
	}
	return strings.Join(screen, "\n")
}
