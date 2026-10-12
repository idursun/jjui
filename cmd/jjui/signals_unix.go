//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
)

// Reload through the UI event loop.
func watchThemeReloadSignals(p *tea.Program) func() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	go func() {
		for range signals {
			p.Send(common.ReloadThemeMsg{})
		}
	}()
	return func() {
		signal.Stop(signals)
		close(signals)
	}
}
