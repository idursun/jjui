package helpkeys

import (
	"sort"
	"strings"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actionmeta"
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
	configuredActions map[keybindings.Action]config.ActionConfig,
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

			seenActions[dedupeKey] = struct{}{}
			entries = append(entries, Entry{
				Label: label,
				Desc:  BindingDescription(action, configuredActions[action]),
			})
		}
	}

	return entries
}

// BuildFromContinuations returns sequence continuation entries, sorted for stable display.
func BuildFromContinuations(
	continuations []dispatch.Continuation,
	configuredActions map[keybindings.Action]config.ActionConfig,
) []Entry {
	if len(continuations) == 0 {
		return nil
	}
	entries := make([]Entry, 0, len(continuations))
	for _, continuation := range continuations {
		desc := BindingDescription(continuation.Action, configuredActions[continuation.Action])
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

func BindingDescription(action keybindings.Action, cfg config.ActionConfig) string {
	if desc := strings.TrimSpace(cfg.Desc); desc != "" {
		return desc
	}
	return actionmeta.ActionDescription(string(action))
}

func actionLeaf(action keybindings.Action) keybindings.Action {
	name := strings.TrimSpace(string(action))
	if name == "" {
		return action
	}
	if token := actionmeta.ActionToken(name); token != "" {
		return keybindings.Action(token)
	}
	return keybindings.Action(name)
}
