package oplog

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

func RenderRow(r io.Writer, row Row, highlighted bool, width int) {
	//highlightColor := lipgloss.AdaptiveColor{
	//	Light: config.Current.UI.HighlightLight,
	//	Dark:  config.Current.UI.HighlightDark,
	//}
	//highlightSeq := lipgloss.ColorProfile().FromColor("0").Sequence(true)
	highlightSeq := "[48;5;236m" // dark gray background, can be changed to a different color
	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			if highlighted {
				fmt.Fprint(&lw, segment.WithBackground(highlightSeq).String())
			} else {
				fmt.Fprint(&lw, segment.String())
			}
		}
		line := lw.String()
		fmt.Fprint(r, line)
		if highlighted {
			lineWidth := lipgloss.Width(line)
			gap := width - lineWidth
			if gap > 0 {
				fmt.Fprintf(r, "\033[%sm%s\033[0m", highlightSeq, strings.Repeat(" ", gap))
			}
		}
		fmt.Fprint(r, "\n")
	}
}
