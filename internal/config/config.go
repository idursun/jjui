package config

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

//go:embed default/*.toml
var configFS embed.FS

var Current = loadDefaultConfig()

type Config struct {
	Keys                           KeyMappings[keys] `toml:"keys"`
	UI                             UIConfig          `toml:"ui"`
	Preview                        PreviewConfig     `toml:"preview"`
	OpLog                          OpLogConfig       `toml:"oplog"`
	Graph                          GraphConfig       `toml:"graph"`
	ExperimentalLogBatchingEnabled bool              `toml:"experimental_log_batching_enabled"`
	Limit                          int               `toml:"limit"`
}

type Color struct {
	Fg            string `toml:"fg"`
	Bg            string `toml:"bg"`
	Bold          bool   `toml:"bold"`
	Italic        bool   `toml:"italic"`
	Underline     bool   `toml:"underline"`
	Strikethrough bool   `toml:"strikethrough"`
	Reverse       bool   `toml:"reverse"`
}

func (c *Color) UnmarshalTOML(text any) error {
	switch v := text.(type) {
	case string:
		c.Fg = v
	case map[string]interface{}:
		if p, ok := v["fg"]; ok {
			c.Fg = p.(string)
		}
		if p, ok := v["bg"]; ok {
			c.Bg = p.(string)
		}
		if p, ok := v["bold"]; ok {
			c.Bold = p.(bool)
		}
		if p, ok := v["italic"]; ok {
			c.Italic = p.(bool)
		}
		if p, ok := v["underline"]; ok {
			c.Underline = p.(bool)
		}
		if p, ok := v["strikethrough"]; ok {
			c.Strikethrough = p.(bool)
		}
		if p, ok := v["reverse"]; ok {
			c.Reverse = p.(bool)
		}
	}
	return nil
}

type ThemeConfig struct {
	Dark  string `toml:"dark"`
	Light string `toml:"light"`
}

type UIConfig struct {
	Theme  string           `toml:"theme"`
	Themes ThemeConfig      `toml:"-"`
	Colors map[string]Color `toml:"colors"`
	// TODO(ilyagr): It might make sense to rename this to `auto_refresh_period` to match `--period` option
	// once we have a mechanism to deprecate the old name softly.
	AutoRefreshInterval int `toml:"auto_refresh_interval"`
}

func (ui *UIConfig) UnmarshalTOML(data interface{}) error {
	if m, ok := data.(map[string]interface{}); ok {
		if v, exists := m["auto_refresh_interval"]; exists {
			if i, ok := v.(int64); ok {
				ui.AutoRefreshInterval = int(i)
			}
		}

		if v, exists := m["colors"]; exists {
			if colorMap, ok := v.(map[string]interface{}); ok {
				ui.Colors = make(map[string]Color)
				for name, colorData := range colorMap {
					var color Color
					if err := color.UnmarshalTOML(colorData); err == nil {
						ui.Colors[name] = color
					}
				}
			}
		}

		if themeValue, exists := m["theme"]; exists {
			if themeStr, isString := themeValue.(string); isString {
				ui.Theme = themeStr
			} else if themeMap, isMap := themeValue.(map[string]interface{}); isMap {
				if dark, ok := themeMap["dark"].(string); ok {
					ui.Themes.Dark = dark
				}
				if light, ok := themeMap["light"].(string); ok {
					ui.Themes.Light = light
				}
			}
		}
	}

	return nil
}

type PreviewConfig struct {
	RevisionCommand          []string `toml:"revision_command"`
	OplogCommand             []string `toml:"oplog_command"`
	FileCommand              []string `toml:"file_command"`
	ShowAtStart              bool     `toml:"show_at_start"`
	WidthPercentage          float64  `toml:"width_percentage"`
	WidthIncrementPercentage float64  `toml:"width_increment_percentage"`
}

type OpLogConfig struct {
	Limit int `toml:"limit"`
}

type GraphConfig struct {
	BatchSize int `toml:"batch_size"`
}

type ShowOption string

const (
	ShowOptionDiff        ShowOption = "diff"
	ShowOptionInteractive ShowOption = "interactive"
)

func (s *ShowOption) UnmarshalText(text []byte) error {
	val := string(text)
	switch val {
	case string(ShowOptionDiff),
		string(ShowOptionInteractive):
		*s = ShowOption(val)
		return nil
	default:
		return fmt.Errorf("invalid value for 'show': %q. Allowed: none, interactive and diff", val)
	}
}

func GetDefaultEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	// Fallback to common editors if not set
	if editor == "" {
		candidates := []string{"nano", "vim", "vi", "notepad.exe"} // Windows fallback
		for _, candidate := range candidates {
			if p, err := exec.LookPath(candidate); err == nil {
				editor = p
				break
			}
		}
	}

	return editor
}

func Edit() int {
	configFile := getConfigFilePath()
	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		configPath := path.Dir(configFile)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			err = os.MkdirAll(configPath, 0o755)
			if err != nil {
				log.Fatal(err)
				return -1
			}
		}
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			_, err := os.Create(configFile)
			if err != nil {
				log.Fatal(err)
				return -1
			}
		}
	}

	editor := GetDefaultEditor()
	if editor == "" {
		log.Fatal("No editor found. Please set $EDITOR or $VISUAL")
	}

	cmd := exec.Command(editor, configFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return cmd.ProcessState.ExitCode()
}
