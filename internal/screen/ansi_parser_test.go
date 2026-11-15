package screen

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

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
