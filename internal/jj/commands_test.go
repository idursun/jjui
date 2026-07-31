package jj

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsImmutableError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "real jj immutable error",
			err: errors.New(`Error: Commit 8ce1232b416f is immutable
Hint: Could not modify commit: srzzxumk 8ce1232b x
Hint: Immutable commits are used to protect shared history.
Hint: For more information, see:
      - https://docs.jj-vcs.dev/latest/config/#set-of-immutable-commits
      - ` + "`jj help -k config`" + `, "Set of immutable commits"
Hint: This operation would rewrite 1 immutable commits.
`),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("Error: No such revision 'abc123'"),
			want: false,
		},
		{
			name: "root commit error is not retryable with --ignore-immutable",
			err:  errors.New("Error: The root commit 000000000000 is immutable"),
			want: false,
		},
		{
			name: "empty error message",
			err:  errors.New(""),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsImmutableError(tc.err))
		})
	}
}

func TestBookmarkPatternCommandsUseExactStringPatterns(t *testing.T) {
	name := `1.3.63-+-json-length-"fix"\branch`
	remote := `origin+backup`

	assert.Equal(t, CommandArgs{"bookmark", "move", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--to", "abc123"}, BookmarkMove("abc123", name))
	assert.Equal(t, CommandArgs{"bookmark", "delete", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`}, BookmarkDelete(name))
	assert.Equal(t, CommandArgs{"bookmark", "forget", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`}, BookmarkForget(name))
	assert.Equal(t, CommandArgs{"bookmark", "track", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--remote", `exact:"origin+backup"`}, BookmarkTrack(name, remote))
	assert.Equal(t, CommandArgs{"bookmark", "untrack", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--remote", `exact:"origin+backup"`}, BookmarkUntrack(name, remote))
}
