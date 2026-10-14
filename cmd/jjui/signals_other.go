//go:build !unix

package main

import tea "charm.land/bubbletea/v2"

func watchThemeReloadSignals(_ *tea.Program) func() {
	return func() {}
}
