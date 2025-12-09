package customcommands

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/require"
)

type executedMsg struct{ name string }

type stubCommand struct {
	context.CustomCommandBase
	applicable bool
	prepared   bool
}

func (s *stubCommand) Description(ctx *context.MainContext) string {
	return s.Name
}

func (s *stubCommand) Prepare(ctx *context.MainContext) tea.Cmd {
	return func() tea.Msg {
		s.prepared = true
		return executedMsg{name: s.Name}
	}
}

func (s *stubCommand) IsApplicableTo(item context.SelectedItem) bool {
	return s.applicable
}

type overlayModel struct {
	overlay *SequenceOverlay
	last    SequenceResult
}

func (m *overlayModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		res := m.overlay.HandleKey(msg)
		m.last = res
		if res.Active {
			// Avoid running timeout ticks during tests; we only care about overlay state.
			return nil
		}
		return res.Cmd
	default:
		cmd := m.overlay.Update(msg)
		if cmd != nil {
			m.last = SequenceResult{Cmd: cmd, Handled: true, Active: m.overlay.Active()}
		}
		return cmd
	}
}

func TestHandleKey_executes_single_key_sequence(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))

	cmd := &stubCommand{
		CustomCommandBase: context.CustomCommandBase{
			Name:        "Run",
			KeySequence: []string{"x"},
		},
		applicable: true,
	}
	ctx.CustomCommands["run"] = cmd

	overlay := NewSequenceOverlay(ctx)
	overlay.ViewNode.Parent = common.NewViewNode(80, 24)
	model := &overlayModel{overlay: overlay}

	var msgs []tea.Msg
	test.SimulateModel(model, test.Type("x"), func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	require.True(t, cmd.prepared, "expected command Prepare to run for matching key")
	require.True(t, model.last.Handled, "expected key to be handled")
	require.False(t, model.last.Active, "single key sequences should not leave overlay active")

	var executed bool
	for _, msg := range msgs {
		if got, ok := msg.(executedMsg); ok && got.name == "Run" {
			executed = true
			break
		}
	}
	require.True(t, executed, "expected prepared command to emit executedMsg")
}

func TestHandleKey_executes_multi_key_sequence(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	cmd := &stubCommand{
		CustomCommandBase: context.CustomCommandBase{
			Name:        "GoRun",
			KeySequence: []string{"g", "o"},
		},
		applicable: true,
	}
	ctx.CustomCommands["go-run"] = cmd

	overlay := NewSequenceOverlay(ctx)
	overlay.ViewNode.Parent = common.NewViewNode(80, 24)
	model := &overlayModel{overlay: overlay}

	executed := false
	test.SimulateModel(model, test.Type("go"), func(msg tea.Msg) {
		if got, ok := msg.(executedMsg); ok && got.name == "GoRun" {
			executed = true
		}
	})

	require.True(t, executed, "expected prepared command to emit executedMsg")
	require.True(t, cmd.prepared, "expected command Prepare to run after full sequence")
	require.True(t, model.last.Handled, "expected second key to be handled")
	require.False(t, model.last.Active, "overlay should deactivate after sequence completes")
}
