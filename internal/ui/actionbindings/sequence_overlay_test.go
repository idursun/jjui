package actionbindings

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/actions"
	"github.com/idursun/jjui/internal/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/require"
)

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

func TestHandleKey_executes_single_key_sequence_for_action(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Actions = map[string]actions.Action{
		"Run": {Name: "Run", Lua: "return 'run'"},
	}
	ctx.KeyBindings = []bindings.KeyBinding{
		{Action: "Run", KeySequence: []string{"x"}},
	}

	overlay := NewSequenceOverlay(ctx, func() map[string]any { return map[string]any{} })
	overlay.ViewNode.Parent = common.NewViewNode(80, 24)
	model := &overlayModel{overlay: overlay}

	var msgs []tea.Msg
	test.SimulateModel(model, test.Type("x"), func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	require.True(t, model.last.Handled, "expected key to be handled")
	require.False(t, model.last.Active, "single key sequence should not leave overlay active")

	var executed bool
	for _, msg := range msgs {
		if got, ok := msg.(common.RunLuaScriptMsg); ok && got.Script == "return 'run'" {
			executed = true
			break
		}
	}
	require.True(t, executed, "expected action to emit RunLuaScriptMsg")
}

func TestHandleKey_executes_multi_key_sequence_for_action(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Actions = map[string]actions.Action{
		"GoRun": {Name: "GoRun", Lua: "return 'go'"},
	}
	ctx.KeyBindings = []bindings.KeyBinding{
		{Action: "GoRun", KeySequence: []string{"g", "o"}},
	}

	overlay := NewSequenceOverlay(ctx, func() map[string]any { return map[string]any{} })
	overlay.ViewNode.Parent = common.NewViewNode(80, 24)
	model := &overlayModel{overlay: overlay}

	executed := false
	test.SimulateModel(model, test.Type("go"), func(msg tea.Msg) {
		if got, ok := msg.(common.RunLuaScriptMsg); ok && got.Script == "return 'go'" {
			executed = true
		}
	})

	require.True(t, executed, "expected action to emit RunLuaScriptMsg")
	require.True(t, model.last.Handled, "expected final key to be handled")
	require.False(t, model.last.Active, "overlay should deactivate after sequence completes")
}

func TestHandleKey_respects_when_clause(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Actions = map[string]actions.Action{
		"Run": {Name: "Run", Lua: "return 'run'"},
	}
	cond, err := bindings.ParseCondition("revisions.focused")
	require.NoError(t, err)
	ctx.KeyBindings = []bindings.KeyBinding{
		{
			Action:      "Run",
			KeySequence: []string{"x"},
			When:        "revisions.focused",
			Condition:   cond,
		},
	}

	overlay := NewSequenceOverlay(ctx, func() map[string]any {
		return map[string]any{"revisions.focused": false}
	})
	overlay.ViewNode.Parent = common.NewViewNode(80, 24)
	model := &overlayModel{overlay: overlay}

	test.SimulateModel(model, test.Type("x"))

	require.False(t, model.last.Handled, "expected key to be ignored because condition is false")
	require.False(t, model.last.Active, "overlay should stay inactive when when-clause fails")
}
