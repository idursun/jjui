package config

import "github.com/BurntSushi/toml"

// DeprecatedConfigWarnings reports warnings for legacy config sections.
func DeprecatedConfigWarnings(content string) []string {
	type marker struct{}
	var decoded marker
	md, err := toml.Decode(content, &decoded)
	if err != nil {
		return nil
	}

	var warnings []string
	if md.IsDefined("custom_commands") {
		warnings = append(warnings, "[custom_commands] is no longer supported; define [[actions]] and [[bindings]] instead")
	}
	if md.IsDefined("leader") {
		warnings = append(warnings, "[leader] is no longer supported; use sequence bindings via [[bindings]].seq")
	}
	return warnings
}
