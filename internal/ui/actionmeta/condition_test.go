package actionmeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type state map[string]any

func (s state) QueryState(name string) (any, bool) { v, ok := s[name]; return v, ok }

func TestParseCondition(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		state state
		want  bool
	}{
		{"revision.has_bookmark", state{"revision.has_bookmark": true}, true},
		{"revision.has_bookmark && !revision.is_root", state{"revision.has_bookmark": true, "revision.is_root": false}, true},
		{"revision.a || revision.b", state{"revision.a": false, "revision.b": true}, true},
		{"(revision.a || revision.b) && !revision.c", state{"revision.a": false, "revision.b": true, "revision.c": false}, true},
		{"!revision.missing", state{}, false},
		{"true || revision.missing", state{}, false},
		{"revision.value", state{"revision.value": "yes"}, false},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			condition, err := ParseCondition(tc.expr)
			require.NoError(t, err)
			require.Equal(t, tc.want, condition.Matches(tc.state))
		})
	}
}

func TestParseConditionRejectsNonBooleanSyntax(t *testing.T) {
	for _, expr := range []string{"revision.flag == true", "foo()", "a & b", "!"} {
		_, err := ParseCondition(expr)
		require.Error(t, err, expr)
	}
}
