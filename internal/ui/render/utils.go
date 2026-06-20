package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func ColorizeCommand(cmd string, textStyle, matchedStyle lipgloss.Style) string {
	tokens := strings.Split(strings.ReplaceAll(cmd, "\n", "⏎"), " ")
	var b strings.Builder
	for i, token := range tokens {
		if i > 0 {
			b.WriteByte(' ')
		}
		if strings.HasPrefix(token, "-") {
			b.WriteString(matchedStyle.Render(token))
		} else {
			b.WriteString(textStyle.Render(token))
		}
	}
	return b.String()
}
