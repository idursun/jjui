package dispatch

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
)

// Continuation describes possible next keys while in sequence mode.
type Continuation struct {
	Key    string
	Desc   string
	Action bindings.Action
	IsLeaf bool
}

// ResolveResult is the outcome of resolving a key press.
type ResolveResult struct {
	Action        bindings.Action
	Scope         bindings.ScopeName
	Args          map[string]any
	Pending       bool
	Consumed      bool
	Continuations []Continuation
}

type candidate struct {
	scope   bindings.ScopeName
	binding bindings.Binding
}

// Dispatcher resolves key presses against active scopes and bindings.
type Dispatcher struct {
	bindings        map[bindings.ScopeName][]bindings.Binding
	conditions      map[string]actionmeta.Condition
	builtInActions  map[bindings.Action]actionmeta.Condition
	actionOverrides map[bindings.Action]actionmeta.Condition
	state           common.StateProvider

	buffered   []tea.Key
	candidates []candidate
}

func NewDispatcher(availableBindings []bindings.Binding) (*Dispatcher, error) {
	if err := bindings.ValidateBindings(availableBindings); err != nil {
		return nil, err
	}

	d := &Dispatcher{
		bindings:        make(map[bindings.ScopeName][]bindings.Binding),
		conditions:      make(map[string]actionmeta.Condition),
		builtInActions:  make(map[bindings.Action]actionmeta.Condition),
		actionOverrides: make(map[bindings.Action]actionmeta.Condition),
	}
	for _, action := range actionmeta.BuiltInActions() {
		when := actionmeta.ActionWhen(action)
		if when == "" {
			continue
		}
		condition, _ := actionmeta.ParseCondition(when)
		d.builtInActions[bindings.Action(action)] = condition
	}
	for _, binding := range availableBindings {
		d.conditions[binding.When], _ = actionmeta.ParseCondition(binding.When)
		d.bindings[binding.Scope] = append(d.bindings[binding.Scope], binding)
	}
	return d, nil
}

func (d *Dispatcher) SetStateProvider(state common.StateProvider) { d.state = state }

// SetActionWhen sets the state condition that controls an action's availability.
func (d *Dispatcher) SetActionWhen(action bindings.Action, when string) {
	if d == nil {
		return
	}
	condition, err := actionmeta.ParseCondition(when)
	if err != nil {
		delete(d.actionOverrides, action)
		return
	}
	if strings.TrimSpace(when) == "" {
		delete(d.actionOverrides, action)
		return
	}
	d.actionOverrides[action] = condition
}

// ActionEnabled reports whether the action's declared state condition matches.
func (d *Dispatcher) ActionEnabled(action bindings.Action) bool {
	if d == nil {
		return true
	}
	if condition, ok := d.actionOverrides[action]; ok {
		return condition.Matches(d.state)
	}
	return d.BuiltInActionEnabled(action)
}

// BuiltInActionEnabled tests only generated built-in availability, ignoring
// conditions attached to configured Lua overrides.
func (d *Dispatcher) BuiltInActionEnabled(action bindings.Action) bool {
	if d == nil {
		return true
	}
	condition, ok := d.builtInActions[action]
	return !ok || condition.Matches(d.state)
}

func (d *Dispatcher) enabled(binding bindings.Binding) bool {
	return d.conditions[binding.When].Matches(d.state) && d.ActionEnabled(binding.Action)
}

// Continuations refreshes pending candidates against live state and visible scopes.
// An empty sequence remains pending until the next key is swallowed or cancelled.
func (d *Dispatcher) Continuations(scopes []common.Scope) []Continuation {
	return d.pendingContinuations(d.eligibleCandidates(scopes))
}

func (d *Dispatcher) eligibleCandidates(scopes []common.Scope) []candidate {
	visible := common.VisibleScopes(scopes)
	return slices.DeleteFunc(slices.Clone(d.candidates), func(c candidate) bool {
		return !d.enabled(c.binding) || !slices.ContainsFunc(visible, func(s common.Scope) bool { return s.Name == c.scope })
	})
}

func (d *Dispatcher) ResetSequence() {
	d.buffered = nil
	d.candidates = nil
}

