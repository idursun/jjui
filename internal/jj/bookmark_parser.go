package jj

import (
	"strings"
)

const moveBookmarkTemplate = `separate(";", name, if(remote, "remote", "."), tracked, conflict, normal_target.contained_in("%s"), normal_target.commit_id().shortest(1)) ++ "\n"`
const allBookmarkTemplate = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`

type BookmarkRemote struct {
	Remote   string
	CommitId string
	Tracked  bool
}

type Bookmark struct {
	Name      string
	Remotes   []BookmarkRemote
	Conflict  bool
	Backwards bool
	CommitId  string
}

func (b Bookmark) IsLocal() bool {
	return len(b.Remotes) == 0
}

func ParseBookmarkListOutput(output string) []Bookmark {
	bookmarks := strings.Split(output, "\n")
	var result []Bookmark
	for _, b := range bookmarks {
		parts := strings.Split(b, ";")
		if len(parts) < 5 {
			continue
		} else {
			name := parts[0]
			remote := parts[1]
			tracked := parts[2] == "true"
			conflict := parts[3] == "true"
			backwards := parts[4] == "true"
			commitId := parts[5]
			if remote == "." {
				bookmark := Bookmark{
					Name:      name,
					Conflict:  conflict,
					Backwards: backwards,
					CommitId:  commitId,
				}
				result = append(result, bookmark)
			} else if len(result) > 0 {
				previous := &result[len(result)-1]
				remote := BookmarkRemote{
					Remote:   remote,
					Tracked:  tracked,
					CommitId: commitId,
				}
				previous.Remotes = append(previous.Remotes, remote)
			}
		}
	}
	return result

}
