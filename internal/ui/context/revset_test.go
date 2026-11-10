package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNarrowedRevset(t *testing.T) {
	tests := []struct {
		name    string
		current string
		want    string
	}{
		{"default is replaced", "builtin_log()", "files(x)"},
		{"empty is replaced", "", "files(x)"},
		{"other revset is narrowed", "::feature", "::feature & files(x)"},
		{"union is parenthesised", "a | b", "(a | b) & files(x)"},
		{"narrowing twice", "::feature & author(me)", "::feature & author(me) & files(x)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &MainContext{DefaultRevset: "builtin_log()", CurrentRevset: tt.current}
			assert.Equal(t, tt.want, ctx.NarrowedRevset("files(x)"))
		})
	}
}
