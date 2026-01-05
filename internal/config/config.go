package config

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

//go:embed default/*.toml
var configFS embed.FS

var Current = loadDefaultConfig()

type Config struct {
	Keys      KeyMappings[keys] `toml:"keys"`
	UI        UIConfig          `toml:"ui"`
	Suggest   SuggestConfig     `toml:"suggest"`
	Revisions RevisionsConfig   `toml:"revisions"`
	Preview   PreviewConfig     `toml:"preview"`
	OpLog     OpLogConfig       `toml:"oplog"`
	Limit     int               `toml:"limit"`
	Git       GitConfig         `toml:"git"`
	Ssh       SshConfig         `toml:"ssh"`
}

type Color struct {
	Fg            string `toml:"fg"`
	Bg            string `toml:"bg"`
	Bold          bool   `toml:"bold"`
	Italic        bool   `toml:"italic"`
	Underline     bool   `toml:"underline"`
	Strikethrough bool   `toml:"strikethrough"`
	Reverse       bool   `toml:"reverse"`
	flags         ColorAttribute
}

type ColorAttribute uint8

const (
	ColorAttributeBold ColorAttribute = 1 << iota
	ColorAttributeItalic
	ColorAttributeUnderline
	ColorAttributeStrikethrough
	ColorAttributeReverse
)

func (c *Color) IsSet(flags ColorAttribute) bool {
	return c.flags&flags == flags
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
			c.flags |= ColorAttributeBold
		}
		if p, ok := v["italic"]; ok {
			c.Italic = p.(bool)
			c.flags |= ColorAttributeItalic
		}
		if p, ok := v["underline"]; ok {
			c.Underline = p.(bool)
			c.flags |= ColorAttributeUnderline
		}
		if p, ok := v["strikethrough"]; ok {
			c.Strikethrough = p.(bool)
			c.flags |= ColorAttributeStrikethrough
		}
		if p, ok := v["reverse"]; ok {
			c.Reverse = p.(bool)
			c.flags |= ColorAttributeReverse
		}
	}
	return nil
}

type ThemeConfig struct {
	Dark  string `toml:"dark"`
	Light string `toml:"light"`
}

func (t *ThemeConfig) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		t.Dark = v
		t.Light = v
	case map[string]interface{}:
		if dark, ok := v["dark"]; ok {
			if darkStr, isString := dark.(string); isString {
				t.Dark = darkStr
			} else {
				return fmt.Errorf("invalid type for 'dark' in theme configuration: expected string, got %T", dark)
			}
		}
		if light, ok := v["light"]; ok {
			if lightStr, isString := light.(string); isString {
				t.Light = lightStr
			} else {
				return fmt.Errorf("invalid type for 'light' in theme configuration: expected string, got %T", light)
			}
		}
	}
	return nil
}

type TracerConfig struct {
	Enabled bool `toml:"enabled"`
}

type UIConfig struct {
	Theme  ThemeConfig      `toml:"theme"`
	Colors map[string]Color `toml:"colors"`
	// TODO(ilyagr): It might make sense to rename this to `auto_refresh_period` to match `--period` option
	// once we have a mechanism to deprecate the old name softly.
	AutoRefreshInterval        int          `toml:"auto_refresh_interval"`
	FlashMessageDisplaySeconds int          `toml:"flash_message_display_seconds"`
	Tracer                     TracerConfig `toml:"tracer"`
}

func GetExpiringFlashMessageTimeout(c *Config) time.Duration {
	return time.Duration(c.UI.FlashMessageDisplaySeconds) * time.Second
}

type RevisionsConfig struct {
	LogBatching  bool   `toml:"log_batching"`
	LogBatchSize int    `toml:"log_batch_size"`
	Template     string `toml:"template"`
	Revset       string `toml:"revset"`
}

type PreviewPosition int

const (
	PreviewPositionAuto PreviewPosition = iota
	PreviewPositionBottom
	PreviewPositionRight
)

type PreviewConfig struct {
	RevisionCommand          []string `toml:"revision_command"`
	OplogCommand             []string `toml:"oplog_command"`
	FileCommand              []string `toml:"file_command"`
	ShowAtStart              bool     `toml:"show_at_start"`
	Position                 string   `toml:"position"`
	WidthPercentage          float64  `toml:"width_percentage"`
	WidthIncrementPercentage float64  `toml:"width_increment_percentage"`
}

func GetPreviewPosition(c *Config) (PreviewPosition, error) {
	switch value := c.Preview.Position; value {
	case "auto":
		return PreviewPositionAuto, nil
	case "bottom":
		return PreviewPositionBottom, nil
	case "right":
		return PreviewPositionRight, nil
	default:
		return PreviewPositionAuto, fmt.Errorf("invalid value for 'preview.position': %q (expected one of: auto, bottom, right)", value)
	}
}

type OpLogConfig struct {
	Limit int `toml:"limit"`
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

type SuggestMode int

const (
	SuggestModeOff SuggestMode = iota
	SuggestModeFuzzy
	SuggestModeRegex
)

type SuggestConfig struct {
	Exec SuggestExecConfig `toml:"exec"`
}

type SuggestExecConfig struct {
	Mode string `toml:"mode"`
}

func GetSuggestExecMode(c *Config) (SuggestMode, error) {
	switch value := c.Suggest.Exec.Mode; value {
	case "off":
		return SuggestModeOff, nil
	case "fuzzy":
		return SuggestModeFuzzy, nil
	case "regex":
		return SuggestModeRegex, nil
	default:
		return SuggestModeOff, fmt.Errorf("invalid value for 'suggest.exec.mode': %q (expected one of: off, fuzzy, regex)", value)
	}
}

type GitConfig struct {
	DefaultRemote string `toml:"default_remote"`
}

func GetGitDefaultRemote(c *Config) string {
	remote := c.Git.DefaultRemote
	if strings.TrimSpace(remote) == "" {
		return "origin"
	}
	return remote
}

type SshConfig struct {
	HijackAskpass bool `toml:"hijack_askpass"`
}
