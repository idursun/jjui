package details

import (
	"bytes"
	"github.com/idursun/jjui/internal/jj"
	"testing"
	"time"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest"
)

const (
	Revision     = "ignored"
	StatusOutput = "M file.txt\nA newfile.txt\n"
)

func TestModel_Init_ExecutesStatusCommand(t *testing.T) {
	context := test.NewTestContext(t)
	context.Expect(jj.Snapshot())
	context.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer context.Verify()

	tm := teatest.NewTestModel(t, New(context, Revision))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})
}

func TestModel_Update_RestoresSelectedFiles(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.Snapshot())
	c.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	c.Expect(jj.Restore(Revision, []string{"file.txt"}))
	defer c.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(New(c, Revision)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.Snapshot())
	c.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	c.Expect(jj.Split(Revision, []string{"file.txt"}))
	defer c.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(New(c, Revision)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.Snapshot())
	c.Expect(jj.Status(Revision)).SetOutput([]byte("R internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	c.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}))
	defer c.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(New(c, Revision)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
