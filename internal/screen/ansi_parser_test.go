package screen

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func Test_paramToStyle(t *testing.T) {
	type args struct {
		param string
	}
	tests := []struct {
		name string
		args args
		want lipgloss.Style
	}{
		{
			name: "empty",
			args: args{param: ""},
			want: lipgloss.NewStyle(),
		},
		{
			name: "bold",
			args: args{param: "1"},
			want: lipgloss.NewStyle().Bold(true),
		},
		{
			name: "italic",
			args: args{param: "3"},
			want: lipgloss.NewStyle().Italic(true),
		},
		{
			name: "underline",
			args: args{param: "4"},
			want: lipgloss.NewStyle().Underline(true),
		},
		{
			name: "dim",
			args: args{param: "2"},
			want: lipgloss.NewStyle().Faint(true),
		},
		{
			name: "reset",
			args: args{param: "0"},
			want: lipgloss.NewStyle(),
		},
		{
			name: "multiple styles",
			args: args{param: "1;3;4"},
			want: lipgloss.NewStyle().Bold(true).Italic(true).Underline(true),
		},
		{
			name: "multiple styles with reset",
			args: args{param: "0;1;3;4"},
			want: lipgloss.NewStyle().Bold(true).Italic(true).Underline(true),
		},
		{
			name: "multiple styles with invalid code",
			args: args{param: "1;3;99;4"},
			want: lipgloss.NewStyle().Bold(true).Italic(true).Underline(true),
		},
		{
			name: "multiple styles with reset and invalid code",
			args: args{param: "0;1;3;99;4"},
			want: lipgloss.NewStyle().Bold(true).Italic(true).Underline(true),
		},
		{
			name: "with fg color",
			args: args{param: "38;5;123"},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color("123")),
		},
		{
			name: "with bg color",
			args: args{param: "48;5;123"},
			want: lipgloss.NewStyle().Background(lipgloss.Color("123")),
		},
		{
			name: "with fg and bg color",
			args: args{param: "38;5;123;48;5;231"},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color("123")).Background(lipgloss.Color("231")),
		},
		{
			name: "with 8-bit color",
			args: args{param: "38;5;123"},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color("123")),
		},
		{
			name: "with 24-bit color",
			args: args{param: "38;2;255;0;0"},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		},
		{
			name: "with 24-bit bg color",
			args: args{param: "48;2;0;255;0"},
			want: lipgloss.NewStyle().Background(lipgloss.Color("#00ff00")),
		},
		{
			name: "with fg, bg, underline",
			args: args{param: "38;5;123;48;5;231;4"},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color("123")).Background(lipgloss.Color("231")).Underline(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, paramToStyle(tt.args.param), "paramToStyle(%v)", tt.args.param)
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want []Segment
	}{
		{
			name: "empty input",
			args: args{data: []byte("")},
		},
		{
			name: "simple text",
			args: args{data: []byte("Hello, World!")},
			want: []Segment{{Text: "Hello, World!", Style: lipgloss.NewStyle()}},
		},
		{
			name: "text with ANSI escape codes",
			args: args{data: []byte("\033[1;31mHello\033[0m")},
			want: []Segment{
				{Text: "Hello", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(1))},
			},
		},
		{
			name: "text with underline with fg",
			args: args{data: []byte("\033[4m\033[38;5;3mUnderlined Text\033[0m")},
			want: []Segment{
				{Text: "Underlined Text", Style: lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))},
			},
		},
		{
			name: "text with style reset followed by plain text",
			args: args{data: []byte("\033[1;4mBold Underlined\033[0mPlain Text")},
			want: []Segment{
				{Text: "Bold Underlined", Style: lipgloss.NewStyle().Bold(true).Underline(true)},
				{Text: "Plain Text", Style: lipgloss.NewStyle()},
			},
		},
		{
			name: "multiple styled segments",
			args: args{data: []byte("\033[1mBold\033[0m \033[4mUnderlined\033[0m \033[31mRed\033[0m")},
			want: []Segment{
				{Text: "Bold", Style: lipgloss.NewStyle().Bold(true)},
				{Text: " ", Style: lipgloss.NewStyle()},
				{Text: "Underlined", Style: lipgloss.NewStyle().Underline(true)},
				{Text: " ", Style: lipgloss.NewStyle()},
				{Text: "Red", Style: lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))},
			},
		},
		{
			name: "style bleeding test - bold then plain",
			args: args{data: []byte("\033[1mBold Text\033[0mPlain Text")},
			want: []Segment{
				{Text: "Bold Text", Style: lipgloss.NewStyle().Bold(true)},
				{Text: "Plain Text", Style: lipgloss.NewStyle()},
			},
		},
		{
			name: "style bleeding test - color then plain",
			args: args{data: []byte("\033[31mRed Text\033[0mPlain Text")},
			want: []Segment{
				{Text: "Red Text", Style: lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))},
				{Text: "Plain Text", Style: lipgloss.NewStyle()},
			},
		},
		{
			name: "underline disable with 24m",
			args: args{data: []byte("\033[4m\033[38;5;3m(no description set)\033[24m\033[39m")},
			want: []Segment{
				{Text: "(no description set)", Style: lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))},
			},
		},
		{
			name: "underline disable followed by new content",
			args: args{data: []byte("\033[4m\033[38;5;3m(content)\033[24m\033[39m\033[1m\033[38;5;14m(new content)\033[0m")},
			want: []Segment{
				{Text: "(content)", Style: lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))},
				{Text: "(new content)", Style: lipgloss.NewStyle().Underline(true).UnsetUnderline().Bold(true).Foreground(lipgloss.Color("14"))},
			},
		},
		{
			name: "underline disable then new style",
			args: args{data: []byte("\033[4mtext\033[24m\033[1mnew\033[0m")},
			want: []Segment{
				{Text: "text", Style: lipgloss.NewStyle().Underline(true)},
				{Text: "new", Style: lipgloss.NewStyle().Underline(true).UnsetUnderline().Bold(true)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Parse(tt.args.data)
			assert.Equalf(t, tt.want, actual, "Parse(%v)", tt.args.data)
		})
	}
}

