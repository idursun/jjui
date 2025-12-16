package askpass

import (
	"golang.org/x/sys/unix"
)

func getPPid(pid int) (int, error) {
	kproc, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, err
	}
	return int(kproc.Eproc.Ppid), nil
}
