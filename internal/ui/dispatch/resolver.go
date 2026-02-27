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
	Owner         string
	Args          map[string]any
	LuaScript     string
	Pending       bool
	Consumed      bool
	Continuations []Continuation
}

// IntentOverride lets the caller (e.g. active operation) override
// the default action-to-intent mapping.
type IntentOverride func(action keybindings.Action, args map[string]any) (intents.Intent, bool)

// Resolver wraps a Dispatcher and extends the pipeline to resolve
// keys all the way to intents and owners.
type Resolver struct {
	dispatcher        *Dispatcher
	configuredActions map[keybindings.Action]config.ActionConfig
}

// NewResolver creates a Resolver that wraps the given dispatcher.
func NewResolver(d *Dispatcher, configured map[keybindings.Action]config.ActionConfig) *Resolver {
	return &Resolver{
		dispatcher:        d,
		configuredActions: configured,
	}
}

// ResolveKey resolves a key press through the full pipeline: key → binding → action → intent.
func (r *Resolver) ResolveKey(msg tea.KeyMsg, scopes []keybindings.Scope, override IntentOverride) Result {
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
		return r.resolveAction(bindResult.Scope, bindResult.Action, bindResult.Args, override)
	}
	if bindResult.Consumed {
		return Result{Consumed: true}
	}
	return Result{}
}

// ResolveAction resolves a dispatched action through configured-action aliasing and intent resolution.
func (r *Resolver) ResolveAction(scope keybindings.Scope, action keybindings.Action, args map[string]any, override IntentOverride) Result {
	return r.resolveAction(scope, action, args, override)
}

// ResetSequence resets any in-progress key sequence.
func (r *Resolver) ResetSequence() {
	if r.dispatcher != nil {
		r.dispatcher.ResetSequence()
	}
}

func (r *Resolver) resolveAction(scope keybindings.Scope, action keybindings.Action, args map[string]any, override IntentOverride) Result {
	// 1. Try operation override first
	if override != nil {
		if intent, ok := override(action, args); ok {
			owner := DeriveOwner(action)
			return Result{Intent: intent, Owner: owner, Args: args, Consumed: true}
		}
	}

	// 2. Try catalog resolution
	if intent, owner, ok := r.resolveFromCatalog(action, args); ok {
		return Result{Intent: intent, Owner: owner, Args: args, Consumed: true}
	}

	// 3. Try configured actions (Lua only).
	cfg, hasCfg := r.configuredActions[action]
	if hasCfg {
		if script := strings.TrimSpace(cfg.Lua); script != "" {
			return Result{LuaScript: script, Consumed: true}
		}
		// Configured action with no Lua is invalid config; treat as consumed no-op.
		return Result{Consumed: true}
	}

	// 4. Unresolved — action was matched by binding but has no handler
	return Result{Consumed: false}
}

func (r *Resolver) resolveFromCatalog(action keybindings.Action, args map[string]any) (intents.Intent, string, bool) {
	intent, ok := actions.ResolveByAction(action, args)
	if !ok {
		return nil, "", false
	}

	owner := DeriveOwner(action)
	return intent, owner, true
}
