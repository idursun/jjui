package jj

import (
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLogAndEvologIncludeRootMetadata(t *testing.T) {
	logArgs := strings.Join(Log("@", 0, ""), " ")
	evologArgs := strings.Join(Evolog("@"), " ")
	assert.Contains(t, logArgs, "working_copies.len() > 0, root, empty")
	assert.Contains(t, logArgs, "diff.stat().total_added()")
	assert.Contains(t, logArgs, "diff.stat().total_removed()")
	assert.Contains(t, evologArgs, "commit.commit_id().shortest(), commit.root(), commit.empty()")
}

func TestDiffstatCommandsRespectShowDiffstat(t *testing.T) {
	orig := config.Current.Revisions.ShowDiffstat
	t.Cleanup(func() { config.Current.Revisions.ShowDiffstat = orig })

	config.Current.Revisions.ShowDiffstat = true
	assert.Contains(t, strings.Join(Log("@", 0, ""), " "), "diff.stat().total_added()")
	assert.Contains(t, strings.Join(Status("abc"), " "), "diff.stat().files()")

	config.Current.Revisions.ShowDiffstat = false
	assert.NotContains(t, strings.Join(Log("@", 0, ""), " "), "diff.stat()")
	assert.NotContains(t, strings.Join(Status("abc"), " "), "diff.stat()")
}

func TestFileStatsUsesMachineReadablePerFileTemplate(t *testing.T) {
	args := strings.Join(FileStats("abc"), " ")
	assert.Contains(t, args, "diff.stat().files()")
	assert.Contains(t, args, "lines_added()")
	assert.Contains(t, args, "lines_removed()")
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

func TestDiffRangeOmitsEmptyToRevision(t *testing.T) {
	assert.Equal(t,
		CommandArgs{"diff", "--from", "source", "--color", "always", "--ignore-working-copy"},
		DiffRange("source", ""),
	)
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
