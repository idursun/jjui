package help

import (
	"sort"
	"strings"

	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
)

// Entry is a status-help key entry rendered as "key description".
type Entry struct {
	Label      string
	Desc       string
	Overridden bool
}

// ScopeGroup is a named group of help entries for a single scope.
type ScopeGroup struct {
	Name    string
	Entries []Entry
}

// FlatEntries returns all entries from a slice of groups in order.
func FlatEntries(groups []ScopeGroup) []Entry {
	var entries []Entry
	for _, g := range groups {
		entries = append(entries, g.Entries...)
	}
	return entries
}

// BuildFromBindings returns short-help entries for the given scope.
func BuildFromBindings(
	scope keybindings.ScopeName,
	bindings []config.BindingConfig,
) []Entry {
	entries := make([]Entry, 0)
	seenActions := map[keybindings.Action]struct{}{}

	for i := len(bindings) - 1; i >= 0; i-- {
		b := bindings[i]
		if keybindings.ScopeName(strings.TrimSpace(b.Scope)) != scope {
			continue
		}
		action := keybindings.Action(strings.TrimSpace(b.Action))
		if action == "" {
			continue
		}
		if _, seen := seenActions[action]; seen {
			continue
		}

		label := BindingLabel(b)
		if label == "" {
			continue
		}

		entries = append(entries, Entry{
			Label: label,
			Desc:  bindingDesc(b),
		})
		seenActions[action] = struct{}{}
	}

	// Reverse to restore original order
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

// BuildGroupedFromBindings returns help entries grouped by scope.
func BuildGroupedFromBindings(
	scopes []keybindings.ScopeName,
	bindings []config.BindingConfig,
) []ScopeGroup {
	var groups []ScopeGroup
	for _, scope := range scopes {
		entries := BuildFromBindings(scope, bindings)
		if len(entries) > 0 {
			groups = append(groups, ScopeGroup{
				Name:    scopeDisplayName(string(scope)),
				Entries: entries,
			})
		}
	}
	return groups
}

// MarkOverriddenKeys marks entries in outer groups as overridden when an inner
// group binds the same key label.
func MarkOverriddenKeys(groups []ScopeGroup) {
	seenKeys := make(map[string]struct{})
	for i := range groups {
		for j := range groups[i].Entries {
			label := groups[i].Entries[j].Label
			if _, seen := seenKeys[label]; seen {
				groups[i].Entries[j].Overridden = true
			}
		}
		// After processing the group, add all its keys to seenKeys
		for _, e := range groups[i].Entries {
			if !e.Overridden {
				seenKeys[e.Label] = struct{}{}
			}
		}
	}
}

// BuildFromContinuations returns sequence continuation entries, sorted for stable display.
func BuildFromContinuations(continuations []dispatch.Continuation) []Entry {
	if len(continuations) == 0 {
		return nil
	}
	entries := make([]Entry, 0, len(continuations))
	for _, continuation := range continuations {
		desc := continuationDesc(continuation)
		if !continuation.IsLeaf {
			desc += " ..."
		}
		entries = append(entries, Entry{
			Label: NormalizeDisplayKey(continuation.Key),
			Desc:  desc,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Label != entries[j].Label {
			return entries[i].Label < entries[j].Label
		}
		return entries[i].Desc < entries[j].Desc
	})
	return entries
}

func BindingLabel(binding config.BindingConfig) string {
	if len(binding.Key) > 0 {
		keys := make([]string, 0, len(binding.Key))
		for _, k := range binding.Key {
			keys = append(keys, NormalizeDisplayKey(k))
		}
		return strings.Join(keys, "/")
	}
	if len(binding.Seq) > 0 {
		keys := make([]string, 0, len(binding.Seq))
		for _, k := range binding.Seq {
			keys = append(keys, NormalizeDisplayKey(k))
		}
		return strings.Join(keys, " ")
	}
	return ""
}

func NormalizeDisplayKey(key string) string {
	key = strings.TrimSpace(key)
	switch strings.ToLower(key) {
	case " ":
		return "space"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	}
	return key
}

func bindingDesc(b config.BindingConfig) string {
	if desc := strings.TrimSpace(b.Desc); desc != "" {
		return desc
	}
	return descFromAction(string(keybindings.Action(strings.TrimSpace(b.Action))))
}

func continuationDesc(c dispatch.Continuation) string {
	if desc := strings.TrimSpace(c.Desc); desc != "" {
		return desc
	}
	return descFromAction(string(c.Action))
}

// descFromAction derives a human-readable description from the action token
// (last segment after '.'), replacing underscores with spaces.
func descFromAction(action string) string {
	token := actionToken(action)
	if token == "" {
		return ""
	}
	return strings.ReplaceAll(token, "_", " ")
}

// actionToken extracts the last segment after '.' from a canonical action ID.
func actionToken(action string) string {
	if idx := strings.LastIndexByte(action, '.'); idx >= 0 && idx < len(action)-1 {
		return action[idx+1:]
	}
	return action
}
