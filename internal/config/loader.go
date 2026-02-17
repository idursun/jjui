package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type mergeOverlay struct {
	Actions  []ActionConfig  `toml:"actions"`
	Bindings []BindingConfig `toml:"bindings"`
}

func getConfigFilePath() string {
	var configDirs []string

	// useful during development or other non-standard setups.
	if dir := os.Getenv("JJUI_CONFIG_DIR"); dir != "" {
		if s, err := os.Stat(dir); err == nil && s.IsDir() {
			return filepath.Join(dir, "config.toml")
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

	if configDir, err := os.UserConfigDir(); err == nil {
		configDirs = append(configDirs, configDir)
	}

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

func GetConfigDir() string {
	configFile := getConfigFilePath()
	if configFile == "" {
		return ""
	}
	return filepath.Dir(configFile)
}

func loadDefaultConfig() *Config {
	data, err := configFS.ReadFile("default/config.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: no embedded default config found: %v\n", err)
		os.Exit(1)
	}

	config := &Config{}
	if err := config.Load(string(data)); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: failed to load embedded default config: %v\n", err)
		os.Exit(1)
	}

	bindingsData, err := configFS.ReadFile("default/bindings.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: no embedded default bindings found: %v\n", err)
		os.Exit(1)
	}
	if err := config.Load(string(bindingsData)); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: failed to load embedded default bindings: %v\n", err)
		os.Exit(1)
	}

	return config
}

func (c *Config) Load(data string) error {
	baseActions := append([]ActionConfig(nil), c.Actions...)
	baseBindings := append([]BindingConfig(nil), c.Bindings...)

	metadata, err := toml.Decode(data, c)
	if err != nil {
		return err
	}

	// Decode only merge-managed array/table fields into a fresh struct so these
	// collections are always read from file content, without carrying prior state.
	// Keep this type explicit and extend it when new merge-managed arrays are added.
	overlay := &mergeOverlay{}
	if _, err := toml.Decode(data, overlay); err != nil {
		return err
	}

	// If a custom bindings profile is specified, use it as the base instead of built-in defaults
	if metadata.IsDefined("bindings_profile") && c.BindingsProfile != "" && c.BindingsProfile != ":builtin" {
		profileBindings, err := loadProfileBindings(c.BindingsProfile)
		if err != nil {
			return err
		}
		baseBindings = profileBindings
		if !metadata.IsDefined("bindings") {
			c.Bindings = profileBindings
		}
	}

	if metadata.IsDefined("actions") {
		c.Actions = MergeActions(baseActions, overlay.Actions)
	}
	if metadata.IsDefined("bindings") {
		c.Bindings = MergeBindings(baseBindings, overlay.Bindings)
	}

	return c.ValidateBindingsAndActions()
}

func loadProfileBindings(profile string) ([]BindingConfig, error) {
	configFilePath := getConfigFilePath()
	profilePath := profile
	if !filepath.IsAbs(profilePath) {
		profilePath = filepath.Join(filepath.Dir(configFilePath), profilePath)
	}
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("loading bindings profile %q: %w", profile, err)
	}
	var overlay mergeOverlay
	if _, err := toml.Decode(string(data), &overlay); err != nil {
		return nil, fmt.Errorf("parsing bindings profile %q: %w", profile, err)
	}
	return overlay.Bindings, nil
}

func LoadLuaConfigFile() (string, error) {
	configDir := GetConfigDir()
	if configDir == "" {
		return "", nil
	}
	luaFile := filepath.Join(configDir, "config.lua")
	data, err := os.ReadFile(luaFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
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

func loadTheme(data []byte, base map[string]Color) (map[string]Color, error) {
	colors := make(map[string]Color)
	for key, color := range base {
		colors[key] = color
	}
	err := toml.Unmarshal(data, &colors)
	if err != nil {
		return nil, err
	}
	return colors, nil
}

func LoadEmbeddedTheme(name string) (map[string]Color, error) {
	embeddedPath := "default/" + name + ".toml"
	data, err := configFS.ReadFile(embeddedPath)
	if err != nil {
		return nil, err
	}
	return loadTheme(data, nil)
}

func LoadTheme(name string, base map[string]Color) (map[string]Color, error) {
	configFilePath := getConfigFilePath()
	themeFile := filepath.Join(filepath.Dir(configFilePath), "themes", name+".toml")

	data, err := os.ReadFile(themeFile)
	if err != nil {
		return nil, err
	}
	return loadTheme(data, base)
}
