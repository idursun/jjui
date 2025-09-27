package view

import "github.com/charmbracelet/bubbles/help"

type IStatus interface {
	help.KeyMap
	Name() string
}
