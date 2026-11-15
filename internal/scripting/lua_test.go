package scripting

import (
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	uicontext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
)

type testStateProvider struct {
	snapshot common.SelectionSnapshot
	states   map[string]any
}

func TestStateGetterTracksProviderReplacementAndRejectsUnsupportedValues(t *testing.T) {
	ctx := &uicontext.MainContext{}
	L := lua.NewState()
	defer L.Close()
	registerAPI(L, ctx)
	read := func() lua.LValue {
		t.Helper()
		require.NoError(t, L.DoString(`result = jjui.revisions.inline_describe.content()`))
		return L.GetGlobal("result")
	}
	assert.Equal(t, lua.LNil, read())
	ctx.SetStateProvider(&testStateProvider{states: map[string]any{"revisions.inline_describe.content": "first"}})
	assert.Equal(t, lua.LString("first"), read())
	ctx.SetStateProvider(&testStateProvider{states: map[string]any{"revisions.inline_describe.content": "second"}})
	assert.Equal(t, lua.LString("second"), read())
	ctx.SetStateProvider(&testStateProvider{states: map[string]any{"revisions.inline_describe.content": struct{}{}}})
	err := L.DoString(`result = jjui.revisions.inline_describe.content()`)
	require.ErrorContains(t, err, "unsupported queried state type")
	ctx.SetStateProvider(nil)
	assert.Equal(t, lua.LNil, read())
}

func (p *testStateProvider) Selection() common.SelectionSnapshot { return p.snapshot }
func (p *testStateProvider) QueryState(name string) (any, bool) {
	value, ok := p.states[name]
	return value, ok
}

func contextWithSelection(selected common.SelectedItem, checked ...common.SelectedItem) *uicontext.MainContext {
	ctx := &uicontext.MainContext{}
	ctx.SetStateProvider(&testStateProvider{snapshot: common.SelectionSnapshot{
		Highlighted: selected,
		Checked:     checked,
	}})
	return ctx
}

func assertLuaStringOrNil(t *testing.T, val lua.LValue, expected *string) {
	t.Helper()
	if expected == nil {
		assert.Equal(t, lua.LNil, val)
		return
	}
	assert.Equal(t, *expected, val.String())
}

// runScriptAndGetGlobal runs a Lua script and returns the value of a global variable
// before the Lua state is closed.
func runScriptAndGetGlobal(t *testing.T, ctx *uicontext.MainContext, script, varName string) lua.LValue {
	results := runScriptAndGetGlobals(t, ctx, script, varName)
	if len(results) > 0 {
		return results[0]
	}
	return lua.LNil
}

func runScriptAndGetGlobals(t *testing.T, ctx *uicontext.MainContext, script string, varNames ...string) []lua.LValue {
	L := lua.NewState()
	defer L.Close()

	registerAPI(L, ctx)

	fn, err := L.LoadString(script)
	assert.NoError(t, err)

	err = L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	})
	assert.NoError(t, err)

	var results []lua.LValue
	for _, name := range varNames {
		results = append(results, L.GetGlobal(name))
	}
	return results
}

func TestContext_ChangeId(t *testing.T) {
	tests := []struct {
		name string
		ctx  *uicontext.MainContext
		want *string
	}{
		{
			name: "selected revision",
			ctx: contextWithSelection(uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}),
			want: new("abc123"),
		},
		{
			name: "selected file",
			ctx: contextWithSelection(uicontext.SelectedFile{
				ChangeId: "file123",
				CommitId: "commit456",
				File:     jj.NewFileName("test.go"),
			}),
			want: new("file123"),
		},
		{
			name: "no selection",
			ctx:  &uicontext.MainContext{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := runScriptAndGetGlobal(t, tt.ctx, `result = context.change_id()`, "result")
			assertLuaStringOrNil(t, val, tt.want)
		})
	}
}

