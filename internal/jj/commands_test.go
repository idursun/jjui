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

func TestFileCommandsUseTypedRepositoryPaths(t *testing.T) {
	file := NewFileName(`dir/a "quoted"\file.go`)
	escaped := `file:"dir/a \"quoted\"\\file.go"`
	revisions := NewSelectedRevisions(&Commit{ChangeId: "source"})

	assert.Equal(t, CommandArgs{"diff", "-r", "source", "--color", "always", "--ignore-working-copy", escaped}, Diff("source", file))
	assert.Equal(t, CommandArgs{"split", "-r", "source", escaped}, Split("source", []FileName{file}, false, false))
	assert.Equal(t, CommandArgs{"restore", "-c", "source", escaped}, Restore("source", []FileName{file}, false))
	assert.Equal(t, CommandArgs{"squash", "--from", "source", "--into", "target", escaped}, Squash(revisions, "target", []FileName{file}, false, false, false, false))
	assert.Equal(t, CommandArgs{"absorb", "--from", "source", "--color", "never", escaped}, Absorb("source", nil, file))
}
