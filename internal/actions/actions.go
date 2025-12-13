package actions

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

// Action represents a callable action loaded from config.
type Action struct {
	Name string
	Lua  string `toml:"lua"`
}

// Load parses [actions."..."] blocks from the provided TOML content.
// Later loads can overwrite earlier ones by name.
func Load(data string) (map[string]Action, error) {
	type actionsTOML struct {
		Raw map[string]toml.Primitive `toml:"actions"`
	}

	var decoded actionsTOML
	md, err := toml.Decode(data, &decoded)
	if err != nil {
		return nil, err
	}

	registry := make(map[string]Action)
	for name, prim := range decoded.Raw {
		var action Action
		if err := md.PrimitiveDecode(prim, &action); err != nil {
			return nil, fmt.Errorf("failed to decode action %s: %w", name, err)
		}
		action.Name = name
		if action.Lua == "" {
			return nil, fmt.Errorf("action %s: lua cannot be empty", name)
		}
		registry[name] = action
	}
	return registry, nil
}
