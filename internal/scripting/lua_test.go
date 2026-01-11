package scripting

import (
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/stretchr/testify/assert"

	uicontext "github.com/idursun/jjui/internal/ui/context"
)

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

	r := &Runner{ctx: ctx, main: L}
	registerAPI(L, r)

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

func TestContext_ChangeId_WithSelectedRevision(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedRevision{
			ChangeId: "abc123",
			CommitId: "def456",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.change_id()`, "result")
	assert.Equal(t, "abc123", val.String())
}

func TestContext_ChangeId_WithSelectedFile(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedFile{
			ChangeId: "file123",
			CommitId: "commit456",
			File:     "test.go",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.change_id()`, "result")
	assert.Equal(t, "file123", val.String())
}

func TestContext_ChangeId_WithNoSelection(t *testing.T) {
	ctx := &uicontext.MainContext{}

	val := runScriptAndGetGlobal(t, ctx, `result = context.change_id()`, "result")
	assert.Equal(t, lua.LNil, val)
}

func TestContext_CommitId_WithSelectedRevision(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedRevision{
			ChangeId: "abc123",
			CommitId: "def456",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.commit_id()`, "result")
	assert.Equal(t, "def456", val.String())
}

func TestContext_CommitId_WithSelectedFile(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedFile{
			ChangeId: "file123",
			CommitId: "commit456",
			File:     "test.go",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.commit_id()`, "result")
	assert.Equal(t, "commit456", val.String())
}

func TestContext_CommitId_WithSelectedCommit(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedCommit{
			CommitId: "onlycommit789",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.commit_id()`, "result")
	assert.Equal(t, "onlycommit789", val.String())
}

func TestContext_File_WithSelectedFile(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedFile{
			ChangeId: "file123",
			CommitId: "commit456",
			File:     "path/to/file.go",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.file()`, "result")
	assert.Equal(t, "path/to/file.go", val.String())
}

func TestContext_File_WithSelectedRevision(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedRevision{
			ChangeId: "abc123",
			CommitId: "def456",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.file()`, "result")
	assert.Equal(t, lua.LNil, val)
}

func TestContext_OperationId_WithSelectedOperation(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedOperation{
			OperationId: "op123456",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.operation_id()`, "result")
	assert.Equal(t, "op123456", val.String())
}

func TestContext_OperationId_WithSelectedRevision(t *testing.T) {
	ctx := &uicontext.MainContext{
		SelectedItem: uicontext.SelectedRevision{
			ChangeId: "abc123",
			CommitId: "def456",
		},
	}

	val := runScriptAndGetGlobal(t, ctx, `result = context.operation_id()`, "result")
	assert.Equal(t, lua.LNil, val)
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
