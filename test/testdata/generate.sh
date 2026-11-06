#!/usr/bin/env bash
# Regenerates the `jj log --color=debug` fixtures used by the parser and
# renderer tests. Run it from anywhere: ./test/testdata/generate.sh
set -euo pipefail

out="$(cd "$(dirname "$0")" && pwd)"
revisions_out="$out/../../internal/ui/revisions/testdata"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

: >"$work/config.toml"
export JJ_CONFIG="$work/config.toml"
export JJ_USER="Test User" JJ_EMAIL="test@example.com"
export JJ_TIMESTAMP="2026-01-02T03:04:05+00:00" JJ_OP_TIMESTAMP="2026-01-02T03:04:05+00:00"
export JJ_OP_HOSTNAME="host" JJ_OP_USERNAME="test"

seed=0
# j runs jj with a fresh randomness seed so change ids are stable but unique.
j() {
	seed=$((seed + 1))
	JJ_RANDOMNESS_SEED=$seed jj --no-pager --quiet "$@"
}
log() {
	j log --color=debug "$@"
}

new_repo() {
	rm -rf "$work/repo"
	mkdir "$work/repo"
	cd "$work/repo"
	j git init .
}

# graph.log: merges, a conflicted merge, bookmarks (one very long) and a
# working copy on a side branch.
new_repo
echo base >file
j describe -m "base"
j bookmark create main -r @
j new -m "feature a"
echo a >file
j new -m "feature a2"
j bookmark create feature-with-a-very-very-very-long-bookmark-name -r @
j new main -m "feature b"
echo b >file
j new 'description(exact:"feature a2\n")' 'description(exact:"feature b\n")' -m "conflicted merge"
j new main -m "side work"
log -r 'all()' >"$out/graph.log"
# elided.log: hiding commits between revisions produces elided markers.
log -r '@ | main | description(exact:"conflicted merge\n")' >"$out/elided.log"
cp "$out/elided.log" "$revisions_out/jj-log-with-elided.log"
# oneline.log, detailed.log and commit-id-second-line.log: other templates.
j describe -m $'side work\n\nwith a longer body\nover two lines'
log -r '@ | main' -T builtin_log_oneline >"$out/oneline.log"
# Only the root has no graph lines below it, giving a one-line row.
log -r 'root()' -T builtin_log_oneline >"$revisions_out/single-line-log.log"
log -r '@ | main' -T builtin_log_detailed >"$out/detailed.log"
log -r '@ | main' -T 'change_id.shortest() ++ " " ++ description.first_line() ++ "\n" ++ commit_id.shortest() ++ "\n"' >"$out/commit-id-second-line.log"

# evolog.log and hidden.log: rewritten commits.
hidden=$(j log -r 'main' --no-graph -T 'commit_id')
j describe -r main -m "base, reworded"
j evolog --color=debug -r main -T builtin_evolog_compact >"$out/evolog.log"
log -r "$hidden" >"$out/hidden.log"

# divergent.log: the same change rewritten at two operations.
new_repo
j describe -m "one"
j describe -m "two"
j --at-op @- describe -m "three"
log -r 'all()' >"$out/divergent.log"
