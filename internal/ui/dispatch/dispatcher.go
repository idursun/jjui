package dispatch

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	Scope         bindings.Scope
	Args          map[string]any
	Pending       bool
	Consumed      bool
	Continuations []Continuation
}

type candidate struct {
	scope   bindings.Scope
	binding bindings.Binding
}

// Dispatcher resolves key presses against active scopes and bindings.
type Dispatcher struct {
	bindings map[bindings.Scope][]bindings.Binding

	buffered   []string
	candidates []candidate
}

func NewDispatcher(availableBindings []bindings.Binding) (*Dispatcher, error) {
	if err := bindings.ValidateBindings(availableBindings); err != nil {
		return nil, err
	}

	d := &Dispatcher{bindings: make(map[bindings.Scope][]bindings.Binding)}
	for _, binding := range availableBindings {
		d.bindings[binding.Scope] = append(d.bindings[binding.Scope], binding)
	}
	return d, nil
}

func (d *Dispatcher) ResetSequence() {
	d.buffered = nil
	d.candidates = nil
}

// Resolve applies dispatch rules for a key in the provided scope chain.
// Scopes must be ordered from innermost to outermost.
func (d *Dispatcher) Resolve(msg tea.KeyMsg, scopes []bindings.Scope) ResolveResult {
	key := msg.String()
	if key == "" {
		return ResolveResult{}
	}

	if len(d.candidates) > 0 {
		return d.resolveSequenceKey(key)
	}

	seqCandidates := d.initialSequenceCandidates(key, scopes)
	if len(seqCandidates) > 0 {
		d.buffered = []string{key}
		d.candidates = seqCandidates
		return ResolveResult{
			Pending:       true,
			Consumed:      true,
			Continuations: d.pendingContinuations(),
		}
	}

	for _, scope := range scopes {
		scopeBindings := d.bindings[scope]
		for i := len(scopeBindings) - 1; i >= 0; i-- {
			binding := scopeBindings[i]
			if len(binding.Key) == 0 {
				continue
			}
			for _, candidateKey := range binding.Key {
				if keysEqual(candidateKey, key) {
					return ResolveResult{Action: binding.Action, Scope: scope, Args: bindings.CloneArgs(binding.Args), Consumed: true}
				}
			}
		}
	}

	return ResolveResult{}
}

func (d *Dispatcher) resolveSequenceKey(key string) ResolveResult {
	if key == "esc" {
		d.ResetSequence()
		return ResolveResult{Consumed: true}
	}

	nextBuffer := append(append([]string(nil), d.buffered...), key)
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
	var matchScope bindings.Scope
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

func (d *Dispatcher) initialSequenceCandidates(key string, scopes []bindings.Scope) []candidate {
	var candidates []candidate
	for _, scope := range scopes {
		for _, binding := range d.bindings[scope] {
			if len(binding.Seq) > 0 && keysEqual(binding.Seq[0], key) {
				candidates = append(candidates, candidate{scope: scope, binding: binding})
			}
		}
	}
	return candidates
}

func (d *Dispatcher) pendingContinuations() []Continuation {
	seen := map[string]struct{}{}
	continuations := make([]Continuation, 0, len(d.candidates))
	for _, c := range d.candidates {
		idx := len(d.buffered)
		if idx >= len(c.binding.Seq) {
			continue
		}

		next := c.binding.Seq[idx]
		if _, ok := seen[next]; ok {
			continue
		}
		seen[next] = struct{}{}

		continuations = append(continuations, Continuation{
			Key:    next,
			Desc:   c.binding.Desc,
			Action: c.binding.Action,
			IsLeaf: idx == len(c.binding.Seq)-1,
		})
	}
	return continuations
}

func isPrefix(full []string, prefix []string) bool {
	if len(prefix) > len(full) {
		return false
	}
	for i := range prefix {
		if !keysEqual(full[i], prefix[i]) {
			return false
		}
	}
	return true
}

func normalizeKeyName(key string) string {
	if strings.EqualFold(key, "space") {
		return " "
	}
	return key
}

func keysEqual(a string, b string) bool {
	return normalizeKeyName(a) == normalizeKeyName(b)
}
