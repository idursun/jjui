package scripting

import (
	"fmt"
	"strings"

	"github.com/idursun/jjui/internal/config"
	uicontext "github.com/idursun/jjui/internal/ui/context"
	lua "github.com/yuin/gopher-lua"
)

const (
	actionRegistryName = "__jjui_actions"
	actionCounterName  = "__jjui_action_counter"
)

func InitVM(ctx *uicontext.MainContext) error {
	if ctx == nil {
		return fmt.Errorf("lua vm: context is nil")
	}
	CloseVM(ctx)

	L := lua.NewState()
	registerAPI(L, ctx)
	ensureActionRegistry(L)
	L.SetGlobal(actionCounterName, lua.LNumber(0))
	ctx.ScriptVM = L
	return nil
}

func CloseVM(ctx *uicontext.MainContext) {
	if ctx == nil {
		return
	}
	if ctx.ScriptVM != nil {
		ctx.ScriptVM.Close()
		ctx.ScriptVM = nil
	}
}

func RunSetup(ctx *uicontext.MainContext, source string) ([]config.ActionConfig, []config.BindingConfig, error) {
	if source == "" {
		return nil, nil, nil
	}

	L, err := vmFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	var actions []config.ActionConfig
	var bindings []config.BindingConfig
	registry := ensureActionRegistry(L)

	configTable := L.NewTable()
	configTable.RawSetString("action", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		fn := L.CheckFunction(2)

		desc := ""
		scope := ""
		var key []string
		var seq []string
		hasKey := false
		hasSeq := false
		if L.GetTop() >= 3 {
			optsVal := L.Get(3)
			if optsVal != lua.LNil {
				optsTbl, ok := optsVal.(*lua.LTable)
				if !ok {
					L.ArgError(3, "expected table or nil")
					return 0
				}
				if descVal := optsTbl.RawGetString("desc"); descVal != lua.LNil {
					descStr, ok := descVal.(lua.LString)
					if !ok {
						L.RaiseError("config.action: opts.desc must be a string")
						return 0
					}
					desc = descStr.String()
				}
				if scopeVal := optsTbl.RawGetString("scope"); scopeVal != lua.LNil {
					scopeStr, ok := scopeVal.(lua.LString)
					if !ok {
						L.RaiseError("config.action: opts.scope must be a string")
						return 0
					}
					scope = scopeStr.String()
				}
				if keyVal := optsTbl.RawGetString("key"); keyVal != lua.LNil {
					parsed, err := stringListFromValue(keyVal, "opts.key")
					if err != nil {
						L.RaiseError("config.action: %s", err.Error())
						return 0
					}
					key = parsed
					hasKey = true
				}
				if seqVal := optsTbl.RawGetString("seq"); seqVal != lua.LNil {
					parsed, err := stringListFromValue(seqVal, "opts.seq")
					if err != nil {
						L.RaiseError("config.action: %s", err.Error())
						return 0
					}
					seq = parsed
					hasSeq = true
				}
			}
		}
		if hasKey && hasSeq {
			L.RaiseError("config.action: opts.key and opts.seq are mutually exclusive")
			return 0
		}
		if hasKey && len(key) == 0 {
			L.RaiseError("config.action: opts.key must not be empty")
			return 0
		}
		if hasSeq && len(seq) == 0 {
			L.RaiseError("config.action: opts.seq must not be empty")
			return 0
		}
		if (hasKey || hasSeq) && strings.TrimSpace(scope) == "" {
			L.RaiseError("config.action: opts.scope is required when opts.key or opts.seq is set")
			return 0
		}

		counter := int(lua.LVAsNumber(L.GetGlobal(actionCounterName)))
		counter++
		L.SetGlobal(actionCounterName, lua.LNumber(counter))

		id := fmt.Sprintf("action_%d", counter)
		registry.RawSetString(id, fn)
		actions = append(actions, config.ActionConfig{
			Name: name,
			Desc: desc,
			Lua:  fmt.Sprintf(`%s[%q]()`, actionRegistryName, id),
		})
		if hasKey || hasSeq {
			binding := config.BindingConfig{
				Action: name,
				Scope:  scope,
			}
			if len(key) > 0 {
				binding.Key = config.StringList(key)
			}
			if len(seq) > 0 {
				binding.Seq = config.StringList(seq)
			}
			bindings = append(bindings, binding)
		}
		return 0
	}))
	configTable.RawSetString("bind", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		binding := config.BindingConfig{
			Action: stringFieldFromTable(tbl, "action"),
			Scope:  stringFieldFromTable(tbl, "scope"),
		}
		if key := stringListFieldFromTable(tbl, "key"); len(key) > 0 {
			binding.Key = config.StringList(key)
		}
		if seq := stringListFieldFromTable(tbl, "seq"); len(seq) > 0 {
			binding.Seq = config.StringList(seq)
		}
		bindings = append(bindings, binding)
		return 0
	}))

	if err := L.DoString(source); err != nil {
		return nil, nil, fmt.Errorf("config.lua: %w", err)
	}

	setupFn := L.GetGlobal("setup")
	if setupFn == lua.LNil {
		return nil, nil, nil
	}
	fn, ok := setupFn.(*lua.LFunction)
	if !ok {
		return nil, nil, fmt.Errorf("config.lua: setup is not a function")
	}
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, configTable); err != nil {
		return nil, nil, fmt.Errorf("config.lua: setup(): %w", err)
	}

	return actions, bindings, nil
}

func vmFromContext(ctx *uicontext.MainContext) (*lua.LState, error) {
	if ctx == nil {
		return nil, fmt.Errorf("lua vm: context is nil")
	}
	if ctx.ScriptVM == nil {
		return nil, fmt.Errorf("lua vm is not initialized")
	}
	return ctx.ScriptVM, nil
}

func ensureActionRegistry(L *lua.LState) *lua.LTable {
	if existing, ok := L.GetGlobal(actionRegistryName).(*lua.LTable); ok {
		return existing
	}
	tbl := L.NewTable()
	L.SetGlobal(actionRegistryName, tbl)
	return tbl
}

func stringFieldFromTable(tbl *lua.LTable, key string) string {
	v := tbl.RawGetString(key)
	if s, ok := v.(lua.LString); ok {
		return s.String()
	}
	return ""
}

func stringListFieldFromTable(tbl *lua.LTable, key string) []string {
	v := tbl.RawGetString(key)
	switch vv := v.(type) {
	case lua.LString:
		return []string{vv.String()}
	case *lua.LTable:
		return stringSliceFromTable(vv)
	default:
		return nil
	}
}

func stringListFromValue(v lua.LValue, field string) ([]string, error) {
	switch vv := v.(type) {
	case lua.LString:
		return []string{vv.String()}, nil
	case *lua.LTable:
		var out []string
		var badType string
		vv.ForEach(func(_, value lua.LValue) {
			if badType != "" {
				return
			}
			s, ok := value.(lua.LString)
			if !ok {
				badType = value.Type().String()
				return
			}
			out = append(out, s.String())
		})
		if badType != "" {
			return nil, fmt.Errorf("%s must be a string or array of strings (got array element %s)", field, badType)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be a string or array of strings", field)
	}
}
