package scripting

import (
	stdcontext "context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/choose"
	"github.com/idursun/jjui/internal/ui/common"
	uicontext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	lua "github.com/yuin/gopher-lua"
)

type step struct {
	cmd     tea.Cmd
	matcher func(tea.Msg) (bool, []lua.LValue)
}

type Runner struct {
	ctx        *uicontext.MainContext
	main       *lua.LState
	thread     *lua.LState
	cancel     stdcontext.CancelFunc
	fn         *lua.LFunction
	started    bool
	await      func(tea.Msg) (bool, []lua.LValue)
	resumeArgs []lua.LValue
	done       bool
}

func RunScript(ctx *uicontext.MainContext, src string) (*Runner, tea.Cmd, error) {
	L := lua.NewState()
	r := &Runner{ctx: ctx, main: L}

	registerAPI(L, r)

	fn, err := L.LoadString(src)
	if err != nil {
		L.Close()
		return nil, nil, fmt.Errorf("lua: %w", err)
	}
	r.fn = fn
	r.thread, r.cancel = L.NewThread()

	cmd := r.resume()
	if r.done {
		r.close()
	}
	return r, cmd, nil
}

func (r *Runner) close() {
	if r.main != nil {
		if r.cancel != nil {
			r.cancel()
			r.cancel = nil
		}
		r.main.Close()
		r.main = nil
	}
}

