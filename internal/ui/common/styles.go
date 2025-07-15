package common

import (
	"github.com/idursun/jjui/internal/config"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Black         = lipgloss.Color("0")
	Red           = lipgloss.Color("1")
	Green         = lipgloss.Color("2")
	Yellow        = lipgloss.Color("3")
	Blue          = lipgloss.Color("4")
	Magenta       = lipgloss.Color("5")
	Cyan          = lipgloss.Color("6")
	White         = lipgloss.Color("7")
	BrightBlack   = lipgloss.Color("8")
	BrightRed     = lipgloss.Color("9")
	BrightGreen   = lipgloss.Color("10")
	BrightYellow  = lipgloss.Color("11")
	BrightBlue    = lipgloss.Color("12")
	BrightMagenta = lipgloss.Color("13")
	BrightCyan    = lipgloss.Color("14")
	BrightWhite   = lipgloss.Color("15")
)

var DefaultPalette = Palette{
	Normal:        lipgloss.NewStyle(),
	ChangeId:      lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Dimmed:        lipgloss.NewStyle().Foreground(BrightBlack),
	Shortcut:      lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Text:          lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Button:        lipgloss.NewStyle().Foreground(White).PaddingLeft(2).PaddingRight(2),
	FocusedButton: lipgloss.NewStyle().Foreground(BrightWhite).Background(Blue).PaddingLeft(2).PaddingRight(2),
	Added:         lipgloss.NewStyle().Foreground(Green),
	Deleted:       lipgloss.NewStyle().Foreground(Red),
	Modified:      lipgloss.NewStyle().Foreground(Cyan),
	Renamed:       lipgloss.NewStyle().Foreground(Cyan),
	Success:       lipgloss.NewStyle().Foreground(Green),
	Error:         lipgloss.NewStyle().Foreground(Red),
	StatusMode:    lipgloss.NewStyle().Foreground(Black).Bold(true).Background(Magenta),
	TargetMarker:  lipgloss.NewStyle().Bold(true).Foreground(Black).Background(Red),
	SourceMarker:  lipgloss.NewStyle().Foreground(Black).Background(Cyan).Bold(true),
	Selected:      lipgloss.NewStyle().Foreground(Cyan).Background(BrightBlack),
	Matched:       lipgloss.NewStyle().Foreground(Cyan),
}

type Palette struct {
	Normal        lipgloss.Style
	ChangeId      lipgloss.Style
	Dimmed        lipgloss.Style
	Shortcut      lipgloss.Style
	Text          lipgloss.Style
	Title         lipgloss.Style
	Button        lipgloss.Style
	FocusedButton lipgloss.Style
	Added         lipgloss.Style
	Deleted       lipgloss.Style
	Modified      lipgloss.Style
	Renamed       lipgloss.Style
	StatusMode    lipgloss.Style
	Success       lipgloss.Style
	Error         lipgloss.Style
	TargetMarker  lipgloss.Style
	SourceMarker  lipgloss.Style
	Matched       lipgloss.Style
	Selected      lipgloss.Style
}

func (p *Palette) Update(styleMap map[string]config.Color) {
	if color, ok := styleMap["change_id"]; ok {
		p.ChangeId = createStyleFrom(color)
	}
	if color, ok := styleMap["diff renamed"]; ok {
		p.Renamed = createStyleFrom(color)
	}
	if color, ok := styleMap["diff modified"]; ok {
		p.Modified = createStyleFrom(color)
	}
	if color, ok := styleMap["diff removed"]; ok {
		p.Deleted = createStyleFrom(color)
	}
	if color, ok := styleMap["dimmed"]; ok {
		p.Dimmed = createStyleFrom(color)
	}
	if color, ok := styleMap["shortcut"]; ok {
		p.Shortcut = createStyleFrom(color)
	}
	if color, ok := styleMap["success"]; ok {
		p.Success = createStyleFrom(color)
	}
	if color, ok := styleMap["error"]; ok {
		p.Error = createStyleFrom(color)
	}
	if color, ok := styleMap["status"]; ok {
		p.StatusMode = createStyleFrom(color)
	}
	if color, ok := styleMap["button"]; ok {
		p.Button = createStyleFrom(color).PaddingLeft(2).PaddingRight(2)
	}
	if color, ok := styleMap["target_marker"]; ok {
		p.TargetMarker = createStyleFrom(color)
	}
	if color, ok := styleMap["source_marker"]; ok {
		p.SourceMarker = createStyleFrom(color)
	}
	if color, ok := styleMap["matched"]; ok {
		p.Matched = createStyleFrom(color)
	}
	if color, ok := styleMap["selected"]; ok {
		p.Selected = createStyleFrom(color)
	}
	if color, ok := styleMap["text"]; ok {
		p.Text = createStyleFrom(color)
	}
	if color, ok := styleMap["title"]; ok {
		p.Title = createStyleFrom(color)
	}
	p.FocusedButton = lipgloss.NewStyle().
		Inherit(p.Button).
		Background(p.Selected.GetBackground()).
		PaddingLeft(2).
		PaddingRight(2)
}

func createStyleFrom(color config.Color) lipgloss.Style {
	style := lipgloss.NewStyle()
	if color.Fg != "" {
		style = style.Foreground(parseColor(color.Fg))
	}
	if color.Bg != "" {
		style = style.Background(parseColor(color.Bg))
	}
	if color.Bold {
		style = style.Bold(true)
	}
	if color.Underline {
		style = style.Underline(true)
	}
	return style
}

func parseColor(color string) lipgloss.Color {
	// if it's a hex color, return it directly
	if len(color) == 7 && color[0] == '#' {
		return lipgloss.Color(color)
	}
	// if it's an ANSI256 color, return it directly
	if v, err := strconv.Atoi(color); err == nil {
		if v >= 0 && v <= 255 {
			return lipgloss.Color(color)
		}
	}
	// otherwise, try to parse it as a named color
	switch color {
	case "black":
		return "0"
	case "red":
		return "1"
	case "green":
		return "2"
	case "yellow":
		return "3"
	case "blue":
		return "4"
	case "magenta":
		return "5"
	case "cyan":
		return "6"
	case "white":
		return "7"
	case "bright black":
		return "8"
	case "bright red":
		return "9"
	case "bright green":
		return "10"
	case "bright yellow":
		return "11"
	case "bright blue":
		return "12"
	case "bright magenta":
		return "13"
	case "bright cyan":
		return "14"
	case "bright white":
		return "15"
	default:
		if strings.HasPrefix(color, "ansi-color-") {
			code := strings.TrimPrefix(color, "ansi-color-")
			if v, err := strconv.Atoi(code); err == nil && v >= 0 && v <= 255 {
				return lipgloss.Color(code)
			}
		}
		return ""
	}
}
