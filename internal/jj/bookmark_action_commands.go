package jj

func BookmarkDeleteCommand(bookmark Bookmark) (CommandArgs, bool) {
	if !bookmark.IsDeletable() {
		return nil, false
	}
	return BookmarkDelete(bookmark.Name), true
}

func BookmarkForgetCommand(bookmark Bookmark) (CommandArgs, bool) {
	if bookmark.Name == "" {
		return nil, false
	}
	return BookmarkForget(bookmark.Name), true
}

func BookmarkTrackLocalCommand(bookmark Bookmark, defaultRemote string) (CommandArgs, bool) {
	if bookmark.Local == nil || !bookmark.Local.Present || defaultRemote == "" {
		return nil, false
	}
	return BookmarkTrack(bookmark.Name, defaultRemote), true
}

func BookmarkTrackRemoteCommand(bookmark Bookmark, remote BookmarkRemote) (CommandArgs, bool) {
	if remote.Remote == "" || remote.Tracked {
		return nil, false
	}
	return BookmarkTrack(bookmark.Name, remote.Remote), true
}

func BookmarkUntrackRemoteCommand(bookmark Bookmark, remote BookmarkRemote) (CommandArgs, bool) {
	if remote.Remote == "" || !remote.Tracked {
		return nil, false
	}
	return BookmarkUntrack(bookmark.Name, remote.Remote), true
}
