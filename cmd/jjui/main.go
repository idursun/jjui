package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui"
)

var Version string

func getVersion() string {
	if Version != "" {
		// set explicitly from build flags
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		// obtained by go build, usually from VCS
		return info.Main.Version
	}
	return "unknown"
}

var (
	revset     string
	period     int
	limit      int
	version    bool
	editConfig bool
	help       bool
)

func init() {
	flag.StringVar(&revset, "revset", "", "Set default revset")
	flag.StringVar(&revset, "r", "", "Set default revset (same as --revset)")
	flag.IntVar(&period, "period", -1, "Override auto-refresh interval (seconds, set to 0 to disable)")
	flag.IntVar(&period, "p", -1, "Override auto-refresh interval (alias for --period)")
	flag.IntVar(&limit, "limit", 0, "Number of revisions to show (default: 0)")
	flag.IntVar(&limit, "n", 0, "Number of revisions to show (alias for --limit)")
	flag.BoolVar(&version, "version", false, "Show version information")
	flag.BoolVar(&editConfig, "config", false, "Open configuration file in $EDITOR")
	flag.BoolVar(&help, "help", false, "Show help information")

	flag.Usage = func() {
		fmt.Printf("Usage: jjui [flags] [location]\n")
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}
}

func getJJRootDir(location string) (string, error) {
	cmd := exec.Command("jj", "root")
	cmd.Dir = location
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	flag.Parse()
	switch {
	case help:
		flag.Usage()
		os.Exit(0)
	case version:
		fmt.Println(getVersion())
		os.Exit(0)
	case editConfig:
		exitCode := config.Edit()
		os.Exit(exitCode)
	}

	var location string
	if args := flag.Args(); len(args) > 0 {
		location = args[0]
	}

	if location == "" {
		var err error
		if location, err = os.Getwd(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: couldn't determine the current directory: %v.\n", err)
			fmt.Fprintf(os.Stderr, "Please pass the location of a `jj` repo as an argument to `jjui`.\n")
			os.Exit(1)
		}
	}

	rootLocation, err := getJJRootDir(location)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: There is no jj repo in \"%s\".\n", location)
		os.Exit(1)
	}

	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			log.Fatalf("failed to set logging file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	} else {
		log.SetOutput(io.Discard)
	}

	if limit > 0 {
		config.Current.Limit = limit
	}

	appContext := context.NewAppContext(rootLocation)
	defer appContext.Histories.Flush()
	if output, err := config.LoadConfigFile(); err == nil {
		if err := config.Current.Load(string(output)); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}
		if registry, err := context.LoadCustomCommands(string(output)); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading custom commands: %v\n", err)
			os.Exit(1)
		} else {
			appContext.CustomCommands = registry
		}
		if registry, err := context.LoadLeader(string(output)); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading leader keys: %v\n", err)
			os.Exit(1)
		} else {
			appContext.Leader = registry
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var theme map[string]config.Color

	var defaultThemeName string
	if lipgloss.HasDarkBackground() {
		defaultThemeName = "default_dark"
	} else {
		defaultThemeName = "default_light"
	}

	theme, err = config.LoadEmbeddedTheme(defaultThemeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading default theme '%s': %v\n", defaultThemeName, err)
		os.Exit(1)
	}

	var userThemeName string
	if lipgloss.HasDarkBackground() {
		userThemeName = config.Current.UI.Theme.Dark
	} else {
		userThemeName = config.Current.UI.Theme.Light
	}

	if userThemeName != "" {
		theme, err = config.LoadTheme(userThemeName, theme)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading user theme '%s': %v\n", userThemeName, err)
			os.Exit(1)
		}
	}

	common.DefaultPalette.Update(theme)
	common.DefaultPalette.Update(appContext.JJConfig.GetApplicableColors())
	common.DefaultPalette.Update(config.Current.UI.Colors)

	if period >= 0 {
		config.Current.UI.AutoRefreshInterval = period
	}
	if revset != "" {
		appContext.DefaultRevset = revset
	} else if config.Current.Revisions.Revset != "" {
		appContext.DefaultRevset = config.Current.Revisions.Revset
	} else {
		appContext.DefaultRevset = appContext.JJConfig.Revsets.Log
	}
	appContext.CurrentRevset = appContext.DefaultRevset

	p := tea.NewProgram(ui.New(appContext), tea.WithAltScreen(), tea.WithReportFocus(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
