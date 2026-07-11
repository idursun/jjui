package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

type mergeOverlay struct {
	Actions  []ActionConfig  `toml:"actions"`
	Bindings []BindingConfig `toml:"bindings"`
}

func EnvConfigDir() string {
	if dir := os.Getenv("JJUI_CONFIG_DIR"); dir != "" {
		if s, err := os.Stat(dir); err == nil && s.IsDir() {
			return dir
		}
	}
	return ""
}

// getConfigFilePath returns the effective global config file path.
// When JJUI_CONFIG_DIR is set and valid, it takes precedence over standard dirs.
func getConfigFilePath() string {
	var configDirs []string

	// useful during development or other non-standard setups.
	if dir := EnvConfigDir(); dir != "" {
		return filepath.Join(dir, "config.toml")
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
	if err := config.Load(string(data), ""); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: failed to load embedded default config: %v\n", err)
		os.Exit(1)
	}

	bindingsData, err := configFS.ReadFile("default/bindings.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: no embedded default bindings found: %v\n", err)
		os.Exit(1)
	}
	if err := config.Load(string(bindingsData), ""); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: failed to load embedded default bindings: %v\n", err)
		os.Exit(1)
	}

	return config
}

// Load loads config data where relative bindings_profile paths are resolved
// against baseDir.
func (c *Config) Load(data, baseDir string) error {
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
	overlayActions := make([]ActionConfig, 0, len(overlay.Actions))
	actionBindings := make([]BindingConfig, 0, len(overlay.Actions))
	for i, action := range overlay.Actions {
		overlayActions = append(overlayActions, ActionConfig{
			Name: action.Name,
			Lua:  action.Lua,
			Args: action.Args,
		})

		hasKey := len(action.Key) > 0
		hasSeq := len(action.Seq) > 0
		if !hasKey && !hasSeq {
			continue
		}
		if hasKey && hasSeq {
			return fmt.Errorf("actions[%d]: must set exactly one of key or seq", i)
		}
		if strings.TrimSpace(action.Scope) == "" {
			return fmt.Errorf("actions[%d]: scope is required when key or seq is set", i)
		}

		binding := BindingConfig{
			Action: action.Name,
			Desc:   action.Desc,
			Scope:  action.Scope,
		}
		if hasKey {
			binding.Key = action.Key
		}
		if hasSeq {
			binding.Seq = action.Seq
		}
		actionBindings = append(actionBindings, binding)
	}

	// If a custom bindings profile is specified, use it as the base instead of built-in defaults
	if metadata.IsDefined("bindings_profile") && c.BindingsProfile != "" && c.BindingsProfile != ":builtin" {
		profileBindings, err := loadProfileBindings(c.BindingsProfile, baseDir)
		if err != nil {
			return err
		}
		baseBindings = profileBindings
		if !metadata.IsDefined("bindings") {
			c.Bindings = profileBindings
		}
	}

	if metadata.IsDefined("actions") {
		c.Actions = append(baseActions, overlayActions...)
	}
	if metadata.IsDefined("actions") && len(actionBindings) > 0 && !metadata.IsDefined("bindings") {
		c.Bindings = append(baseBindings, actionBindings...)
	}
	if metadata.IsDefined("bindings") {
		overlayBindings := append(actionBindings, overlay.Bindings...)
		c.Bindings = append(baseBindings, overlayBindings...)
	}

	return c.ValidateBindingsAndActions()
}

