package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBookmarkPatternCommandsUseExactStringPatterns(t *testing.T) {
	name := `1.3.63-+-json-length-"fix"\branch`
	remote := `origin+backup`

	assert.Equal(t, CommandArgs{"bookmark", "move", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--to", "abc123"}, BookmarkMove("abc123", name))
	assert.Equal(t, CommandArgs{"bookmark", "delete", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`}, BookmarkDelete(name))
	assert.Equal(t, CommandArgs{"bookmark", "forget", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`}, BookmarkForget(name))
	assert.Equal(t, CommandArgs{"bookmark", "track", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--remote", `exact:"origin+backup"`}, BookmarkTrack(name, remote))
	assert.Equal(t, CommandArgs{"bookmark", "untrack", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--remote", `exact:"origin+backup"`}, BookmarkUntrack(name, remote))
}

func TestAnnotationCommandsUseStableMachineReadableOutput(t *testing.T) {
	assert.Equal(t,
		CommandArgs{"diff", "-r", "abc123", "--git", "--color", "never", "--ignore-working-copy"},
		AnnotationDiff("abc123"),
	)
	assert.Equal(t,
		CommandArgs{
			"file", "show", "-r", "abc123",
			"--color", "never", "--no-pager", "--quiet", "--ignore-working-copy",
			"--template", "",
			`file:"dir/a b.go"`,
		},
		FileShow("abc123", "dir/a b.go"),
	)
	assert.Equal(t,
		CommandArgs{
			"log", "-r", "abc123-", "--color", "never", "--no-graph", "--quiet",
			"--ignore-working-copy", "--template",
			"change_id.shortest() ++ if(divergent, \"/\" ++ change_offset) ++ \"\\t\" ++ description.first_line() ++ \"\\n\"",
		},
		GetRevisionSummariesFromRevset("abc123-"),
	)
}
