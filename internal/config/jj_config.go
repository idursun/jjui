package config

import (
	"github.com/BurntSushi/toml"
)

type JJConfig struct {
	Colors        map[string]Color  `toml:"colors"`
	RevsetAliases map[string]string `toml:"revset-aliases"`
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
