package dispatch

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/bindings"
	"github.com/stretchr/testify/require"
)

func TestValidateBindings_KeyAndSeqExclusive(t *testing.T) {
	err := bindings.ValidateBindings([]bindings.Binding{{
		Action: "open_git",
		Scope:  "revisions",
		Key:    []string{"g"},
		Seq:    []string{"g", "p"},
	}})
	require.Error(t, err)
}

func TestDispatcher_ResolvesSingleKeyByScopePrecedence(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "global_action", Scope: "ui", Key: []string{"x"}},
		{Action: "inner_action", Scope: "revisions", Key: []string{"x"}},
	})
	require.NoError(t, err)

	result := d.Resolve(runeKey('x'), []bindings.Scope{"revisions", "ui"})
	require.True(t, result.Consumed)
	require.False(t, result.Pending)
	require.Equal(t, bindings.Action("inner_action"), result.Action)
}

func TestDispatcher_SequencePrefixWinsOverSingleKey(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "open_git", Scope: "revisions", Key: []string{"g"}},
		{Action: "git_push", Scope: "revisions", Seq: []string{"g", "p"}},
	})
	require.NoError(t, err)

	first := d.Resolve(runeKey('g'), []bindings.Scope{"revisions"})
	require.True(t, first.Consumed)
	require.True(t, first.Pending)
	require.Empty(t, first.Action)
	require.Len(t, first.Continuations, 1)
	require.Equal(t, "p", first.Continuations[0].Key)

	second := d.Resolve(runeKey('p'), []bindings.Scope{"revisions"})
	require.True(t, second.Consumed)
	require.False(t, second.Pending)
	require.Equal(t, bindings.Action("git_push"), second.Action)
}

func TestDispatcher_SwallowsMismatchedContinuationAndResets(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "git_push", Scope: "revisions", Seq: []string{"g", "p"}},
		{Action: "open_git", Scope: "revisions", Key: []string{"g"}},
	})
	require.NoError(t, err)

	first := d.Resolve(runeKey('g'), []bindings.Scope{"revisions"})
	require.True(t, first.Pending)

	mismatch := d.Resolve(runeKey('x'), []bindings.Scope{"revisions"})
	require.True(t, mismatch.Consumed)
	require.False(t, mismatch.Pending)
	require.Empty(t, mismatch.Action)

	// Sequence state is reset; g starts sequence mode again.
	again := d.Resolve(runeKey('g'), []bindings.Scope{"revisions"})
	require.True(t, again.Pending)
}

func TestDispatcher_CancelSequenceWithEsc(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{{Action: "git_push", Scope: "revisions", Seq: []string{"g", "p"}}})
	require.NoError(t, err)

	first := d.Resolve(runeKey('g'), []bindings.Scope{"revisions"})
	require.True(t, first.Pending)

	cancel := d.Resolve(tea.KeyPressMsg{Code: tea.KeyEsc}, []bindings.Scope{"revisions"})
	require.True(t, cancel.Consumed)
	require.False(t, cancel.Pending)
	require.Empty(t, cancel.Action)
}

func TestDispatcher_ResolvesSpaceAliasForSingleKey(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "toggle_select", Scope: "revisions", Key: []string{"space"}},
	})
	require.NoError(t, err)

	result := d.Resolve(runeKey(' '), []bindings.Scope{"revisions"})
	require.True(t, result.Consumed)
	require.False(t, result.Pending)
	require.Equal(t, bindings.Action("toggle_select"), result.Action)
}

func TestDispatcher_ResolvesSpaceAliasInSequence(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "do_thing", Scope: "ui", Seq: []string{"w", "space"}},
	})
	require.NoError(t, err)

	first := d.Resolve(runeKey('w'), []bindings.Scope{"ui"})
	require.True(t, first.Consumed)
	require.True(t, first.Pending)

	second := d.Resolve(runeKey(' '), []bindings.Scope{"ui"})
	require.True(t, second.Consumed)
	require.False(t, second.Pending)
	require.Equal(t, bindings.Action("do_thing"), second.Action)
}

func TestDispatcher_ResolvesBindingArgs(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "apply", Scope: "revisions.squash", Key: []string{"enter"}, Args: map[string]any{"force": true}},
	})
	require.NoError(t, err)

	result := d.Resolve(tea.KeyPressMsg{Code: tea.KeyEnter}, []bindings.Scope{"revisions.squash"})
	require.True(t, result.Consumed)
	require.False(t, result.Pending)
	require.Equal(t, bindings.Action("apply"), result.Action)
	require.Equal(t, true, result.Args["force"])
}

func TestDispatcher_ResolvesThreeKeySequence(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "open_bookmarks", Scope: "ui", Seq: []string{"g", "p", "f"}},
	})
	require.NoError(t, err)

	first := d.Resolve(runeKey('g'), []bindings.Scope{"ui"})
	require.True(t, first.Consumed)
	require.True(t, first.Pending)
	require.Len(t, first.Continuations, 1)
	require.Equal(t, "p", first.Continuations[0].Key)

	second := d.Resolve(runeKey('p'), []bindings.Scope{"ui"})
	require.True(t, second.Consumed)
	require.True(t, second.Pending)
	require.Len(t, second.Continuations, 1)
	require.Equal(t, "f", second.Continuations[0].Key)

	third := d.Resolve(runeKey('f'), []bindings.Scope{"ui"})
	require.True(t, third.Consumed)
	require.False(t, third.Pending)
	require.Equal(t, bindings.Action("open_bookmarks"), third.Action)
}

func TestDispatcher_ResolvesFromDeepScopeChainByPrecedence(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "global_action", Scope: "ui", Key: []string{"x"}},
		{Action: "preview_action", Scope: "preview", Key: []string{"x"}},
		{Action: "quick_search_action", Scope: "quick_search", Key: []string{"x"}},
		{Action: "operation_action", Scope: "revisions.rebase", Key: []string{"x"}},
	})
	require.NoError(t, err)

	result := d.Resolve(runeKey('x'), []bindings.Scope{"revisions.rebase", "quick_search", "preview", "ui"})
	require.True(t, result.Consumed)
	require.False(t, result.Pending)
	require.Equal(t, bindings.Action("operation_action"), result.Action)
}

func TestDispatcher_SequencePrefixCollisionAcrossScopes_InnerScopeWins(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "global_git_push", Scope: "ui", Seq: []string{"g", "p"}},
		{Action: "revisions_git_push", Scope: "revisions", Seq: []string{"g", "p"}},
	})
	require.NoError(t, err)

	first := d.Resolve(runeKey('g'), []bindings.Scope{"revisions", "ui"})
	require.True(t, first.Consumed)
	require.True(t, first.Pending)

	second := d.Resolve(runeKey('p'), []bindings.Scope{"revisions", "ui"})
	require.True(t, second.Consumed)
	require.False(t, second.Pending)
	require.Equal(t, bindings.Action("revisions_git_push"), second.Action)
}

func TestDispatcher_DuplicateKeySameScope_LastBindingWins(t *testing.T) {
	d, err := NewDispatcher([]bindings.Binding{
		{Action: "first_action", Scope: "revisions", Key: []string{"x"}},
		{Action: "second_action", Scope: "revisions", Key: []string{"x"}},
	})
	require.NoError(t, err)

	result := d.Resolve(runeKey('x'), []bindings.Scope{"revisions"})
	require.True(t, result.Consumed)
	require.False(t, result.Pending)
	require.Equal(t, bindings.Action("second_action"), result.Action)
}

func runeKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: string(r), Code: r}
}
