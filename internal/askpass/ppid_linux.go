package askpass

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func getPPid(pid int) (int, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		prefix, suffix, ok := strings.Cut(scan.Text(), "\t")
		if !ok || prefix != "PPid:" {
			continue
		}
		return strconv.Atoi(suffix)
	}

	return 0, scan.Err()
}
