package scripting

import (
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/ui/common"
	uicontext "github.com/idursun/jjui/internal/ui/context"
)

func strPtr(v string) *string {
	return &v
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
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}},
			want: strPtr("abc123"),
		},
		{
			name: "selected file",
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedFile{
				ChangeId: "file123",
				CommitId: "commit456",
				File:     "test.go",
			}},
			want: strPtr("file123"),
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
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}},
			want: strPtr("def456"),
		},
		{
			name: "selected file",
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedFile{
				ChangeId: "file123",
				CommitId: "commit456",
				File:     "test.go",
			}},
			want: strPtr("commit456"),
		},
		{
			name: "selected commit",
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedCommit{
				CommitId: "onlycommit789",
			}},
			want: strPtr("onlycommit789"),
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
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedFile{
				ChangeId: "file123",
				CommitId: "commit456",
				File:     "path/to/file.go",
			}},
			want: strPtr("path/to/file.go"),
		},
		{
			name: "selected revision",
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}},
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
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedOperation{
				OperationId: "op123456",
			}},
			want: strPtr("op123456"),
		},
		{
			name: "selected revision",
			ctx: &uicontext.MainContext{SelectedItem: uicontext.SelectedRevision{
				ChangeId: "abc123",
				CommitId: "def456",
			}},
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

func TestContext_CheckedFiles(t *testing.T) {
	ctx := &uicontext.MainContext{
		CheckedItems: []uicontext.SelectedItem{
			uicontext.SelectedFile{ChangeId: "c1", CommitId: "co1", File: "file1.go"},
			uicontext.SelectedFile{ChangeId: "c2", CommitId: "co2", File: "file2.go"},
			uicontext.SelectedRevision{ChangeId: "rev1", CommitId: "com1"}, // should be ignored
			uicontext.SelectedFile{ChangeId: "c3", CommitId: "co3", File: "file3.go"},
		},
	}

	vals := runScriptAndGetGlobals(t, ctx, `
		files = context.checked_files()
		count = #files
		first = files[1]
		second = files[2]
		third = files[3]
	`, "count", "first", "second", "third")

	assert.Equal(t, lua.LNumber(3), vals[0])
	assert.Equal(t, "file1.go", vals[1].String())
	assert.Equal(t, "file2.go", vals[2].String())
	assert.Equal(t, "file3.go", vals[3].String())
}

func TestContext_CheckedFiles_Empty(t *testing.T) {
	ctx := &uicontext.MainContext{
		CheckedItems: []uicontext.SelectedItem{},
	}

	val := runScriptAndGetGlobal(t, ctx, `
		files = context.checked_files()
		result = #files
	`, "result")

	assert.Equal(t, lua.LNumber(0), val)
}

func TestContext_CheckedChangeIds(t *testing.T) {
	ctx := &uicontext.MainContext{
		CheckedItems: []uicontext.SelectedItem{
			uicontext.SelectedRevision{ChangeId: "change1", CommitId: "com1"},
			uicontext.SelectedFile{ChangeId: "change2", CommitId: "com2", File: "f.go"},
			uicontext.SelectedOperation{OperationId: "op1"}, // should be ignored
			uicontext.SelectedRevision{ChangeId: "change3", CommitId: "com3"},
		},
	}

	vals := runScriptAndGetGlobals(t, ctx, `
		ids = context.checked_change_ids()
		count = #ids
		first = ids[1]
		second = ids[2]
		third = ids[3]
	`, "count", "first", "second", "third")

	assert.Equal(t, lua.LNumber(3), vals[0])
	assert.Equal(t, "change1", vals[1].String())
	assert.Equal(t, "change2", vals[2].String())
	assert.Equal(t, "change3", vals[3].String())
}

func TestContext_CheckedCommitIds(t *testing.T) {
	ctx := &uicontext.MainContext{
		CheckedItems: []uicontext.SelectedItem{
			uicontext.SelectedRevision{ChangeId: "c1", CommitId: "commit1"},
			uicontext.SelectedFile{ChangeId: "c2", CommitId: "commit2", File: "f.go"},
			uicontext.SelectedCommit{CommitId: "commit3"},
			uicontext.SelectedOperation{OperationId: "op1"}, // should be ignored
		},
	}

	vals := runScriptAndGetGlobals(t, ctx, `
		ids = context.checked_commit_ids()
		count = #ids
		first = ids[1]
		second = ids[2]
		third = ids[3]
	`, "count", "first", "second", "third")

	assert.Equal(t, lua.LNumber(3), vals[0])
	assert.Equal(t, "commit1", vals[1].String())
	assert.Equal(t, "commit2", vals[2].String())
	assert.Equal(t, "commit3", vals[3].String())
}

func TestContext_AccessViaJjuiNamespace(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedRevision{
			ChangeId: "ns_change",
			CommitId: "ns_commit",
		},
	}

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
