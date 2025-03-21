package common

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	Black          = lipgloss.Color("0")
	Red            = lipgloss.Color("1")
	Green          = lipgloss.Color("2")
	Yellow         = lipgloss.Color("3")
	Blue           = lipgloss.Color("4")
	Magenta        = lipgloss.Color("5")
	Cyan           = lipgloss.Color("6")
	White          = lipgloss.Color("7")
	IntenseBlack   = lipgloss.Color("8")
	IntenseRed     = lipgloss.Color("9")
	IntenseGreen   = lipgloss.Color("10")
	IntenseYellow  = lipgloss.Color("11")
	IntenseBlue    = lipgloss.Color("12")
	IntenseMagenta = lipgloss.Color("13")
	IntenseCyan    = lipgloss.Color("14")
	IntenseWhite   = lipgloss.Color("15")
)

var DropStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Black).
	Background(Red)

var DefaultPalette = Palette{
	Normal:           lipgloss.NewStyle(),
	ChangeId:         lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	CommitId:         lipgloss.NewStyle().Foreground(Blue).Bold(true),
	Rest:             lipgloss.NewStyle().Foreground(IntenseBlack),
	EmptyPlaceholder: lipgloss.NewStyle().Foreground(Green).Bold(true),
	ConfirmationText: lipgloss.NewStyle().Foreground(Magenta).Bold(true),
	Button:           lipgloss.NewStyle().Foreground(White).PaddingLeft(2).PaddingRight(2),
	FocusedButton:    lipgloss.NewStyle().Foreground(IntenseWhite).Background(Blue).PaddingLeft(2).PaddingRight(2),
	Added:            lipgloss.NewStyle().Foreground(Green),
	Deleted:          lipgloss.NewStyle().Foreground(Red),
	Modified:         lipgloss.NewStyle().Foreground(Cyan),
	Renamed:          lipgloss.NewStyle().Foreground(Cyan),
	Hint:             lipgloss.NewStyle().Foreground(IntenseBlack).PaddingLeft(1),
	StatusSuccess:    lipgloss.NewStyle().Foreground(Green),
	StatusError:      lipgloss.NewStyle().Foreground(Red),
	StatusMode:       lipgloss.NewStyle().Foreground(Black).Bold(true).Background(Magenta),
}

type Palette struct {
	Normal           lipgloss.Style
	ChangeId         lipgloss.Style
	CommitId         lipgloss.Style
	Rest             lipgloss.Style
	EmptyPlaceholder lipgloss.Style
	ConfirmationText lipgloss.Style
	Button           lipgloss.Style
	FocusedButton    lipgloss.Style
	ListTitle        lipgloss.Style
	ListItem         lipgloss.Style
	Added            lipgloss.Style
	Deleted          lipgloss.Style
	Modified         lipgloss.Style
	Renamed          lipgloss.Style
	Hint             lipgloss.Style
	StatusMode       lipgloss.Style
	StatusSuccess    lipgloss.Style
	StatusError      lipgloss.Style
}
