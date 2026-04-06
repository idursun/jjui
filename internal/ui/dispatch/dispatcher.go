package dispatch

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/bindings"
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
	bindings map[bindings.ScopeName][]bindings.Binding

	buffered   []tea.Key
	candidates []candidate
}

func NewDispatcher(availableBindings []bindings.Binding) (*Dispatcher, error) {
	if err := bindings.ValidateBindings(availableBindings); err != nil {
		return nil, err
	}

	d := &Dispatcher{bindings: make(map[bindings.ScopeName][]bindings.Binding)}
	for _, binding := range availableBindings {
		d.bindings[binding.Scope] = append(d.bindings[binding.Scope], binding)
	}
	return d, nil
}

func (d *Dispatcher) ResetSequence() {
	d.buffered = nil
	d.candidates = nil
}

// Resolve applies dispatch rules for a key in the provided layer chain.
// Scopes must be ordered from innermost to outermost.
func (d *Dispatcher) Resolve(msg tea.KeyMsg, scopes []Scope) ResolveResult {
	if msg.String() == "" {
		return ResolveResult{}
	}
	key := msg.Key()

	if len(d.candidates) > 0 {
		return d.resolveSequenceKey(key)
	}

	seqCandidates := d.initialSequenceCandidates(key, scopes)
	if len(seqCandidates) > 0 {
		d.buffered = []tea.Key{key}
		d.candidates = seqCandidates
		return ResolveResult{
			Pending:       true,
			Consumed:      true,
			Continuations: d.pendingContinuations(),
		}
	}

	for _, scope := range VisibleScopes(scopes) {
		scopeBindings := d.bindings[scope.Name]
		for i := len(scopeBindings) - 1; i >= 0; i-- {
			binding := scopeBindings[i]
			if len(binding.Key) == 0 {
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

func (d *Dispatcher) resolveSequenceKey(key tea.Key) ResolveResult {
	if keyMatches("esc", key) {
		d.ResetSequence()
		return ResolveResult{Consumed: true}
	}

	nextBuffer := append(append([]tea.Key(nil), d.buffered...), key)
	filtered := make([]candidate, 0, len(d.candidates))
	for _, c := range d.candidates {
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
		Continuations: d.pendingContinuations(),
	}
}

func (d *Dispatcher) initialSequenceCandidates(key tea.Key, scopes []Scope) []candidate {
	var candidates []candidate
	for _, scope := range VisibleScopes(scopes) {
		for _, binding := range d.bindings[scope.Name] {
			if len(binding.Seq) > 0 && keyMatches(binding.Seq[0], key) {
				candidates = append(candidates, candidate{scope: scope.Name, binding: binding})
			}
		}
	}
	return candidates
}

func (d *Dispatcher) pendingContinuations() []Continuation {
	type entry struct {
		cont  Continuation
		descs []string
	}
	order := make([]string, 0, len(d.candidates))
	byKey := make(map[string]*entry, len(d.candidates))
	for _, c := range d.candidates {
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
