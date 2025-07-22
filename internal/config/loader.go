package config

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

func getConfigFilePath() string {
	var configDirs []string

	// useful during development or other non-standard setups.
	if dir := os.Getenv("JJUI_CONFIG_DIR"); dir != "" {
		if s, err := os.Stat(dir); err == nil && s.IsDir() {
			configDirs = append(configDirs, dir)
		}
	}

	// os.UserConfigDir() already does this for linux leaving darwin to handle
	if runtime.GOOS == "darwin" {
		configDirs = append(configDirs, path.Join(os.Getenv("HOME"), ".config"))
		xdgConfigDir := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigDir != "" {
			configDirs = append(configDirs, xdgConfigDir)
		}
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	configDirs = append(configDirs, configDir)

	for _, dir := range configDirs {
		configPath := filepath.Join(dir, "jjui", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	if len(configDirs) > 0 {
		return filepath.Join(configDirs[0], "jjui", "config.toml")
	}
	return ""
}

func (c *Config) Load(data string) error {
	var err error

	_, err = toml.Decode(data, c)
	if err != nil {
		return err
	}

	return nil
}

func LoadConfigFile() ([]byte, error) {
	configFile := getConfigFilePath()
	_, err := os.Stat(configFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func LoadTheme(name string) (map[string]Color, error) {
	configFilePath := getConfigFilePath()
	themeFile := filepath.Join(filepath.Dir(configFilePath), "themes", name+".toml")
	_, err := os.Stat(themeFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(themeFile)
	if err != nil {
		return nil, err
	}
	colors := make(map[string]Color)
	err = toml.Unmarshal(data, &colors)
	if err != nil {
		return nil, err
	}
	return colors, nil
}
