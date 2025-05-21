package jj

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
				output: "feat-1;.;false;false;false;9",
			},
			want: []Bookmark{
				{
					Name:      "feat-1",
					Remotes:   nil,
					Conflict:  false,
					Backwards: false,
					CommitId:  "9",
				},
			},
		},
		{
			name: "remote",
			args: args{
				output: `feature;.;false;false;false;b
feature;origin;true;false;false;b`,
			},
			want: []Bookmark{
				{
					Name: "feature",
					Remotes: []BookmarkRemote{
						{"origin", "b", true},
					},
					Conflict:  false,
					Backwards: false,
					CommitId:  "b",
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