func TestApplyParamsToStyle(t *testing.T) {
	type args struct {
		style lipgloss.Style
		param string
	}
	tests := []struct {
		name string
		args args
		want lipgloss.Style
	}{
		{
			name: "apply underline to existing bold style",
			args: args{
				style: lipgloss.NewStyle().Bold(true),
				param: "4",
			},
			want: lipgloss.NewStyle().Bold(true).Underline(true),
		},
		{
			name: "apply color to existing underline style",
			args: args{
				style: lipgloss.NewStyle().Underline(true),
				param: "38;5;3",
			},
			want: lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3")),
		},
		{
			name: "apply multiple styles to existing style",
			args: args{
				style: lipgloss.NewStyle().Bold(true),
				param: "4;38;5;3",
			},
			want: lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("3")),
		},
		{
			name: "reset should clear existing style",
			args: args{
				style: lipgloss.NewStyle().Bold(true).Underline(true),
				param: "0",
			},
			want: lipgloss.NewStyle(),
		},
		{
			name: "24 should disable underline",
			args: args{
				style: lipgloss.NewStyle().Underline(true),
				param: "24",
			},
			want: lipgloss.NewStyle().Underline(true).UnsetUnderline(),
		},
		{
			name: "24 should disable underline with other styles",
			args: args{
				style: lipgloss.NewStyle().Bold(true).Underline(true),
				param: "24",
			},
			want: lipgloss.NewStyle().Bold(true).Underline(true).UnsetUnderline(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, applyParamsToStyle(tt.args.style, tt.args.param), "applyParamsToStyle(%v, %v)", tt.args.style, tt.args.param)
		})
	}
}
