package test

import (
	"os/exec"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/stretchr/testify/assert"
)

type JJCommands struct {
	expectations map[string]*MockedCommand
}

type MockedCommand struct {
	args   []string
	Output []byte
	Err    error
	called bool
	t      *testing.T
}

func (m *MockedCommand) CombinedOutput() ([]byte, error) {
	m.called = true
	return m.Output, m.Err
}

func (m *MockedCommand) GetCommand() *exec.Cmd {
	m.called = true
	// do nothing
	return exec.Command(":")
}

func (m *MockedCommand) Args() []string {
	return m.args
}

func (t *JJCommands) GetConfig(key string) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) RebaseCommand(from string, to string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) RebaseBranchCommand(from string, to string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) SetDescription(rev string, description string) jj.Command {
	expectation := t.expectations["SetDescription"]
	if expectation == nil {
		panic("unexpected call to SetDescription")
	}

	assert.Equal(expectation.t, expectation.args, []string{rev, description})
	return expectation
}

func (t *JJCommands) ListBookmark(revision string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) SetBookmark(revision string, name string) jj.Command {
	expectation := t.expectations["SetBookmark"]
	if expectation == nil {
		panic("unexpected call to SetBookmark")
	}
	assert.Equal(expectation.t, expectation.args, []string{revision, name})
	return expectation
}

func (t *JJCommands) MoveBookmark(revision string, bookmark string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) DeleteBookmark(bookmark string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) GitFetch() jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) GitPush() jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Diff(revision string, fineName string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Edit(revision string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) DiffEdit(revision string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Abandon(revision string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) New(from string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Undo() jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Split(revision string, files []string) jj.Command {
	expectation := t.expectations["Split"]
	if expectation == nil {
		panic("unexpected call to Split")
	}
	assert.Equal(expectation.t, expectation.args, append([]string{revision}, files...))
	return expectation
}

func (t *JJCommands) ExpectSplit(tt *testing.T, revision string, files []string) *MockedCommand {
	command := MockedCommand{
		args: append([]string{revision}, files...),
		t:    tt,
	}
	t.expectations["Split"] = &command
	return &command
}

func (t *JJCommands) GetCommits(revset string) ([]jj.GraphRow, error) {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Squash(from string, destination string) jj.Command {
	// TODO implement me
	panic("implement me")
}

func (t *JJCommands) Status(revision string) jj.Command {
	expectation := t.expectations["Status"]
	if expectation == nil {
		panic("unexpected call to Status")
	}
	assert.Equal(expectation.t, expectation.args, []string{revision})
	return expectation
}

func (t *JJCommands) ExpectStatus(tt *testing.T, revision string) *MockedCommand {
	command := MockedCommand{
		args: []string{revision},
		t:    tt,
	}
	t.expectations["Status"] = &command
	return &command
}

func (t *JJCommands) Restore(revision string, files []string) jj.Command {
	expectation := t.expectations["Restore"]
	if expectation == nil {
		panic("unexpected call to Restore")
	}
	assert.Equal(expectation.t, expectation.args, append([]string{revision}, files...))
	return expectation
}

func (t *JJCommands) ExpectRestore(tt *testing.T, revision string, files []string) *MockedCommand {
	command := MockedCommand{
		args: append([]string{revision}, files...),
		t:    tt,
	}
	t.expectations["Restore"] = &command
	return &command
}

func (t *JJCommands) ExpectSetDescription(tt *testing.T, rev string, description string) *MockedCommand {
	command := MockedCommand{
		args: []string{rev, description},
		t:    tt,
	}
	t.expectations["SetDescription"] = &command
	return &command
}

func (t *JJCommands) ExpectSetBookmark(tt *testing.T, revision string, name string) *MockedCommand {
	command := MockedCommand{
		args: []string{revision, name},
		t:    tt,
	}
	t.expectations["SetBookmark"] = &command
	return &command
}

func (t *JJCommands) Verify(tt *testing.T) {
	for name, expectation := range t.expectations {
		if !expectation.called {
			tt.Errorf("expected %s to be called", name)
		}
	}
}

func NewJJCommands() *JJCommands {
	return &JJCommands{
		expectations: make(map[string]*MockedCommand),
	}
}
