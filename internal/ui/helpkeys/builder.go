package helpkeys

import (
	"sort"
	"strings"

	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
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

	seenActions := map[keybindings.Action]struct{}{}
	entries := make([]Entry, 0)

	for _, scope := range scopes {
		scopeBindings := bindingsByScope[scope]
		for _, b := range scopeBindings {
			action := keybindings.Action(strings.TrimSpace(b.Action))
			if action == "" {
				continue
			}
			dedupeKey := actionLeaf(action)
			if _, seen := seenActions[dedupeKey]; seen {
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
		}
		for _, b := range scopeBindings {
			dedupeKey := actionLeaf(keybindings.Action(strings.TrimSpace(b.Action)))
			seenActions[dedupeKey] = struct{}{}
		}
	}

	return entries
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
