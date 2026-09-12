package main

import (
	"strings"
	"testing"
)

func TestIsSecretPromptFailsSafe(t *testing.T) {
	tests := []struct {
		prompt string
		secret bool
	}{
		{prompt: "Username for 'https://github.com':", secret: false},
		{prompt: "Login: ", secret: false},
		{prompt: "Password for 'https://github.com':", secret: true},
		{prompt: "Enter OTP: ", secret: true},
		{prompt: "Token: ", secret: true},
		{prompt: "Mot de passe: ", secret: true},
	}

	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			if got := isSecretPrompt(tt.prompt); got != tt.secret {
				t.Fatalf("isSecretPrompt(%q) = %v, want %v", tt.prompt, got, tt.secret)
			}
		})
	}
}

func TestAdjustAskpassPrompt(t *testing.T) {
	if got := adjustAskpassPrompt("\x1b[31mPassword\nfor key\x1b[0m"); got != "Password for key" {
		t.Fatalf("adjustAskpassPrompt() = %q", got)
	}
	if got := adjustAskpassPrompt("\t\r\n"); got != "askpass: " {
		t.Fatalf("empty prompt = %q", got)
	}
	if got := adjustAskpassPrompt(strings.Repeat("x", 201)); len([]rune(got)) != 201 || !strings.HasSuffix(got, "…") {
		t.Fatalf("long prompt was not bounded: %q", got)
	}
}
