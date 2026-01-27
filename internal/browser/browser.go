package browser

import (
	"os/exec"
	"runtime"
)

func Open(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		// Fall back to xdg-open for other Unix-like systems
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
