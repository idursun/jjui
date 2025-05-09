package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

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
	version    bool
	editConfig bool
	help       bool
)

func init() {
	flag.StringVar(&revset, "revset", "", "Set default revset")
	flag.StringVar(&revset, "r", "", "Set default revset (same as --revset)")
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
		location = os.Getenv("PWD")
	}

	rootLocation, err := getJJRootDir(location)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: There is no jj repo in \"%s\".\n", location)
		os.Exit(1)
	}

	appContext := context.NewAppContext(rootLocation)

	p := tea.NewProgram(ui.New(appContext, revset), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
