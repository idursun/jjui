package revisions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreviewObject(t *testing.T) {
	tests := []struct {
		name      string
		obj       textObject
		wantKind  string
		wantValue string
	}{
		{"plain bookmark", textObject{Kind: textObjectBookmark, Value: "exp/text-objects"}, "bookmark", "exp/text-objects"},
		{"remote bookmark", textObject{Kind: textObjectBookmark, Value: "main", Remote: "origin"}, "bookmark", "main@origin"},
		{"quoted bookmark", textObject{Kind: textObjectBookmark, Value: "a b", Remote: "origin"}, "bookmark", `"a b"@origin`},
		{"conflicted bookmark", textObject{Kind: textObjectBookmark, Value: "main", Conflicted: true}, "conflicted_bookmark", `exact:"main"`},
		{"other kind", textObject{Kind: textObjectAuthor, Value: "me"}, "author", "me"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, value := previewObject(&tc.obj)
			assert.Equal(t, tc.wantKind, kind)
			assert.Equal(t, tc.wantValue, value)
		})
	}
}
