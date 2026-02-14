package config

import (
	"slices"
	"strings"
)

type bindingInputMode int

const (
	bindingInputNone bindingInputMode = iota
	bindingInputKey
	bindingInputSeq
	bindingInputInvalid
)

func mergeActions(base []ActionConfig, overlay []ActionConfig) []ActionConfig {
	if len(base) == 0 {
		return append([]ActionConfig(nil), overlay...)
	}
	if len(overlay) == 0 {
		return append([]ActionConfig(nil), base...)
	}

	overlayByName := make(map[string]ActionConfig, len(overlay))
	for _, action := range overlay {
		name := strings.TrimSpace(action.Name)
		if name == "" {
			continue
		}
		overlayByName[name] = action
	}

	merged := make([]ActionConfig, 0, len(base)+len(overlay))
	for _, action := range base {
		name := strings.TrimSpace(action.Name)
		if replacement, ok := overlayByName[name]; ok {
			merged = append(merged, replacement)
			delete(overlayByName, name)
			continue
		}
		merged = append(merged, action)
	}

	for _, action := range overlay {
		name := strings.TrimSpace(action.Name)
		if _, ok := overlayByName[name]; ok {
			merged = append(merged, action)
			delete(overlayByName, name)
		}
	}
	return merged
}

func mergeBindings(base []BindingConfig, overlay []BindingConfig) []BindingConfig {
	merged := append([]BindingConfig(nil), base...)
	for _, userBinding := range overlay {
		merged = removeShadowedBindings(merged, userBinding)
		merged = append(merged, userBinding)
	}
	return merged
}

func removeShadowedBindings(existing []BindingConfig, user BindingConfig) []BindingConfig {
	scope := strings.TrimSpace(user.Scope)
	if scope == "" {
		return existing
	}
	mode := bindingInputModeOf(user)
	if mode == bindingInputNone || mode == bindingInputInvalid {
		return existing
	}

	userKeys := make(map[string]struct{}, len(user.Key))
	for _, key := range user.Key {
		userKeys[key] = struct{}{}
	}

	filtered := make([]BindingConfig, 0, len(existing))
	for _, binding := range existing {
		if strings.TrimSpace(binding.Scope) != scope {
			filtered = append(filtered, binding)
			continue
		}

		// seq bindings shadow exact sequence matches in the same scope.
		if mode == bindingInputSeq {
			if len(binding.Seq) > 0 && slices.Equal(binding.Seq, user.Seq) {
				continue
			}
			filtered = append(filtered, binding)
			continue
		}

		// key bindings shadow only overlapping keys in the same scope.
		if len(user.Key) > 0 && len(binding.Key) > 0 {
			kept := make([]string, 0, len(binding.Key))
			for _, key := range binding.Key {
				if _, shadowed := userKeys[key]; shadowed {
					continue
				}
				kept = append(kept, key)
			}
			if len(kept) == 0 {
				continue
			}
			binding.Key = kept
		}

		filtered = append(filtered, binding)
	}
	return filtered
}

func bindingInputModeOf(binding BindingConfig) bindingInputMode {
	hasKey := len(binding.Key) > 0
	hasSeq := len(binding.Seq) > 0
	switch {
	case hasKey && hasSeq:
		return bindingInputInvalid
	case hasKey:
		return bindingInputKey
	case hasSeq:
		return bindingInputSeq
	default:
		return bindingInputNone
	}
}
