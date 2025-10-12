package config

import (
	"strings"
)

func JoinKeys(keys []string) string {
	var joined []string
	for _, key := range keys {
		k := key
		switch key {
		case "up":
			k = "↑"
		case "down":
			k = "↓"
		case " ":
			k = "space"
		}
		joined = append(joined, k)
	}
	return strings.Join(joined, "/")
}