func (r *Runner) resume() tea.Cmd {
	if r.done {
		return nil
	}
	var cmds []tea.Cmd
	for {
		var fn *lua.LFunction
		if !r.started {
			fn = r.fn
		}
		args := r.resumeArgs
		r.resumeArgs = nil
		state, err, values := r.main.Resume(r.thread, fn, args...)
		r.started = true
		if err != nil {
			r.done = true
			cmds = append(cmds, intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err}))
			break
		}
		for _, v := range values {
			if ud, ok := v.(*lua.LUserData); ok {
				if st, ok := ud.Value.(step); ok {
					if st.matcher != nil {
						r.await = st.matcher
						if st.cmd != nil {
							cmds = append(cmds, st.cmd)
						}
						return tea.Sequence(cmds...)
					}
					if st.cmd != nil {
						cmds = append(cmds, st.cmd)
					}
				}
			}
		}
		if state == lua.ResumeOK {
			r.done = true
			break
		}
		// continue to resume to collect subsequent steps until an await or completion
		if len(cmds) > 0 {
			continue
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

// HandleMsg resumes the script if waiting for a matching message.
func (r *Runner) HandleMsg(msg tea.Msg) tea.Cmd {
	if r.await == nil {
		return nil
	}
	ok, resume := r.await(msg)
	if !ok {
		return nil
	}
	r.await = nil
	r.resumeArgs = resume
	cmd := r.resume()
	if r.done {
		r.close()
	}
	return cmd
}

func (r *Runner) Done() bool {
	return r.done && r.await == nil
}

func registerAPI(L *lua.LState, runner *Runner) {
	revisionsTable := L.NewTable()
	revisionsTable.RawSetString("current", L.NewFunction(func(L *lua.LState) int {
		if rev, ok := runner.ctx.SelectedItem.(uicontext.SelectedRevision); ok {
			L.Push(lua.LString(rev.ChangeId))
			return 1
		}
		return 0
	}))
	revisionsTable.RawSetString("checked", L.NewFunction(func(L *lua.LState) int {
		tbl := L.NewTable()
		for _, item := range runner.ctx.CheckedItems {
			if rev, ok := item.(uicontext.SelectedRevision); ok {
				tbl.Append(lua.LString(rev.ChangeId))
			}
		}
		L.Push(tbl)
		return 1
	}))
	revisionsTable.RawSetString("refresh", L.NewFunction(func(L *lua.LState) int {
		payload := payloadFromTop(L)
		intent := intents.Refresh{
			KeepSelections:   boolVal(payload, "keep_selections"),
			SelectedRevision: stringVal(payload, "selected_revision"),
		}
		return yieldStep(L, step{cmd: revisions.RevisionsCmd(intent), matcher: matchUpdateRevisionsSuccess})
	}))
	revisionsTable.RawSetString("navigate", L.NewFunction(func(L *lua.LState) int {
		payload := payloadFromTop(L)
		target := parseNavigateTarget(stringVal(payload, "target"))
		intent := intents.Navigate{
			Delta:      intVal(payload, "by"),
			IsPage:     boolVal(payload, "page"),
			Target:     target,
			ChangeID:   stringVal(payload, "to"),
			FallbackID: stringVal(payload, "fallback"),
		}
		if v, ok := payload["ensureView"]; ok {
			if b, ok := v.(bool); ok {
				intent.EnsureView = boolPtr(b)
			}
		}
		if v, ok := payload["allowStream"]; ok {
			if b, ok := v.(bool); ok {
				intent.AllowStream = boolPtr(b)
			}
		}
		return yieldStep(L, step{cmd: revisions.RevisionsCmd(intent)})
	}))
	revisionsTable.RawSetString("start_squash", L.NewFunction(func(L *lua.LState) int {
		payload := payloadFromTop(L)
		intent := intents.StartSquash{
			Files: stringSlice(payload, "files"),
		}
		return yieldStep(L, step{cmd: revisions.RevisionsCmd(intent)})
	}))
	revisionsTable.RawSetString("start_rebase", L.NewFunction(func(L *lua.LState) int {
		payload := payloadFromTop(L)
		intent := intents.StartRebase{
			Source: parseRebaseSource(stringVal(payload, "source")),
			Target: parseRebaseTarget(stringVal(payload, "target")),
		}
		return yieldStep(L, step{cmd: revisions.RevisionsCmd(intent)})
	}))
	revisionsTable.RawSetString("open_details", L.NewFunction(func(L *lua.LState) int {
		return yieldStep(L, step{cmd: revisions.RevisionsCmd(intents.OpenDetails{})})
	}))
	revisionsTable.RawSetString("start_inline_describe", L.NewFunction(func(L *lua.LState) int {
		return yieldStep(L, step{cmd: revisions.RevisionsCmd(intents.StartInlineDescribe{}), matcher: matchCloseViewMsg})
	}))

	revsetTable := L.NewTable()
	revsetTable.RawSetString("set", L.NewFunction(func(L *lua.LState) int {
		value := L.CheckString(1)
		return yieldStep(L, step{cmd: revset.RevsetCmd(intents.Set{Value: value})})
	}))
	revsetTable.RawSetString("reset", L.NewFunction(func(L *lua.LState) int {
		return yieldStep(L, step{cmd: revset.RevsetCmd(intents.Reset{})})
	}))
	revsetTable.RawSetString("current", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(runner.ctx.CurrentRevset))
		return 1
	}))
	revsetTable.RawSetString("default", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(runner.ctx.DefaultRevset))
		return 1
	}))

	contextTable := L.NewTable()
	contextTable.RawSetString("change_id", L.NewFunction(func(L *lua.LState) int {
		switch item := runner.ctx.SelectedItem.(type) {
		case uicontext.SelectedRevision:
			L.Push(lua.LString(item.ChangeId))
			return 1
		case uicontext.SelectedFile:
			L.Push(lua.LString(item.ChangeId))
			return 1
		}
		return 0
	}))
	contextTable.RawSetString("commit_id", L.NewFunction(func(L *lua.LState) int {
		switch item := runner.ctx.SelectedItem.(type) {
		case uicontext.SelectedRevision:
			L.Push(lua.LString(item.CommitId))
			return 1
		case uicontext.SelectedFile:
			L.Push(lua.LString(item.CommitId))
			return 1
		case uicontext.SelectedCommit:
			L.Push(lua.LString(item.CommitId))
			return 1
		}
		return 0
	}))
	contextTable.RawSetString("file", L.NewFunction(func(L *lua.LState) int {
		if item, ok := runner.ctx.SelectedItem.(uicontext.SelectedFile); ok {
			L.Push(lua.LString(item.File))
			return 1
		}
		return 0
	}))
	contextTable.RawSetString("operation_id", L.NewFunction(func(L *lua.LState) int {
		if item, ok := runner.ctx.SelectedItem.(uicontext.SelectedOperation); ok {
			L.Push(lua.LString(item.OperationId))
			return 1
		}
		return 0
	}))
	contextTable.RawSetString("checked_files", L.NewFunction(func(L *lua.LState) int {
		tbl := L.NewTable()
		for _, item := range runner.ctx.CheckedItems {
			if f, ok := item.(uicontext.SelectedFile); ok {
				tbl.Append(lua.LString(f.File))
			}
		}
		L.Push(tbl)
		return 1
	}))
	contextTable.RawSetString("checked_change_ids", L.NewFunction(func(L *lua.LState) int {
		tbl := L.NewTable()
		for _, item := range runner.ctx.CheckedItems {
			switch i := item.(type) {
			case uicontext.SelectedRevision:
				tbl.Append(lua.LString(i.ChangeId))
			case uicontext.SelectedFile:
				tbl.Append(lua.LString(i.ChangeId))
			}
		}
		L.Push(tbl)
		return 1
	}))
	contextTable.RawSetString("checked_commit_ids", L.NewFunction(func(L *lua.LState) int {
		tbl := L.NewTable()
		for _, item := range runner.ctx.CheckedItems {
			switch i := item.(type) {
			case uicontext.SelectedRevision:
				tbl.Append(lua.LString(i.CommitId))
			case uicontext.SelectedFile:
				tbl.Append(lua.LString(i.CommitId))
			case uicontext.SelectedCommit:
				tbl.Append(lua.LString(i.CommitId))
			}
		}
		L.Push(tbl)
		return 1
	}))

	jjAsyncFn := L.NewFunction(func(L *lua.LState) int {
		args := argsFromLua(L)
		return yieldStep(L, step{cmd: runner.ctx.RunCommand(args)})
	})
	jjInteractiveFn := L.NewFunction(func(L *lua.LState) int {
		args := argsFromLua(L)
		return yieldStep(L, step{cmd: runner.ctx.RunInteractiveCommand(args, nil)})
	})
	jjFn := L.NewFunction(func(L *lua.LState) int {
		args := argsFromLua(L)
		out, err := runner.ctx.RunCommandImmediate(args)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(out))
		L.Push(lua.LNil)
		return 2
	})
	flashFn := L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		return yieldStep(L, step{cmd: intents.Invoke(intents.AddMessage{Text: msg})})
	})
	copyToClipboardFn := L.NewFunction(func(L *lua.LState) int {
		text := L.CheckString(1)
		if err := clipboard.WriteAll(text); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LBool(true))
		L.Push(lua.LNil)
		return 2
	})
	execShellFn := L.NewFunction(func(L *lua.LState) int {
		command := L.CheckString(1)
		msg := common.ExecMsg{
			Line: command,
			Mode: common.ExecShell,
		}
		return yieldStep(L, step{cmd: exec_process.ExecLine(runner.ctx, msg)})
	})
	splitLinesFn := L.NewFunction(func(L *lua.LState) int {
		text := L.CheckString(1)
		keepEmpty := false
		if L.GetTop() >= 2 {
			keepEmpty = L.CheckBool(2)
		}
		tbl := L.NewTable()
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSuffix(line, "\r")
			if line == "" && !keepEmpty {
				continue
			}
			tbl.Append(lua.LString(line))
		}
		L.Push(tbl)
		return 1
	})
	chooseFn := L.NewFunction(func(L *lua.LState) int {
		var (
			options []string
			title   string
		)
		if L.GetTop() == 1 {
			if tbl, ok := L.Get(1).(*lua.LTable); ok {
				if optVal := tbl.RawGetString("options"); optVal != lua.LNil {
					if optTbl, ok := optVal.(*lua.LTable); ok {
						options = stringSliceFromTable(optTbl)
					} else if s, ok := optVal.(lua.LString); ok {
						options = []string{s.String()}
					}
				}
				if titleVal := tbl.RawGetString("title"); titleVal != lua.LNil {
					title = titleVal.String()
				}
				if options == nil {
					options = stringSliceFromTable(tbl)
				}
				return yieldStep(L, step{cmd: choose.ShowWithTitle(options, title), matcher: matchChoose})
			}
		}
		options = argsFromLua(L)
		return yieldStep(L, step{cmd: choose.ShowWithTitle(options, ""), matcher: matchChoose})
	})
	inputFn := L.NewFunction(func(L *lua.LState) int {
		var title, prompt string
		if L.GetTop() == 1 {
			if tbl, ok := L.Get(1).(*lua.LTable); ok {
				if titleVal := tbl.RawGetString("title"); titleVal != lua.LNil {
					title = titleVal.String()
				}
				if promptVal := tbl.RawGetString("prompt"); promptVal != lua.LNil {
					prompt = promptVal.String()
				}
				return yieldStep(L, step{cmd: input.ShowWithTitle(title, prompt), matcher: matchInput})
			}
		}
		return yieldStep(L, step{cmd: input.ShowWithTitle("", ""), matcher: matchInput})
	})

	// make sure we have a `jjui` namespace
	root := L.NewTable()
	root.RawSetString("revisions", revisionsTable)
	root.RawSetString("revset", revsetTable)
	root.RawSetString("context", contextTable)
	root.RawSetString("jj_async", jjAsyncFn)
	root.RawSetString("jj_interactive", jjInteractiveFn)
	root.RawSetString("jj", jjFn)
	root.RawSetString("flash", flashFn)
	root.RawSetString("copy_to_clipboard", copyToClipboardFn)
	root.RawSetString("exec_shell", execShellFn)
	root.RawSetString("split_lines", splitLinesFn)
	root.RawSetString("choose", chooseFn)
	root.RawSetString("input", inputFn)
	L.SetGlobal("jjui", root)

	// but also expose at the top level for convenience
	L.SetGlobal("revisions", revisionsTable)
	L.SetGlobal("revset", revsetTable)
	L.SetGlobal("context", contextTable)
	L.SetGlobal("jj_async", jjAsyncFn)
	L.SetGlobal("jj_interactive", jjInteractiveFn)
	L.SetGlobal("jj", jjFn)
	L.SetGlobal("flash", flashFn)
	L.SetGlobal("copy_to_clipboard", copyToClipboardFn)
	L.SetGlobal("exec_shell", execShellFn)
	L.SetGlobal("split_lines", splitLinesFn)
	L.SetGlobal("choose", chooseFn)
	L.SetGlobal("input", inputFn)
}