func TestContext_CommitId(t *testing.T) {
	tests := []struct {
		name string
		ctx  *uicontext.MainContext
		want *string
	}{
		{
			name: "selected revision",
			ctx: contextWithSelection(uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}),
			want: new("def456"),
		},
		{
			name: "selected file",
			ctx: contextWithSelection(uicontext.SelectedFile{
				ChangeId: "file123",
				CommitId: "commit456",
				File:     jj.NewFileName("test.go"),
			}),
			want: new("commit456"),
		},
		{
			name: "selected commit",
			ctx: contextWithSelection(uicontext.SelectedCommit{
				CommitId: "onlycommit789",
			}),
			want: new("onlycommit789"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := runScriptAndGetGlobal(t, tt.ctx, `result = context.commit_id()`, "result")
			assertLuaStringOrNil(t, val, tt.want)
		})
	}
}

func TestContext_File(t *testing.T) {
	tests := []struct {
		name string
		ctx  *uicontext.MainContext
		want *string
	}{
		{
			name: "selected file",
			ctx: contextWithSelection(uicontext.SelectedFile{
				ChangeId: "file123",
				CommitId: "commit456",
				File:     jj.NewFileName("path/to/file.go"),
			}),
			want: new("path/to/file.go"),
		},
		{
			name: "selected revision",
			ctx: contextWithSelection(uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := runScriptAndGetGlobal(t, tt.ctx, `result = context.file()`, "result")
			assertLuaStringOrNil(t, val, tt.want)
		})
	}
}

func TestContext_OperationId(t *testing.T) {
	tests := []struct {
		name string
		ctx  *uicontext.MainContext
		want *string
	}{
		{
			name: "selected operation",
			ctx: contextWithSelection(uicontext.SelectedOperation{
				OperationId: "op123456",
			}),
			want: new("op123456"),
		},
		{
			name: "selected revision",
			ctx: contextWithSelection(uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := runScriptAndGetGlobal(t, tt.ctx, `result = context.operation_id()`, "result")
			assertLuaStringOrNil(t, val, tt.want)
		})
	}
}

func TestContext_CheckedObjects(t *testing.T) {
	ctx := &uicontext.MainContext{}
	ctx.SetStateProvider(&testStateProvider{snapshot: common.SelectionSnapshot{
		CheckedObjects: []common.SelectedRevision{
			{ChangeId: "change1", CommitId: "commit1", ObjectKind: "author", ObjectText: "ada@example.com"},
			{ChangeId: "change2", CommitId: "commit2", ObjectKind: "bookmark", ObjectText: "main", ObjectRemote: "origin"},
			{ChangeId: "change3", CommitId: "commit3", ObjectKind: "node", ObjectText: "change3"},
		},
	}})

	vals := runScriptAndGetGlobals(t, ctx, `
		objects = context.checked_objects()
		count = #objects
		kind = objects[1].kind
		value = objects[1].value
		change = objects[1].change_id
		remote1 = objects[1].remote
		remote2 = objects[2].remote
		commit2 = objects[2].commit_id
		kind3 = objects[3].kind
	`, "count", "kind", "value", "change", "remote1", "remote2", "commit2", "kind3")

	assert.Equal(t, lua.LNumber(3), vals[0])
	assert.Equal(t, "author", vals[1].String())
	assert.Equal(t, "ada@example.com", vals[2].String())
	assert.Equal(t, "change1", vals[3].String())
	assert.Equal(t, lua.LNil, vals[4])
	assert.Equal(t, "origin", vals[5].String())
	assert.Equal(t, "commit2", vals[6].String())
	assert.Equal(t, "revision", vals[7].String())
}

func TestContext_ObjectIsTheRevisionWithoutAFocusedObject(t *testing.T) {
	for name, ctx := range map[string]*uicontext.MainContext{
		"node":      contextWithSelection(uicontext.SelectedRevision{ChangeId: "abc", CommitId: "def", ObjectKind: "node", ObjectValue: "○", ObjectText: "○"}),
		"no object": contextWithSelection(uicontext.SelectedRevision{ChangeId: "abc", CommitId: "def"}),
	} {
		got := runScriptAndGetGlobals(t, ctx, `
			local o = context.object()
			kind, value, change, commit = o.kind, o.value, o.change_id, o.commit_id
		`, "kind", "value", "change", "commit")
		assert.Equal(t, []lua.LValue{lua.LString("revision"), lua.LString("abc"), lua.LString("abc"), lua.LString("def")}, got, name)
	}
}

