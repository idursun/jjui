package common

import (
	"log"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

var (
	Black        = lipgloss.Color("0")
	Red          = lipgloss.Color("1")
	Green        = lipgloss.Color("2")
	Yellow       = lipgloss.Color("3")
	Blue         = lipgloss.Color("4")
	Magenta      = lipgloss.Color("5")
	Cyan         = lipgloss.Color("6")
	White        = lipgloss.Color("7")
	BrightBlack  = lipgloss.Color("8")
	BrightRed    = lipgloss.Color("9")
	BrightGreen  = lipgloss.Color("10")
	BrightYellow = lipgloss.Color("11")
	BrightBlue   = lipgloss.Color("12")
	BrightMagent = lipgloss.Color("13")
	BrightCyan   = lipgloss.Color("14")
	BrightWhite  = lipgloss.Color("15")
)

type Color struct {
	Fg        string `toml:"fg"`
	Bg        string `toml:"bg"`
	Bold      bool   `toml:"bold"`
	Underline bool   `toml:"underline"`
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
		return ""
	}
}

var DefaultPalette = Palette{
	Normal:           lipgloss.NewStyle(),
	ChangeId:         lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Dimmed:           lipgloss.NewStyle().Foreground(BrightBlack),
	Shortcut:         lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	EmptyPlaceholder: lipgloss.NewStyle().Foreground(Green).Bold(true),
	ConfirmationText: lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Button:           lipgloss.NewStyle().Foreground(White).PaddingLeft(2).PaddingRight(2),
	FocusedButton:    lipgloss.NewStyle().Foreground(BrightWhite).Background(Blue).PaddingLeft(2).PaddingRight(2),
	Added:            lipgloss.NewStyle().Foreground(Green),
	Deleted:          lipgloss.NewStyle().Foreground(Red),
	Modified:         lipgloss.NewStyle().Foreground(Cyan),
	Renamed:          lipgloss.NewStyle().Foreground(Cyan),
	StatusSuccess:    lipgloss.NewStyle().Foreground(Green),
	StatusError:      lipgloss.NewStyle().Foreground(Red),
	StatusMode:       lipgloss.NewStyle().Foreground(Black).Bold(true).Background(Magenta),
	Drop:             lipgloss.NewStyle().Bold(true).Foreground(Black).Background(Red),
}

type Palette struct {
	Normal           lipgloss.Style
	ChangeId         lipgloss.Style
	Dimmed           lipgloss.Style
	Shortcut         lipgloss.Style
	EmptyPlaceholder lipgloss.Style
	ConfirmationText lipgloss.Style
	Button           lipgloss.Style
	FocusedButton    lipgloss.Style
	Added            lipgloss.Style
	Deleted          lipgloss.Style
	Modified         lipgloss.Style
	Renamed          lipgloss.Style
	StatusMode       lipgloss.Style
	StatusSuccess    lipgloss.Style
	StatusError      lipgloss.Style
	Drop             lipgloss.Style
	jjColors         map[string]Color
}

func (p *Palette) SetColors(jjColorList string) {
	p.jjColors = ParseColorMap(jjColorList)
	if style, ok := p.Get("change_id"); ok {
		p.ChangeId = style
	}
	if style, ok := p.Get("rest"); ok {
		p.Dimmed = style
	}
	if style, ok := p.Get("diff renamed"); ok {
		p.Renamed = style
	}
	if style, ok := p.Get("diff modified"); ok {
		p.Modified = style
	}
	if style, ok := p.Get("diff removed"); ok {
		p.Deleted = style
	}
}

func (p *Palette) Get(name string) (lipgloss.Style, bool) {
	if color, ok := p.jjColors[name]; ok {
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
		return style, true
	}
	return lipgloss.NewStyle(), false
}

func ParseColorMap(jjColorList string) map[string]Color {
	type Palette struct {
		Colors map[string]toml.Primitive `toml:"colors"`
	}

	var colorTable Palette
	md, err := toml.Decode(jjColorList, &colorTable)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}
	colorMap := make(map[string]Color)
	for name, prim := range colorTable.Colors {
		var c Color
		if err := md.PrimitiveDecode(prim, &c); err != nil {
			var fg string
			if err := md.PrimitiveDecode(prim, &fg); err == nil {
				c.Fg = fg
			}
		}
		colorMap[name] = c
	}
	return colorMap
}