func payloadFromTop(L *lua.LState) map[string]any {
	if L.GetTop() >= 1 && L.CheckAny(1) != lua.LNil {
		if tbl, ok := L.Get(1).(*lua.LTable); ok {
			return luaTableToMap(tbl)
		}
	}
	return map[string]any{}
}

func boolVal(payload map[string]any, key string) bool {
	if v, ok := payload[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func intVal(payload map[string]any, key string) int {
	if v, ok := payload[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		case float32:
			return int(n)
		}
	}
	return 0
}

func stringVal(payload map[string]any, key string) string {
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func stringSlice(payload map[string]any, key string) []string {
	v, ok := payload[key]
	if !ok {
		return nil
	}
	var out []string
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		for _, item := range vv {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func argsFromLua(L *lua.LState) []string {
	if L.GetTop() == 0 {
		return nil
	}
	if tbl, ok := L.Get(1).(*lua.LTable); ok {
		return stringSliceFromTable(tbl)
	}
	var out []string
	top := L.GetTop()
	for i := 1; i <= top; i++ {
		out = append(out, L.CheckString(i))
	}
	return out
}

func stringSliceFromTable(tbl *lua.LTable) []string {
	var out []string
	tbl.ForEach(func(_, value lua.LValue) {
		if s, ok := value.(lua.LString); ok {
			out = append(out, s.String())
		}
	})
	return out
}

func luaTableToMap(tbl *lua.LTable) map[string]any {
	result := map[string]any{}
	tbl.ForEach(func(key, value lua.LValue) {
		if key.Type() != lua.LTString {
			return
		}
		result[key.String()] = luaValueToGo(value)
	})
	return result
}

func luaTableToSlice(tbl *lua.LTable) []any {
	var result []any
	tbl.ForEach(func(_, value lua.LValue) {
		result = append(result, luaValueToGo(value))
	})
	return result
}

func luaValueToGo(value lua.LValue) any {
	switch value.Type() {
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTNumber:
		return float64(value.(lua.LNumber))
	case lua.LTString:
		return value.String()
	case lua.LTTable:
		t := value.(*lua.LTable)
		// Heuristic: if keys are string, convert to map; otherwise, slice.
		isMap := false
		t.ForEach(func(key, _ lua.LValue) {
			if key.Type() == lua.LTString {
				isMap = true
			}
		})
		if isMap {
			return luaTableToMap(t)
		}
		return luaTableToSlice(t)
	default:
		return nil
	}
}

func yieldStep(L *lua.LState, st step) int {
	ud := L.NewUserData()
	ud.Value = st
	return L.Yield(ud)
}

func boolPtr(v bool) *bool {
	return &v
}

func parseNavigateTarget(val string) intents.NavigationTarget {
	switch strings.ToLower(val) {
	case "parent":
		return intents.TargetParent
	case "child", "children":
		return intents.TargetChild
	case "working", "working_copy", "work":
		return intents.TargetWorkingCopy
	default:
		return intents.TargetNone
	}
}

func matchUpdateRevisionsSuccess(msg tea.Msg) (bool, []lua.LValue) {
	switch msg.(type) {
	case common.UpdateRevisionsSuccessMsg, common.UpdateRevisionsFailedMsg:
		return true, nil
	default:
		return false, nil
	}
}

func parseRebaseSource(val string) rebase.Source {
	switch strings.ToLower(val) {
	case "branch":
		return rebase.SourceBranch
	case "descendants", "source":
		return rebase.SourceDescendants
	default:
		return rebase.SourceRevision
	}
}

func parseRebaseTarget(val string) rebase.Target {
	switch strings.ToLower(val) {
	case "after":
		return rebase.TargetAfter
	case "before":
		return rebase.TargetBefore
	case "insert":
		return rebase.TargetInsert
	default:
		return rebase.TargetDestination
	}
}

func matchCloseViewMsg(msg tea.Msg) (bool, []lua.LValue) {
	if closeMsg, ok := msg.(common.CloseViewMsg); ok {
		return true, []lua.LValue{lua.LBool(closeMsg.Applied)}
	}
	return false, nil
}

func matchChoose(msg tea.Msg) (bool, []lua.LValue) {
	switch msg := msg.(type) {
	case choose.SelectedMsg:
		return true, []lua.LValue{lua.LString(msg.Value)}
	case choose.CancelledMsg:
		return true, []lua.LValue{lua.LNil}
	default:
		return false, nil
	}
}

func matchInput(msg tea.Msg) (bool, []lua.LValue) {
	switch msg := msg.(type) {
	case input.SelectedMsg:
		return true, []lua.LValue{lua.LString(msg.Value)}
	case input.CancelledMsg:
		return true, []lua.LValue{lua.LNil}
	default:
		return false, nil
	}
}
