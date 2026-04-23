package flash

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/render"
)

const commandMarkWidth = 3

type CardRenderer struct {
}

func NewCardRenderer() CardRenderer {
	return CardRenderer{}
}

func (r CardRenderer) RenderMessage(command, text string, commandErr error, maxWidth int) string {
	return r.renderCard(command, text, commandErr, maxWidth, true, true)
}

func (r CardRenderer) RenderHistoryEntry(entry commandHistoryEntry, maxWidth int, selected bool) string {
	return r.renderCard(entry.Command, entry.Text, entry.Err, maxWidth, selected, selected)
}

func (r CardRenderer) RenderRunningCommand(command, indicator string, maxWidth int) string {
	if command == "" {
		return ""
	}
	return r.wrapCard(
		r.renderCommandLine(command, nil, true, indicator),
		maxWidth,
	)
}

func (r CardRenderer) renderCard(command, text string, commandErr error, maxWidth int, showBody bool, highlight bool) string {
	successStyle := common.DefaultPalette.Get("flash success")
	errorStyle := common.DefaultPalette.Get("flash error")

	statusStyle := successStyle
	if commandErr != nil {
		statusStyle = errorStyle
	}

	var parts []string
	if command != "" {
		parts = append(parts, r.renderCommandLine(command, commandErr, false, ""))
	}
	if showBody {
		bodyText := text
		if commandErr != nil {
			bodyText = commandErr.Error()
		}
		if bodyText != "" {
			bodyStyle := lipgloss.NewStyle()
			if highlight {
				bodyStyle = statusStyle
			}
			parts = append(parts, bodyStyle.Render(bodyText))
		}
	}

	borderStyle := lipgloss.NewStyle()
	if highlight {
		borderStyle = statusStyle
	}

	return r.wrapCard(strings.Join(parts, "\n"), maxWidth, borderStyle)
}

func (r CardRenderer) wrapCard(content string, maxWidth int, borderStyle ...lipgloss.Style) string {
	if render.BlockWidth(content) > maxWidth {
		content = lipgloss.NewStyle().Width(maxWidth).Render(content)
	}
	style := common.DefaultPalette.Get("flash text")
	if len(borderStyle) > 0 && borderStyle[0].GetForeground() != nil {
		style = borderStyle[0]
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(style.GetForeground()).
		Render(content)
}

func (r CardRenderer) renderCommandLine(command string, commandErr error, running bool, indicator string) string {
	if command == "" {
		return ""
	}
	successStyle := common.DefaultPalette.Get("flash success")
	errorStyle := common.DefaultPalette.Get("flash error")
	textStyle := common.DefaultPalette.Get("flash text")
	matchedStyle := common.DefaultPalette.Get("flash matched")

	mark := successStyle.Width(commandMarkWidth).Render("✓ ")
	if running {
		mark = textStyle.Width(commandMarkWidth).Render(indicator + " ")
	} else if commandErr != nil {
		mark = errorStyle.Width(commandMarkWidth).Render("✗ ")
	}
	return mark + colorizeCommand(command, textStyle, matchedStyle)
}

func colorizeCommand(cmd string, textStyle, matchedStyle lipgloss.Style) string {
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
