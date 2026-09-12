package password

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
	"github.com/idursun/jjui/internal/ui/common"
)

func TestNewEchoMode(t *testing.T) {
	tests := []struct {
		name     string
		echo     bool
		expected textinput.EchoMode
	}{
		{name: "secret", echo: true, expected: textinput.EchoPassword},
		{name: "visible", echo: false, expected: textinput.EchoNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := New(common.TogglePasswordMsg{
				Prompt:       "credential: ",
				Password:     make(chan []byte, 1),
				EchoPassword: tt.echo,
			})
			if model.textInput.EchoMode != tt.expected {
				t.Fatalf("echo mode = %v, want %v", model.textInput.EchoMode, tt.expected)
			}
		})
	}
}
