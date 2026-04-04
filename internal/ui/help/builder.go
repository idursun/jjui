package help

import (
	"sort"
	"strings"

	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/routing"
)

// Entry is a status-help key entry rendered as "key description".
type Entry struct {
	Label string
	Desc  string
}

// BuildFromBindings returns short-help entries for the provided scope chain.
// Scopes are expected from innermost to outermost.
func BuildFromBindings(
	scopes []keybindings.Scope,
	bindings []config.BindingConfig,
) []Entry {
	bindingsByScope := make(map[keybindings.Scope][]config.BindingConfig)
	for _, binding := range bindings {
		scope := keybindings.Scope(strings.TrimSpace(binding.Scope))
		bindingsByScope[scope] = append(bindingsByScope[scope], binding)
	}

	seenLeaves := map[keybindings.Action]struct{}{}
	entries := make([]Entry, 0)

	for _, scope := range scopes {
		scopeBindings := bindingsByScope[scope]
		scopeEntries := make([]Entry, 0, len(scopeBindings))
		seenInScope := map[keybindings.Action]struct{}{}
		for i := len(scopeBindings) - 1; i >= 0; i-- {
			b := scopeBindings[i]
			action := keybindings.Action(strings.TrimSpace(b.Action))
			if action == "" {
				continue
			}
			leaf := actionLeaf(action)
			if _, seen := seenInScope[action]; seen {
				continue
			}
			if _, seen := seenLeaves[leaf]; seen {
				continue
			}

			label := BindingLabel(b)
			if label == "" {
				continue
			}

			scopeEntries = append(scopeEntries, Entry{
				Label: label,
				Desc:  bindingDesc(b),
			})
			seenInScope[action] = struct{}{}
		}
		// Mark leaves as seen after processing the entire scope,
		// so that different actions with the same leaf within one scope
		// are all shown, while outer scopes are still shadowed.
		for i := len(scopeEntries) - 1; i >= 0; i-- {
			entries = append(entries, scopeEntries[i])
		}
		for action := range seenInScope {
			seenLeaves[actionLeaf(action)] = struct{}{}
		}
	}

	return entries
}

// BuildFromLayers returns short-help entries for the visible layer chain.
func BuildFromLayers(layers []routing.Layer, bindings []config.BindingConfig) []Entry {
	var scopes []keybindings.Scope
	for _, layer := range layers {
		if layer.Scope != "" {
			scopes = append(scopes, layer.Scope)
		}
		if !layer.AllowLeak {
			break
		}
	}
	return BuildFromBindings(scopes, bindings)
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

func actionLeaf(action keybindings.Action) keybindings.Action {
	name := strings.TrimSpace(string(action))
	if name == "" {
		return action
	}
	return keybindings.Action(actionToken(name))
}

// actionToken extracts the last segment after '.' from a canonical action ID.
func actionToken(action string) string {
	if idx := strings.LastIndexByte(action, '.'); idx >= 0 && idx < len(action)-1 {
		return action[idx+1:]
	}
	return action
}
