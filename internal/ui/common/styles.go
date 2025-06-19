package common

import (
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

var DefaultPalette = Palette{
	Normal:             lipgloss.NewStyle(),
	ChangeId:           lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Dimmed:             lipgloss.NewStyle().Foreground(BrightBlack),
	Shortcut:           lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	EmptyPlaceholder:   lipgloss.NewStyle().Foreground(Green).Bold(true),
	ConfirmationText:   lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Button:             lipgloss.NewStyle().Foreground(White).PaddingLeft(2).PaddingRight(2),
	FocusedButton:      lipgloss.NewStyle().Foreground(BrightWhite).Background(Blue).PaddingLeft(2).PaddingRight(2),
	Added:              lipgloss.NewStyle().Foreground(Green),
	Deleted:            lipgloss.NewStyle().Foreground(Red),
	Modified:           lipgloss.NewStyle().Foreground(Cyan),
	Renamed:            lipgloss.NewStyle().Foreground(Cyan),
	StatusSuccess:      lipgloss.NewStyle().Foreground(Green),
	StatusError:        lipgloss.NewStyle().Foreground(Red),
	StatusMode:         lipgloss.NewStyle().Foreground(Black).Bold(true).Background(Magenta),
	Drop:               lipgloss.NewStyle().Bold(true).Foreground(Black).Background(Red),
	CompletionSelected: lipgloss.NewStyle().Foreground(Cyan).Background(BrightBlack),
	CompletionMatched:  lipgloss.NewStyle().Foreground(Cyan),
}

type Palette struct {
	Normal             lipgloss.Style
	ChangeId           lipgloss.Style
	Dimmed             lipgloss.Style
	Shortcut           lipgloss.Style
	EmptyPlaceholder   lipgloss.Style
	ConfirmationText   lipgloss.Style
	Button             lipgloss.Style
	FocusedButton      lipgloss.Style
	Added              lipgloss.Style
	Deleted            lipgloss.Style
	Modified           lipgloss.Style
	Renamed            lipgloss.Style
	StatusMode         lipgloss.Style
	StatusSuccess      lipgloss.Style
	StatusError        lipgloss.Style
	Drop               lipgloss.Style
	CompletionMatched  lipgloss.Style
	CompletionSelected lipgloss.Style
}
