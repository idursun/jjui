package dispatch

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// Result is the outcome of full dispatch resolution.
type Result struct {
	Intent        intents.Intent
	Scope         string
	Args          map[string]any
	LuaScript     string
	Pending       bool
	Consumed      bool
	Continuations []Continuation
}

// Resolver wraps a Dispatcher and extends the pipeline to resolve
// keys all the way to intents and scopes.
type Resolver struct {
	dispatcher        *Dispatcher
	configuredActions map[keybindings.Action]config.ActionConfig
}

// NewResolver creates a Resolver that wraps the given dispatcher.
func NewResolver(d *Dispatcher) *Resolver {
	return newResolverWithActions(d, config.Current.Actions)
}

func newResolverWithActions(d *Dispatcher, actions []config.ActionConfig) *Resolver {
	configured := make(map[keybindings.Action]config.ActionConfig, len(actions))
	for _, action := range actions {
		name := keybindings.Action(strings.TrimSpace(action.Name))
		if name == "" {
			continue
		}
		configured[name] = action
	}
	return &Resolver{
		dispatcher:        d,
		configuredActions: configured,
	}
}

// ResolveKey resolves a key press through the full pipeline: key → binding → action → intent.
func (r *Resolver) ResolveKey(msg tea.KeyMsg, scopes []Scope) Result {
	if r.dispatcher == nil {
		return Result{}
	}

	bindResult := r.dispatcher.Resolve(msg, scopes)
	if bindResult.Pending {
		return Result{
			Pending:       true,
			Consumed:      true,
			Continuations: bindResult.Continuations,
		}
	}
	if bindResult.Action != "" {
		result := r.resolveAction(bindResult.Action, bindResult.Args, false)
		if bindResult.Scope != "" {
			result.Scope = string(bindResult.Scope)
		}
		return result
	}
	if bindResult.Consumed {
		return Result{Consumed: true}
	}
	return Result{}
}

// ResolveAction resolves a dispatched action through configured-action aliasing and intent resolution.
func (r *Resolver) ResolveAction(action keybindings.Action, args map[string]any) Result {
	return r.resolveAction(action, args, false)
}

// ResolveBuiltInAction resolves an action while skipping configured Lua overrides.
func (r *Resolver) ResolveBuiltInAction(action keybindings.Action, args map[string]any) Result {
	return r.resolveAction(action, args, true)
}

// ResetSequence resets any in-progress key sequence.
func (r *Resolver) ResetSequence() {
	if r.dispatcher != nil {
		r.dispatcher.ResetSequence()
	}
}

func (r *Resolver) resolveAction(action keybindings.Action, args map[string]any, skipConfigured bool) Result {
	if !skipConfigured {
		// Configured Lua actions override built-ins during normal dispatch.
		cfg, hasCfg := r.configuredActions[action]
		if hasCfg {
			if script := strings.TrimSpace(cfg.Lua); script != "" {
				return Result{LuaScript: script, Consumed: true}
			}
			// Configured action with no Lua is invalid config; treat as consumed no-op.
			return Result{Consumed: true}
		}
	}

	// try catalog resolution
	if intent, scope, ok := r.resolveFromCatalog(action, args); ok {
		return Result{Intent: intent, Scope: scope, Args: args, Consumed: true}
	}

	// 4. Unresolved — action was matched by binding but has no handler
	return Result{Consumed: false}
}

func (r *Resolver) resolveFromCatalog(action keybindings.Action, args map[string]any) (intents.Intent, string, bool) {
	intent, ok := actions.ResolveByAction(action, args)
	if !ok {
		return nil, "", false
	}

	scope := DeriveScope(action)
	return intent, scope, true
}
