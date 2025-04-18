package config

import (
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var Current = &Config{
	Keys: DefaultKeyMappings,
	UI: UIConfig{
		HighlightLight: "#a0a0a0",
		HighlightDark:  "#282a36",
	},
	Preview: PreviewConfig{
		ExtraArgs:                []string{},
		ShowAtStart:              false,
		WidthPercentage:          50,
		WidthIncrementPercentage: 5,
	},
	OpLog: OpLogConfig{
		Limit: 200,
	},
}

type Config struct {
	Keys    KeyMappings[keys] `toml:"keys"`
	UI      UIConfig          `toml:"ui"`
	Preview PreviewConfig     `toml:"preview"`
	OpLog   OpLogConfig       `toml:"oplog"`
}

type UIConfig struct {
	HighlightLight string `toml:"highlight_light"`
	HighlightDark  string `toml:"highlight_dark"`
}

type PreviewConfig struct {
	ExtraArgs                []string `toml:"extra_args"`
	ShowAtStart              bool     `toml:"show_at_start"`
	WidthPercentage          float64  `toml:"width_percentage"`
	WidthIncrementPercentage float64  `toml:"width_increment_percentage"`
}

type OpLogConfig struct {
	Limit int `toml:"limit"`
}

func getConfigFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "jjui", "config.toml")
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

func Load() *Config {
	configFile := getConfigFilePath()
	_, err := os.Stat(configFile)
	if err != nil {
		return Current
	}
	_, err = toml.DecodeFile(configFile, &Current)
	if err != nil {
		return Current
	}
	return Current
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
