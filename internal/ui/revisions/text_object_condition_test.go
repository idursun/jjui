package revisions

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/stretchr/testify/assert"
)

func TestRevsetCondition(t *testing.T) {
	commit := &jj.Commit{ChangeId: "kkmpptxz", CommitId: "abc123"}
	tests := []struct {
		name       string
		obj        textObject
		want       string
		wantReason string
	}{
		{"node", textObject{Kind: textObjectNode}, "::kkmpptxz", ""},
		{"change id", textObject{Kind: textObjectChangeId}, "::kkmpptxz", ""},
		{"commit id", textObject{Kind: textObjectCommitId}, "::abc123", ""},
		{"bookmark", textObject{Kind: textObjectBookmark, Value: "main"}, "fork_point(trunk() | main)..main", ""},
		{"remote bookmark", textObject{Kind: textObjectBookmark, Value: "main", Remote: "origin"}, "fork_point(trunk() | main@origin)..main@origin", ""},
		{"quoted bookmark", textObject{Kind: textObjectBookmark, Value: "a b"}, `fork_point(trunk() | "a b").."a b"`, ""},
		{"conflicted bookmark", textObject{Kind: textObjectBookmark, Value: "main", Conflicted: true}, "", "Conflicted bookmarks cannot be added to the revset"},
		{"tag", textObject{Kind: textObjectTag, Value: "v1.0"}, "::v1.0", ""},
		{"workspace", textObject{Kind: textObjectWorkspace, Value: "default"}, "::default@", ""},
		{"author", textObject{Kind: textObjectAuthor, Value: "me@x.com"}, `author(exact:"me@x.com")`, ""},
		{"description first line", textObject{Kind: textObjectDescription, Value: "say \"hi\"\nmore"}, `description(substring:"say \"hi\"")`, ""},
		{"conflict", textObject{Kind: textObjectConflict}, "conflicts()", ""},
		{"divergent", textObject{Kind: textObjectDivergent}, "divergent()", ""},
		{"timestamp", textObject{Kind: textObjectTimestamp}, "", "This object cannot be added to the revset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := revsetCondition(&tt.obj, commit)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}
