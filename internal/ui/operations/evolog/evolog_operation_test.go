package evolog

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var revision = &jj.Commit{
	ChangeId:      "abc",
	IsWorkingCopy: false,
	Hidden:        false,
	CommitId:      "123",
}

func TestNewOperation_Mode(t *testing.T) {
	tests := []struct {
		name      string
		mode      mode
		isFocused bool
		isOverlay bool
	}{
		{
			name:      "select mode is editing",
			mode:      selectMode,
			isFocused: true,
			isOverlay: true,
		},
		{
			name:      "restore mode is not editing",
			mode:      restoreMode,
			isFocused: true,
			isOverlay: false,
		},
	}
	for _, args := range tests {
		t.Run(args.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			context := test.NewTestContext(commandRunner)
			operation := NewOperation(context, revision)
			operation.mode = args.mode

			assert.Equal(t, args.isFocused, operation.IsFocused())
			assert.Equal(t, args.isOverlay, operation.IsOverlay())
		})
	}
}

func TestOperation_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Evolog(revision.ChangeId))
	defer commandRunner.Verify()

	context := test.NewTestContext(commandRunner)
	operation := NewOperation(context, revision)

	test.SimulateModel(operation, operation.Init())

	assert.True(t, commandRunner.IsVerified())
}

func TestOperation_RestoreMode_NavigationDelegatesToRevisions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	context := test.NewTestContext(commandRunner)
	operation := NewOperation(context, revision)

	operation.Update(updateEvologMsg{
		rows: []parser.Row{
			{Commit: &jj.Commit{ChangeId: "a", CommitId: "111"}},
			{Commit: &jj.Commit{ChangeId: "b", CommitId: "222"}},
		},
	})

	operation.Update(intents.EvologRestore{})
	assert.Equal(t, restoreMode, operation.mode)

	cmd := operation.Update(intents.EvologNavigate{Delta: 1})
	assert.Equal(t, 0, operation.cursor)
	if assert.NotNil(t, cmd) {
		msg := cmd()
		navigate, ok := msg.(intents.Navigate)
		if assert.True(t, ok) {
			assert.Equal(t, 1, navigate.Delta)
		}
	}

	cmd = operation.Update(intents.EvologNavigate{Delta: -1})
	assert.Equal(t, 0, operation.cursor)
	if assert.NotNil(t, cmd) {
		msg := cmd()
		navigate, ok := msg.(intents.Navigate)
		if assert.True(t, ok) {
			assert.Equal(t, -1, navigate.Delta)
		}
	}
}

func TestOperation_RestoreMode_CancelClosesEvolog(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	context := test.NewTestContext(commandRunner)
	operation := NewOperation(context, revision)
	operation.mode = restoreMode

	cmd := operation.Update(intents.Cancel{})
	require.NotNil(t, cmd)

	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		require.NotEmpty(t, batch)
		msg = batch[0]()
	}
	_, ok := msg.(common.CloseViewMsg)
	assert.True(t, ok, "cancel in restore mode should close evolog")
}
