package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Captured with `jj log --color=debug` (ANSI codes kept, see the \x1b bytes).
const labelledLog = "\x1b[1m\x1b[38;5;2m<<log commit node working_copy mutable::@>>\x1b[0m  " +
	"\x1b[1m\x1b[38;5;13m<<log commit working_copy mutable divergent change_id shortest prefix::zr>>\x1b[38;5;8m<<log commit working_copy mutable divergent change_id shortest rest::ukyysv>>" +
	"<<log commit working_copy mutable divergent change_offset::/1>><<log commit working_copy mutable:: >>" +
	"<<log commit working_copy mutable author email local::someone>><<log commit working_copy mutable author email::@>><<log commit working_copy mutable author email domain::example.com>>" +
	"<<log commit working_copy mutable:: >><<log commit working_copy mutable commit_id shortest prefix::9>><<log commit working_copy mutable commit_id shortest rest::45f2f35>><<log commit working_copy mutable::>>\x1b[0m\n" +
	"│  <<log commit working_copy mutable empty::(empty)>><<log commit working_copy mutable:: >><<log commit working_copy mutable description first_line::a >> b << c::d>><<log commit working_copy mutable::>>\n" +
	"<<log commit node immutable::◆>>  <<log commit immutable change_id shortest prefix::m>><<log commit immutable change_id shortest rest::orttnoo>><<log commit immutable:: >>" +
	"<<log commit immutable bookmarks name::main>><<log commit immutable bookmarks::@>><<log commit immutable bookmarks remote::origin>><<log commit immutable:: >>" +
	"<<log commit immutable commit_id shortest prefix::b>><<log commit immutable commit_id shortest rest::a61a80e>><<log commit immutable::>>\n" +
	"│  <<log commit immutable description first_line::fix things>><<log commit immutable::>>\n" +
	"<<log commit node elided::~>>  <<elided::(elided revisions)>>\n" +
	"<<log commit node immutable::◆>>  <<log commit immutable change_id shortest prefix::zz>><<log commit immutable change_id shortest rest::zzzzzz>><<log commit immutable:: >>" +
	"<<log commit immutable root::root()>><<log commit immutable:: >><<log commit immutable commit_id shortest prefix::0>><<log commit immutable commit_id shortest rest::0000000>><<log commit immutable::>>\n"

func TestParseRows_LabelledOutputWithoutPrefixUsesRenderedId(t *testing.T) {
	log := "<<log commit node mutable::○>>  <<log commit change_id::abcdefgh>><<log commit:: >><<log commit commit_id::12345678>>\n"
	rows := ParseRows(strings.NewReader(log))
	assert.Len(t, rows, 1)
	assert.Equal(t, "abcdefgh", rows[0].Commit.ChangeId)
	assert.Equal(t, "12345678", rows[0].Commit.CommitId)
}

func TestParseRows_LabelledOutput(t *testing.T) {
	rows := ParseRows(strings.NewReader(labelledLog))
	assert.Len(t, rows, 3)

	wc := rows[0].Commit
	assert.Equal(t, "zr/1", wc.ChangeId)
	assert.Equal(t, "9", wc.CommitId)
	assert.True(t, wc.IsWorkingCopy)
	assert.True(t, wc.HasWorkspace)
	assert.True(t, wc.IsEmpty)
	assert.False(t, wc.IsImmutable)
	assert.False(t, wc.HasBookmarks)
	assert.Equal(t, 3, rows[0].Indent)

	main := rows[1].Commit
	assert.Equal(t, "m", main.ChangeId)
	assert.Equal(t, "b", main.CommitId)
	assert.True(t, main.HasBookmarks)
	assert.True(t, main.IsImmutable)
	assert.False(t, main.HasLocalBookmarks)
	assert.False(t, main.IsWorkingCopy)
	assert.Len(t, rows[1].Lines, 3, "elided line belongs to the previous revision")
	assert.Equal(t, Elided, rows[1].Lines[2].Flags&Elided)

	root := rows[2].Commit
	assert.True(t, root.IsRoot())
	assert.True(t, root.IsImmutable)
	assert.Equal(t, "0", root.CommitId)

	var description strings.Builder
	for _, s := range rows[0].Lines[1].Segments {
		description.WriteString(s.Text)
	}
	assert.Equal(t, "(empty) a >> b << c::d", description.String())
}
