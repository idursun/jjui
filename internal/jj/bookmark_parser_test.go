package jj

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBookmarkListOutput_WithNonLocalBookmarks(t *testing.T) {
	output := `alpha;origin;true;false;false;false;2
main;.;true;false;false;false;b
main;git;true;true;false;false;b
main;origin;true;true;false;false;b
zeta;origin;true;false;false;false;c`
	bookmarks := ParseBookmarkListOutput(output)
	assert.Len(t, bookmarks, 3)

	alpha := bookmarks[slices.IndexFunc(bookmarks, func(b Bookmark) bool { return b.Name == "alpha" })]
	assert.Nil(t, alpha.Local, "alpha should not have a local bookmark")
	assert.Len(t, alpha.Remotes, 1)
	main := bookmarks[slices.IndexFunc(bookmarks, func(b Bookmark) bool { return b.Name == "main" })]
	assert.NotNil(t, main.Local, "main should have a local bookmark")
	assert.Len(t, main.Remotes, 1)
	zeta := bookmarks[slices.IndexFunc(bookmarks, func(b Bookmark) bool { return b.Name == "zeta" })]
	assert.Nil(t, zeta.Local, "zeta should not have a local bookmark")
	assert.Len(t, zeta.Remotes, 1)
}

func TestParseBookmarkListOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name string
		args args
		want []Bookmark
	}{
		{
			name: "empty",
			args: args{
				output: "",
			},
			want: nil,
		},
		{
			name: "single",
			args: args{
				output: "feat-1;.;true;false;false;false;9",
			},
			want: []Bookmark{
				{
					Name:    "feat-1",
					Remotes: nil,
					Local: &BookmarkRemote{
						Remote:   ".",
						CommitId: "9",
						Tracked:  false,
						Present:  true,
					},
					Conflict:  false,
					Backwards: false,
				},
			},
		},
		{
			name: "remote",
			args: args{
				output: `feature;.;true;false;false;false;b
feature;origin;true;true;false;false;b`,
			},
			want: []Bookmark{
				{
					Name: "feature",
					Remotes: []BookmarkRemote{
						{
							Remote:   "origin",
							CommitId: "b",
							Tracked:  true,
							Present:  true,
						},
					},
					Local: &BookmarkRemote{
						Remote:   ".",
						CommitId: "b",
						Tracked:  false,
						Present:  true,
					},
					Conflict:  false,
					Backwards: false,
				},
			},
		},
		{
			name: "quoted bookmarks",
			args: args{
				output: `"test--bookmark";.;true;false;false;false;7
"test--bookmark";git;true;true;false;false;7
"test--bookmark";origin;true;true;false;false;6`,
			},
			want: []Bookmark{
				{
					Name: "test--bookmark",
					Remotes: []BookmarkRemote{
						{
							Remote:   "origin",
							CommitId: "6",
							Tracked:  true,
							Present:  true,
						},
					},
					Local: &BookmarkRemote{
						Remote:   ".",
						CommitId: "7",
						Tracked:  false,
						Present:  true,
					},
					Conflict:  false,
					Backwards: false,
				},
			},
		},
		{
			name: "deleted local bookmark",
			args: args{
				output: "main;.;false;false;false;false;\nmain;origin;true;true;false;false;abc123",
			},
			want: []Bookmark{
				{
					Name: "main",
					Remotes: []BookmarkRemote{
						{
							Remote:   "origin",
							CommitId: "abc123",
							Tracked:  true,
							Present:  true,
						},
					},
					Local: &BookmarkRemote{
						Remote:   ".",
						CommitId: "",
						Tracked:  false,
						Present:  false,
					},
					Conflict:  false,
					Backwards: false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ParseBookmarkListOutput(tt.args.output), "ParseBookmarkListOutput(%v)", tt.args.output)
		})
	}
}
