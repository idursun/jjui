package immutable

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

var errImmutable = errors.New(`Error: Commit abc123 is immutable
Hint: Could not modify commit: abc123`)

func newCountingRetry() (retry tea.Cmd, count func() int) {
	var runs int
	retry = func() tea.Msg {
		runs++
		return common.CommandCompletedMsg{}
	}
	count = func() int { return runs }
	return retry, count
}

func TestNew_RendersTheImmutableLineOnly(t *testing.T) {
	model := New(errImmutable, nil)

	view := model.confirmation.View()
	assert.Contains(t, view, "Error: Commit abc123 is immutable")
	assert.NotContains(t, view, "Hint:")
}

func TestNew_SkipsLeadingWarningsToFindTheImmutableLine(t *testing.T) {
	err := errors.New("Warning: Deprecated config: foo\nError: Commit abc123 is immutable\nHint: ...")
	model := New(err, nil)

	view := model.confirmation.View()
	assert.Contains(t, view, "Error: Commit abc123 is immutable")
	assert.NotContains(t, view, "Warning:")
}

func TestNew_DefaultsSelectionToNo(t *testing.T) {
	retry, runs := newCountingRetry()
	model := New(errImmutable, retry)

	var msgs []tea.Msg
	test.SimulateModel(model, func() tea.Msg { return intents.Apply{} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Equal(t, 0, runs(), "pressing Enter without navigating must not run retry: the safe option must be selected by default")
	assert.Contains(t, msgs, CloseMsg{})
}

func TestSelectingYesThenApply_RunsRetryThenCloses(t *testing.T) {
	retry, runs := newCountingRetry()
	model := New(errImmutable, retry)

	var msgs []tea.Msg
	test.SimulateModel(model, tea.Sequence(
		func() tea.Msg { return intents.OptionSelect{Delta: -1} }, // Yes is the first option
		func() tea.Msg { return intents.Apply{} },
	), func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Equal(t, 1, runs(), "selecting Yes should run the retry command")
	assert.Contains(t, msgs, CloseMsg{}, "selecting Yes should close the dialog once retry completes")
}

func TestPressingY_RunsRetryThenCloses(t *testing.T) {
	retry, runs := newCountingRetry()
	model := New(errImmutable, retry)

	var msgs []tea.Msg
	test.SimulateModel(model, func() tea.Msg { return tea.KeyPressMsg{Text: "y", Code: 'y'} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Equal(t, 1, runs())
	assert.Contains(t, msgs, CloseMsg{})
}

func TestApplyTwice_RunsRetryOnlyOnce(t *testing.T) {
	retry, runs := newCountingRetry()
	model := New(errImmutable, retry)

	test.SimulateModel(model, tea.Sequence(
		func() tea.Msg { return intents.OptionSelect{Delta: -1} },
		func() tea.Msg { return intents.Apply{} },
		func() tea.Msg { return intents.Apply{} },
	))

	assert.Equal(t, 1, runs(), "a second Apply (e.g. from a held key) while the retry's Sequence is still resolving must not run retry again")
}

func TestCancel_ClosesWithoutRunningRetry(t *testing.T) {
	retry, runs := newCountingRetry()
	model := New(errImmutable, retry)

	var msgs []tea.Msg
	test.SimulateModel(model, func() tea.Msg { return intents.Cancel{} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Equal(t, 0, runs(), "cancelling must not run the retry command")
	assert.Contains(t, msgs, CloseMsg{})
}

func TestHandleIntent_UnhandledIntentIsNotConsumed(t *testing.T) {
	model := New(errImmutable, nil)

	_, handled := model.HandleIntent(intents.PreviewShow{})
	assert.False(t, handled)
}

func TestScopes_BlocksOuterScopes(t *testing.T) {
	model := New(errImmutable, nil)

	scopes := model.Scopes()
	assert.Len(t, scopes, 1)
	assert.Equal(t, common.LeakNone, scopes[0].Leak)
}