func loadProfileBindings(profile, baseDir string) ([]BindingConfig, error) {
	profilePath := profile
	if !filepath.IsAbs(profilePath) {
		profilePath = filepath.Join(baseDir, profilePath)
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

func LoadRepoConfigFile(repoRoot string) ([]byte, error) {
	repoConfigPath := filepath.Join(repoRoot, ".jjui", "config.toml")
	data, err := os.ReadFile(repoConfigPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func LoadRepoLuaConfigFile(repoRoot string) (string, error) {
	luaFile := filepath.Join(repoRoot, ".jjui", "config.lua")
	data, err := os.ReadFile(luaFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func loadTheme(data []byte, base ResolvedTheme, isDark bool) (ResolvedTheme, error) {
	colors := NormalizeColorSelectors(base.Colors)
	if colors == nil {
		colors = make(map[string]Color)
	}
	resolved := ResolvedTheme{Colors: colors, BackgroundBlend: base.BackgroundBlend}

	structured, err := isStructuredTheme(data)
	if err != nil {
		return ResolvedTheme{}, err
	}
	if !structured {
		legacyColors := make(map[string]Color)
		if err := toml.Unmarshal(data, &legacyColors); err != nil {
			return ResolvedTheme{}, err
		}
		maps.Copy(resolved.Colors, NormalizeColorSelectors(legacyColors))
		return resolved, nil
	}

	var theme Theme
	if err := toml.Unmarshal(data, &theme); err != nil {
		return ResolvedTheme{}, err
	}
	if err := theme.Validate(); err != nil {
		return ResolvedTheme{}, err
	}

	maps.Copy(resolved.Colors, NormalizeColorSelectors(theme.Colors))
	variant := theme.Light
	if isDark {
		variant = theme.Dark
	}
	maps.Copy(resolved.Colors, NormalizeColorSelectors(variant.Colors))
	if variant.BackgroundBlend != nil {
		resolved.BackgroundBlend = *variant.BackgroundBlend
	}
	return resolved, nil
}

func isStructuredTheme(data []byte) (bool, error) {
	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return false, err
	}
	for _, key := range []string{"colors", "light", "dark"} {
		if table, ok := raw[key].(map[string]any); ok && !isColorValue(table) {
			return true, nil
		}
	}
	return false, nil
}

func isColorValue(table map[string]any) bool {
	for key := range table {
		switch key {
		case "fg", "bg", "bold", "italic", "underline", "strikethrough", "reverse":
		default:
			return false
		}
	}
	return true
}

func (t Theme) Validate() error {
	if err := t.Light.Validate("light"); err != nil {
		return err
	}
	return t.Dark.Validate("dark")
}

func (v ThemeVariant) Validate(name string) error {
	if v.BackgroundBlend == nil {
		return nil
	}
	value := *v.BackgroundBlend
	if math.IsNaN(value) || value < 0 || value > 1 {
		return fmt.Errorf("invalid value for '%s.background_blend': expected 0.0 to 1.0, got %v", name, value)
	}
	return nil
}

func LoadEmbeddedTheme(name string, base ResolvedTheme, isDark bool) (ResolvedTheme, error) {
	embeddedPath := "default/" + name + ".toml"
	data, err := configFS.ReadFile(embeddedPath)
	if err != nil {
		return ResolvedTheme{}, err
	}
	return loadTheme(data, base, isDark)
}

func LoadTheme(name string, base ResolvedTheme, isDark bool) (ResolvedTheme, error) {
	configFilePath := getConfigFilePath()
	themeFile := filepath.Join(filepath.Dir(configFilePath), "themes", name+".toml")

	data, err := os.ReadFile(themeFile)
	if err != nil {
		return ResolvedTheme{}, err
	}
	return loadTheme(data, base, isDark)
}

type LuaTypesInstallResult struct {
	TypesPath    string
	LuaRCPath    string
	LuaRCCreated bool
}

func SetupLuaTypes() (*LuaTypesInstallResult, error) {
	configDir := GetConfigDir()
	if configDir == "" {
		return nil, fmt.Errorf("could not determine config directory")
	}
	data, err := configFS.ReadFile("default/types.lua")
	if err != nil {
		return nil, fmt.Errorf("embedded types.lua not found: %w", err)
	}
	dest := filepath.Join(configDir, "types.lua")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return nil, err
	}

	luaRCPath := filepath.Join(configDir, ".luarc.json")
	created, err := ensureLuaRC(luaRCPath, dest)
	if err != nil {
		return nil, err
	}

	return &LuaTypesInstallResult{
		TypesPath:    dest,
		LuaRCPath:    luaRCPath,
		LuaRCCreated: created,
	}, nil
}

func ensureLuaRC(luaRCPath, typesPath string) (bool, error) {
	if _, err := os.Stat(luaRCPath); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}

	content, err := json.MarshalIndent(map[string]any{
		"workspace": map[string]any{
			"library": []string{typesPath},
		},
	}, "", "  ")
	if err != nil {
		return false, err
	}
	content = append(content, '\n')

	if err := os.WriteFile(luaRCPath, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// ResolveTheme loads the full color map for the given background mode.
// It layers the embedded default theme, optional user theme, jj VCS colors,
// and inline [ui.colors] overrides in the correct order.
func ResolveTheme(isDark bool, jjColors map[string]Color) (ResolvedTheme, error) {
	const defaultThemeName = "default"
	theme, err := LoadEmbeddedTheme(defaultThemeName, ResolvedTheme{}, isDark)
	if err != nil {
		return ResolvedTheme{}, fmt.Errorf("loading default theme %q: %w", defaultThemeName, err)
	}

	userThemeName := Current.UI.Theme.Light
	if isDark {
		userThemeName = Current.UI.Theme.Dark
	}

	if userThemeName != "" {
		theme, err = LoadTheme(userThemeName, theme, isDark)
		if err != nil {
			return ResolvedTheme{}, fmt.Errorf("loading user theme %q: %w", userThemeName, err)
		}
	}

	// Layer jj VCS colors
	if jjColors != nil {
		maps.Copy(theme.Colors, NormalizeColorSelectors(jjColors))
	}

	// Layer inline [ui.colors] overrides
	if Current.UI.Colors != nil {
		maps.Copy(theme.Colors, NormalizeColorSelectors(Current.UI.Colors))
	}

	return theme, nil
}
