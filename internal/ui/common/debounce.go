package common

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type debouncer struct {
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	debounceMu sync.Mutex
	debouncers = map[string]*debouncer{}
)

// Debounce waits for the given duration before running cmd; newer calls with
// the same identifier cancel previous ones.
func Debounce(identifier string, duration time.Duration, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	state := &debouncer{ctx: ctx, cancel: cancel}

	debounceMu.Lock()
	if previous, ok := debouncers[identifier]; ok {
		previous.cancel()
	}
	debouncers[identifier] = state
	debounceMu.Unlock()

	return func() tea.Msg {
		defer func() {
			debounceMu.Lock()
			if debouncers[identifier] == state {
				delete(debouncers, identifier)
			}
			debounceMu.Unlock()
		}()

		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-state.ctx.Done():
			return nil
		}

		debounceMu.Lock()
		latest := debouncers[identifier]
		debounceMu.Unlock()
		if latest != state {
			return nil
		}

		select {
		case <-state.ctx.Done():
			return nil
		default:
		}

		msg := cmd()

		debounceMu.Lock()
		latest = debouncers[identifier]
		debounceMu.Unlock()

		select {
		case <-state.ctx.Done():
			return nil
		default:
		}

		if latest != state {
			return nil
		}

		return msg
	}
}
