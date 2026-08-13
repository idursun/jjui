package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type RevsetAliasMap map[string]string

type JJConfig struct {
	Colors        map[string]Color `toml:"colors"`
	RevsetAliases RevsetAliasMap   `toml:"revset-aliases"`
	Revsets       struct {
		Log string `toml:"log"`
	} `toml:"revsets"`
	Templates struct {
		Log string `toml:"log"`
	} `toml:"templates"`
}

func (c *JJConfig) GetApplicableColors() map[string]Color {
	ret := make(map[string]Color)
	if c == nil || c.Colors == nil {
		return ret
	}
	applicableColorKeys := []string{
		"diff added",
		"diff renamed",
		"diff copied",
		"diff modified",
		"diff removed",
		"bookmark",
		"change_id",
		"commit_id",
		"conflict",
	}
	for _, key := range applicableColorKeys {
		if color, ok := c.Colors[key]; ok {
			ret[key] = color
		}
	}
	return ret
}

func parseConfig(configContent string) (*JJConfig, error) {
	var config JJConfig
	_, err := toml.Decode(configContent, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func DefaultConfig(output []byte) (*JJConfig, error) {
	return parseConfig(string(output))
}

func (m *RevsetAliasMap) UnmarshalTOML(data any) error {
	rawMap, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("expected a table for revset-aliases, got %T", data)
	}

	*m = make(RevsetAliasMap)

	for key, value := range rawMap {
		switch v := value.(type) {
		case string:
			(*m)[key] = v
		case map[string]any:
			if def, ok := v["definition"].(string); ok {
				(*m)[key] = def
			} else {
				return fmt.Errorf("missing or invalid 'definition' field in revset-alias object %q, %T", key, value)
			}
		default:
			return fmt.Errorf("expected string or object for revset-alias %q, got %T", key, value)
		}
	}
	return nil
}