func TestContext_ObjectDescribesFilesAndCommits(t *testing.T) {
	ctx := contextWithSelection(uicontext.SelectedFile{ChangeId: "abc", CommitId: "def", File: jj.NewFileName("a.go")})
	got := runScriptAndGetGlobals(t, ctx, `
		local o = context.object()
		kind, value, change, commit = o.kind, o.value, o.change_id, o.commit_id
	`, "kind", "value", "change", "commit")
	assert.Equal(t, []lua.LValue{lua.LString("file"), lua.LString("a.go"), lua.LString("abc"), lua.LString("def")}, got)

	ctx = contextWithSelection(uicontext.SelectedCommit{CommitId: "def"})
	got = runScriptAndGetGlobals(t, ctx, `
		local o = context.object()
		kind, change, commit = o.kind, o.change_id, o.commit_id
	`, "kind", "change", "commit")
	assert.Equal(t, []lua.LValue{lua.LString("commit"), lua.LNil, lua.LString("def")}, got)
}

func TestContext_ObjectIsNilWithoutASelection(t *testing.T) {
	for name, ctx := range map[string]*uicontext.MainContext{
		"operation":    contextWithSelection(uicontext.SelectedOperation{OperationId: "op1"}),
		"no selection": {},
	} {
		got := runScriptAndGetGlobal(t, ctx, `o = context.object()`, "o")
		assert.Equal(t, lua.LNil, got, name)
	}
}

func TestContext_AccessViaJjuiNamespace(t *testing.T) {
	ctx := contextWithSelection(uicontext.SelectedRevision{
		ChangeId: "ns_change",
		CommitId: "ns_commit",
	})

	vals := runScriptAndGetGlobals(t, ctx, `
		change = jjui.context.change_id()
		commit = jjui.context.commit_id()
	`, "change", "commit")

	assert.Equal(t, "ns_change", vals[0].String())
	assert.Equal(t, "ns_commit", vals[1].String())
}

func TestGeneratedActions_AccessViaJjuiNamespace(t *testing.T) {
	ctx := &uicontext.MainContext{}

	vals := runScriptAndGetGlobals(t, ctx, `
		result_details = type(jjui.revisions.details.cancel)
		result_nested = type(jjui.revisions.details.confirmation.apply)
		result_builtin = type(jjui.builtin.revisions.details.cancel)
		result_legacy = type(jjui.action)
	`, "result_details", "result_nested", "result_builtin", "result_legacy")

	assert.Equal(t, "function", vals[0].String())
	assert.Equal(t, "function", vals[1].String())
	assert.Equal(t, "function", vals[2].String())
	assert.Equal(t, "nil", vals[3].String())
}

func TestWaitHelpers_AccessViaTopLevelAndJjuiNamespace(t *testing.T) {
	ctx := &uicontext.MainContext{}

	vals := runScriptAndGetGlobals(t, ctx, `
		top_close = type(wait_close)
		top_refresh = type(wait_refresh)
		ns_close = type(jjui.wait_close)
		ns_refresh = type(jjui.wait_refresh)
	`, "top_close", "top_refresh", "ns_close", "ns_refresh")

	assert.Equal(t, "function", vals[0].String())
	assert.Equal(t, "function", vals[1].String())
	assert.Equal(t, "function", vals[2].String())
	assert.Equal(t, "function", vals[3].String())
}

func TestSetThemeDispatchesChangeThemeIntent(t *testing.T) {
	ctx := setupVM(t)

	runner, cmd, err := RunScript(ctx, `set_theme("runtime_dark")`)
	require.NoError(t, err)
	require.NotNil(t, runner)
	require.NotNil(t, cmd)

	msg := cmd()
	intent, ok := msg.(intents.ChangeTheme)
	require.True(t, ok)
	assert.Equal(t, "runtime_dark", intent.Name)
}

