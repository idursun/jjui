package jj

import (
	"strings"
)

const (
	moveBookmarkTemplate = `name ++ ";" ++ if(remote, "remote", ".") ++ ";" ++ present ++ ";" ++ tracked ++ ";" ++ conflict ++ ";" ++ if(normal_target, normal_target.contained_in("%s"), false) ++ ";" ++ if(normal_target, normal_target.commit_id().shortest(1), "") ++ "\n"`
	allBookmarkTemplate  = `name ++ ";" ++ if(remote, remote, ".") ++ ";" ++ present ++ ";" ++ tracked ++ ";" ++ conflict ++ ";" ++ false ++ ";" ++ if(normal_target, normal_target.commit_id().shortest(1), "") ++ "\n"`
)

type BookmarkRemote struct {
	Remote   string
	CommitId string
	Tracked  bool
	Present  bool
}

type Bookmark struct {
	Name      string
	Local     *BookmarkRemote
	Remotes   []BookmarkRemote
	Conflict  bool
	Backwards bool
}

func (b Bookmark) BestCommitID() string {
	if b.Local != nil && b.Local.CommitId != "" {
		return b.Local.CommitId
	}
	for _, r := range b.Remotes {
		if r.CommitId != "" {
			return r.CommitId
		}
	}
	return ""
}

func (b Bookmark) IsDeletable() bool {
	return b.Local != nil && b.Local.Present
}

func (b Bookmark) IsTrackable() bool {
	return b.Local != nil && b.Local.Present && len(b.Remotes) == 0
}

func (b Bookmark) IsDeleted() bool {
	return b.Local != nil && !b.Local.Present
}

func ParseBookmarkListOutput(output string) []Bookmark {
	lines := strings.Split(output, "\n")
	bookmarkMap := make(map[string]*Bookmark)
	var orderedNames []string

	for _, b := range lines {
		parts := strings.Split(b, ";")
		if len(parts) != 7 {
			continue
		}

		name := parts[0]
		name = strings.Trim(name, "\"")
		remoteName := parts[1]
		present := parts[2] == "true"
		tracked := parts[3] == "true"
		conflict := parts[4] == "true"
		backwards := parts[5] == "true"
		commitId := parts[6]

		if remoteName == "git" {
			continue
		}

		bookmark, exists := bookmarkMap[name]
		if !exists {
			bookmark = &Bookmark{
				Name:      name,
				Conflict:  conflict,
				Backwards: backwards,
			}
			bookmarkMap[name] = bookmark
			orderedNames = append(orderedNames, name)
		}

		if remoteName == "." {
			bookmark.Local = &BookmarkRemote{
				Remote:   ".",
				CommitId: commitId,
				Tracked:  tracked,
				Present:  present,
			}
		} else {
			remote := BookmarkRemote{
				Remote:   remoteName,
				Tracked:  tracked,
				CommitId: commitId,
				Present:  present,
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
