package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/idursun/jjui/internal/jj"
)

var Current = &Config{
	Keys: DefaultKeyMappings,
	UI: UIConfig{
		HighlightLight: "#a0a0a0",
		HighlightDark:  "#282a36",
		Colors: colors{
			Dimmed:   Color{Fg: "bright black"},
			Shortcut: Color{Fg: "magenta"},
		},
	},
	Preview: PreviewConfig{
		ExtraArgs:                []string{},
		OplogCommand:             []string{"op", "show", jj.OperationIdPlaceholder, "--color", "always"},
		FileCommand:              []string{"diff", "--color", "always", "-r", jj.ChangeIdPlaceholder, jj.FilePlaceholder},
		RevisionCommand:          []string{"show", "--color", "always", "-r", jj.ChangeIdPlaceholder},
		ShowAtStart:              false,
		WidthPercentage:          50,
		WidthIncrementPercentage: 5,
	},
	OpLog: OpLogConfig{
		Limit: 200,
	},
	ExperimentalLogBatchingEnabled: false,
}

type Config struct {
	Keys                           KeyMappings[keys] `toml:"keys"`
	UI                             UIConfig          `toml:"ui"`
	Preview                        PreviewConfig     `toml:"preview"`
	OpLog                          OpLogConfig       `toml:"oplog"`
	ExperimentalLogBatchingEnabled bool              `toml:"experimental_log_batching_enabled"`
	Limit                          int
}

type Color struct {
	Fg        string `toml:"fg"`
	Bg        string `toml:"bg"`
	Bold      bool   `toml:"bold"`
	Underline bool   `toml:"underline"`
}

type colors struct {
	Shortcut Color `toml:"shortcut"`
	Dimmed   Color `toml:"dimmed"`
}

type UIConfig struct {
	HighlightLight string `toml:"highlight_light"`
	HighlightDark  string `toml:"highlight_dark"`
	Colors         colors `toml:"colors"`
	// TODO(ilyagr): It might make sense to rename this to `auto_refresh_period` to match `--period` option
	// once we have a mechanism to deprecate the old name softly.
	AutoRefreshInterval int `toml:"auto_refresh_interval"`
}

type PreviewConfig struct {
	ExtraArgs                []string `toml:"extra_args"`
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

func getConfigFilePath() string {
	var configDirs []string

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

func getDefaultEditor() string {
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

	editor := getDefaultEditor()
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