// Resolve applies dispatch rules for a key in the provided layer chain.
// Scopes must be ordered from innermost to outermost.
func (d *Dispatcher) Resolve(msg tea.KeyMsg, scopes []common.Scope) ResolveResult {
	if msg.String() == "" {
		return ResolveResult{}
	}
	key := msg.Key()

	if len(d.buffered) > 0 {
		return d.resolveSequenceKey(key, d.eligibleCandidates(scopes))
	}

	seqCandidates := d.initialSequenceCandidates(key, scopes)
	if len(seqCandidates) > 0 {
		d.buffered = []tea.Key{key}
		d.candidates = seqCandidates
		return ResolveResult{
			Pending:       true,
			Consumed:      true,
			Continuations: d.pendingContinuations(d.candidates),
		}
	}

	for _, scope := range common.VisibleScopes(scopes) {
		scopeBindings := d.bindings[scope.Name]
		for _, binding := range slices.Backward(scopeBindings) {

			if len(binding.Key) == 0 || !d.enabled(binding) {
				continue
			}
			for _, candidateKey := range binding.Key {
				if keyMatches(candidateKey, key) {
					return ResolveResult{Action: binding.Action, Scope: scope.Name, Args: bindings.CloneArgs(binding.Args), Consumed: true}
				}
			}
		}
	}

	return ResolveResult{}
}

func (d *Dispatcher) resolveSequenceKey(key tea.Key, candidates []candidate) ResolveResult {
	if keyMatches("esc", key) {
		d.ResetSequence()
		return ResolveResult{Consumed: true}
	}

	nextBuffer := append(append([]tea.Key(nil), d.buffered...), key)
	filtered := make([]candidate, 0, len(d.candidates))
	for _, c := range candidates {
		if isPrefix(c.binding.Seq, nextBuffer) {
			filtered = append(filtered, c)
		}
	}

	if len(filtered) == 0 {
		d.ResetSequence()
		// Swallow the key when a sequence was in progress and no continuation matched.
		return ResolveResult{Consumed: true}
	}

	d.buffered = nextBuffer
	d.candidates = filtered

	// Inner scope wins; within the same scope, last-added binding wins.
	var matchScope bindings.ScopeName
	var matchAction bindings.Action
	var matchArgs map[string]any
	found := false
	for _, c := range filtered {
		if len(c.binding.Seq) != len(d.buffered) {
			continue
		}
		if !found {
			found = true
			matchScope = c.scope
			matchAction = c.binding.Action
			matchArgs = bindings.CloneArgs(c.binding.Args)
		} else if c.scope == matchScope {
			matchAction = c.binding.Action
			matchArgs = bindings.CloneArgs(c.binding.Args)
		}
	}
	if found {
		d.ResetSequence()
		return ResolveResult{Action: matchAction, Scope: matchScope, Args: matchArgs, Consumed: true}
	}

	return ResolveResult{
		Pending:       true,
		Consumed:      true,
		Continuations: d.pendingContinuations(d.candidates),
	}
}

func (d *Dispatcher) initialSequenceCandidates(key tea.Key, scopes []common.Scope) []candidate {
	var candidates []candidate
	for _, scope := range common.VisibleScopes(scopes) {
		for _, binding := range d.bindings[scope.Name] {
			if d.enabled(binding) && len(binding.Seq) > 0 && keyMatches(binding.Seq[0], key) {
				candidates = append(candidates, candidate{scope: scope.Name, binding: binding})
			}
		}
	}
	return candidates
}

func (d *Dispatcher) pendingContinuations(candidates []candidate) []Continuation {
	type entry struct {
		cont  Continuation
		descs []string
	}
	order := make([]string, 0, len(d.candidates))
	byKey := make(map[string]*entry, len(d.candidates))
	for _, c := range candidates {
		idx := len(d.buffered)
		if idx >= len(c.binding.Seq) {
			continue
		}

		next := c.binding.Seq[idx]
		isLeaf := idx == len(c.binding.Seq)-1
		desc := c.binding.Desc
		if desc == "" {
			desc = string(c.binding.Action)
		}

		if e, ok := byKey[next]; ok {
			e.descs = append(e.descs, desc)
			e.cont.IsLeaf = e.cont.IsLeaf && isLeaf
		} else {
			order = append(order, next)
			byKey[next] = &entry{
				cont: Continuation{
					Key:    next,
					Action: c.binding.Action,
					IsLeaf: isLeaf,
				},
				descs: []string{desc},
			}
		}
	}

	continuations := make([]Continuation, 0, len(order))
	for _, key := range order {
		e := byKey[key]
		e.cont.Desc = strings.Join(e.descs, ", ")
		continuations = append(continuations, e.cont)
	}
	return continuations
}

func isPrefix(full []string, prefix []tea.Key) bool {
	if len(prefix) > len(full) {
		return false
	}
	for i := range prefix {
		if !keyMatches(full[i], prefix[i]) {
			return false
		}
	}
	return true
}

func keyMatches(candidate string, key tea.Key) bool {
	return candidate == key.String() || candidate == key.Keystroke()
}
