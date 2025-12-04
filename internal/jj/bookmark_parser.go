package jj

import (
	"strings"
)

const (
	moveBookmarkTemplate = `separate(";", name, if(remote, "remote", "."), tracked, conflict, normal_target.contained_in("%s"), normal_target.commit_id().shortest(1)) ++ "\n"`
	allBookmarkTemplate  = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`
)

type BookmarkRemote struct {
	Remote   string
	CommitId string
	Tracked  bool
}

type Bookmark struct {
	Name      string
	Local     *BookmarkRemote
	Remotes   []BookmarkRemote
	Conflict  bool
	Backwards bool
	CommitId  string
}

func (b Bookmark) IsDeletable() bool {
	return b.Local != nil
}

func ParseBookmarkListOutput(output string) []Bookmark {
	lines := strings.Split(output, "\n")
	bookmarkMap := make(map[string]*Bookmark)
	var orderedNames []string

	for _, b := range lines {
		parts := strings.Split(b, ";")
		if len(parts) < 6 {
			continue
		}

		name := parts[0]
		name = strings.Trim(name, "\"")
		remoteName := parts[1]
		tracked := parts[2] == "true"
		conflict := parts[3] == "true"
		backwards := parts[4] == "true"
		commitId := parts[5]

		if remoteName == "git" {
			continue
		}

		bookmark, exists := bookmarkMap[name]
		if !exists {
			bookmark = &Bookmark{
				Name:      name,
				Conflict:  conflict,
				Backwards: backwards,
				CommitId:  commitId,
			}
			bookmarkMap[name] = bookmark
			orderedNames = append(orderedNames, name)
		}

		if remoteName == "." {
			bookmark.Local = &BookmarkRemote{
				Remote:   ".",
				CommitId: commitId,
				Tracked:  tracked,
			}
			bookmark.CommitId = commitId
		} else {
			remote := BookmarkRemote{
				Remote:   remoteName,
				Tracked:  tracked,
				CommitId: commitId,
			}
			if remoteName == "origin" {
				bookmark.Remotes = append([]BookmarkRemote{remote}, bookmark.Remotes...)
			} else {
				bookmark.Remotes = append(bookmark.Remotes, remote)
			}
		}
	}

	if len(orderedNames) == 0 {
		return nil
	}

	bookmarks := make([]Bookmark, len(orderedNames))
	for i, name := range orderedNames {
		bookmarks[i] = *bookmarkMap[name]
	}
	return bookmarks
}
