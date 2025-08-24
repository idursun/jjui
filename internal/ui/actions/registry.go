package actions

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Action[T any] interface {
	Call(args T)
}
