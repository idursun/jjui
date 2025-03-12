package main

import (
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

var Version = "unknown"

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return Version
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			println(getVersion())
			os.Exit(0)
		case "--config":
			exitCode := config.Edit()
			os.Exit(exitCode)
		}
	}
	location, err := getJJRootDir()
	if err != nil {
		location = os.Getenv("PWD")
	}
	if len(os.Args) > 1 {
		location = os.Args[1]
	}
	if _, err = os.Stat(location + "/.jj"); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: There is no jj repo in \"%s\".\n", location)
		os.Exit(1)
	}

	appContext := context.NewAppContext(location)

	p := tea.NewProgram(ui.New(appContext), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func getJJRootDir() (string, error) {
	cmd := exec.Command("jj", "root")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
