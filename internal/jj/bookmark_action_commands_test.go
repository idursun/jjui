package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBookmarkDeleteCommand(t *testing.T) {
	tests := []struct {
		name     string
		bookmark Bookmark
		want     CommandArgs
		ok       bool
	}{
		{
			name:     "local bookmark",
			bookmark: Bookmark{Name: "main", Local: &BookmarkRemote{Present: true}},
			want:     BookmarkDelete("main"),
			ok:       true,
		},
		{
			name:     "deleted local bookmark",
			bookmark: Bookmark{Name: "main", Local: &BookmarkRemote{Present: false}},
			ok:       false,
		},
		{
			name:     "remote only bookmark",
			bookmark: Bookmark{Name: "main"},
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BookmarkDeleteCommand(tt.bookmark)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBookmarkForgetCommand(t *testing.T) {
	tests := []struct {
		name     string
		bookmark Bookmark
		want     CommandArgs
		ok       bool
	}{
		{
			name:     "local bookmark",
			bookmark: Bookmark{Name: "main", Local: &BookmarkRemote{Present: true}},
			want:     BookmarkForget("main"),
			ok:       true,
		},
		{
			name:     "remote only bookmark",
			bookmark: Bookmark{Name: "main", Remotes: []BookmarkRemote{{Remote: "origin", Present: true}}},
			want:     BookmarkForget("main"),
			ok:       true,
		},
		{
			name:     "missing name",
			bookmark: Bookmark{},
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BookmarkForgetCommand(tt.bookmark)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBookmarkTrackLocalCommand(t *testing.T) {
	tests := []struct {
		name          string
		bookmark      Bookmark
		defaultRemote string
		want          CommandArgs
		ok            bool
	}{
		{
			name:          "present local bookmark",
			bookmark:      Bookmark{Name: "main", Local: &BookmarkRemote{Present: true}},
			defaultRemote: "origin",
			want:          BookmarkTrack("main", "origin"),
			ok:            true,
		},
		{
			name:          "deleted local bookmark",
			bookmark:      Bookmark{Name: "main", Local: &BookmarkRemote{Present: false}},
			defaultRemote: "origin",
			ok:            false,
		},
		{
			name:          "remote only bookmark",
			bookmark:      Bookmark{Name: "main"},
			defaultRemote: "origin",
			ok:            false,
		},
		{
			name:          "no default remote",
			bookmark:      Bookmark{Name: "main", Local: &BookmarkRemote{Present: true}},
			defaultRemote: "",
			ok:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BookmarkTrackLocalCommand(tt.bookmark, tt.defaultRemote)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBookmarkTrackRemoteCommand(t *testing.T) {
	tests := []struct {
		name     string
		bookmark Bookmark
		remote   BookmarkRemote
		want     CommandArgs
		ok       bool
	}{
		{
			name:     "untracked remote bookmark",
			bookmark: Bookmark{Name: "main"},
			remote:   BookmarkRemote{Remote: "origin", Tracked: false},
			want:     BookmarkTrack("main", "origin"),
			ok:       true,
		},
		{
			name:     "tracked remote bookmark",
			bookmark: Bookmark{Name: "main"},
			remote:   BookmarkRemote{Remote: "origin", Tracked: true},
			ok:       false,
		},
		{
			name:     "remote with no name",
			bookmark: Bookmark{Name: "main"},
			remote:   BookmarkRemote{},
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BookmarkTrackRemoteCommand(tt.bookmark, tt.remote)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBookmarkUntrackRemoteCommand(t *testing.T) {
	tests := []struct {
		name     string
		bookmark Bookmark
		remote   BookmarkRemote
		want     CommandArgs
		ok       bool
	}{
		{
			name:     "tracked remote bookmark",
			bookmark: Bookmark{Name: "main"},
			remote:   BookmarkRemote{Remote: "origin", Tracked: true},
			want:     BookmarkUntrack("main", "origin"),
			ok:       true,
		},
		{
			name:     "untracked remote bookmark",
			bookmark: Bookmark{Name: "main"},
			remote:   BookmarkRemote{Remote: "origin", Tracked: false},
			ok:       false,
		},
		{
			name:     "remote with no name",
			bookmark: Bookmark{Name: "main"},
			remote:   BookmarkRemote{Tracked: true},
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BookmarkUntrackRemoteCommand(tt.bookmark, tt.remote)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
