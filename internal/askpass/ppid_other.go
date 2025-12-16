//go:build !darwin && !linux

package askpass

import (
	"github.com/tailscale/peercred"
)

func getPPid(pid int) (int, error) {
	return 0, peercred.ErrNotImplemented
}
