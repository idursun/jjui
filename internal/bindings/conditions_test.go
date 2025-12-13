package bindings

import "testing"

func TestParseCondition_Eval(t *testing.T) {
	tests := []struct {
		expr  string
		state map[string]any
		want  bool
	}{
		{"revisions.focused", map[string]any{"revisions.focused": true}, true},
		{"revisions.focused && inline_describe.active", map[string]any{"revisions.focused": true, "inline_describe.active": false}, false},
		{"revisions.focused && inline_describe.active", map[string]any{"revisions.focused": true, "inline_describe.active": true}, true},
		{"lang == \"ts\" || lang == \"js\"", map[string]any{"lang": "ts"}, true},
		{"lang == \"ts\" || lang == \"js\"", map[string]any{"lang": "go"}, false},
		{"!inline_describe.active", map[string]any{"inline_describe.active": false}, true},
	}

	for _, tt := range tests {
		cond, err := ParseCondition(tt.expr)
		if err != nil {
			t.Fatalf("ParseCondition(%q) error = %v", tt.expr, err)
		}
		if got := cond.Eval(tt.state); got != tt.want {
			t.Fatalf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
		}
	}
}

func TestParseCondition_Invalid(t *testing.T) {
	if _, err := ParseCondition("lang === \"go\""); err == nil {
		t.Fatal("expected error for invalid operator")
	}
	if _, err := ParseCondition("(lang == \"go\""); err == nil {
		t.Fatal("expected error for missing )")
	}
}