func TestSetThemeAccessibleViaJjuiNamespace(t *testing.T) {
	ctx := setupVM(t)

	runner, cmd, err := RunScript(ctx, `jjui.set_theme("runtime_light")`)
	require.NoError(t, err)
	require.NotNil(t, runner)
	require.NotNil(t, cmd)

	msg := cmd()
	intent, ok := msg.(intents.ChangeTheme)
	require.True(t, ok)
	assert.Equal(t, "runtime_light", intent.Name)
}

func runWaitingScript(t *testing.T, script string) (*uicontext.MainContext, *Runner) {
	t.Helper()

	ctx := setupVM(t)
	runner, cmd, err := RunScript(ctx, script)
	require.NoError(t, err)
	require.NotNil(t, runner)
	if cmd != nil {
		_ = cmd()
	}
	require.False(t, runner.Done())

	return ctx, runner
}

func TestRunner_WaitClose(t *testing.T) {
	t.Run("ignores non-close messages and resumes on close", func(t *testing.T) {
		ctx, runner := runWaitingScript(t, `
			done = false
			applied = wait_close()
			done = true
		`)

		cmd := runner.HandleMsg(common.UpdateRevisionsSuccessMsg{})
		assert.Nil(t, cmd)
		assert.False(t, runner.Done())
		assert.Equal(t, "false", ctx.ScriptVM.GetGlobal("done").String())

		cmd = runner.HandleMsg(common.CloseViewMsg{Applied: true})
		if cmd != nil {
			_ = cmd()
		}
		assert.True(t, runner.Done())
		assert.Equal(t, "true", ctx.ScriptVM.GetGlobal("done").String())
		assert.Equal(t, "true", ctx.ScriptVM.GetGlobal("applied").String())
	})

	tests := []struct {
		name    string
		applied bool
	}{
		{name: "applied", applied: true},
		{name: "cancelled", applied: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, runner := runWaitingScript(t, `applied = wait_close()`)

			cmd := runner.HandleMsg(common.CloseViewMsg{Applied: tt.applied})
			if cmd != nil {
				_ = cmd()
			}

			assert.True(t, runner.Done())
			assert.Equal(t, tt.applied, ctx.ScriptVM.GetGlobal("applied") == lua.LTrue)
		})
	}
}

func TestRunner_WaitRefreshResumesOnUpdateRevisions(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{name: "success", msg: common.UpdateRevisionsSuccessMsg{}},
		{name: "failed", msg: common.UpdateRevisionsFailedMsg{Output: "err"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, runner := runWaitingScript(t, `
				done = false
				wait_refresh()
				done = true
			`)

			cmd := runner.HandleMsg(tt.msg)
			if cmd != nil {
				_ = cmd()
			}
			assert.True(t, runner.Done())
			assert.Equal(t, "true", ctx.ScriptVM.GetGlobal("done").String())
		})
	}
}

func TestContext_Object(t *testing.T) {
	ctx := contextWithSelection(uicontext.SelectedRevision{
		ChangeId: "abc", CommitId: "def",
		ObjectKind: "bookmark", ObjectValue: `"a b"@origin`,
		ObjectText: "a b", ObjectRemote: "origin", ObjectConflicted: true,
	})
	got := runScriptAndGetGlobals(t, ctx, `
		local o = context.object()
		kind, value, remote, conflicted = o.kind, o.value, o.remote, o.conflicted
	`, "kind", "value", "remote", "conflicted")
	assert.Equal(t, []lua.LValue{lua.LString("bookmark"), lua.LString("a b"), lua.LString("origin"), lua.LTrue}, got)

	ctx = contextWithSelection(uicontext.SelectedRevision{ObjectKind: "remote", ObjectValue: `exact:"origin"`, ObjectText: "origin"})
	got = runScriptAndGetGlobals(t, ctx, `
		local o = context.object()
		kind, value, remote, conflicted = o.kind, o.value, o.remote, o.conflicted
	`, "kind", "value", "remote", "conflicted")
	assert.Equal(t, []lua.LValue{lua.LString("remote"), lua.LString("origin"), lua.LNil, lua.LNil}, got)
}
