package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui"
)

var Version = "unknown"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("jjui version %s\n", Version)
		os.Exit(0)
	}
	location := os.Getenv("PWD")
	if len(os.Args) > 1 {
		location = os.Args[1]
	}
	if _, err := os.Stat(location + "/.jj"); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: There is no jj repo in \"%s\".\n", location)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(jj.JJ{Location: location}), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
