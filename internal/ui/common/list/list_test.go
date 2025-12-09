package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockScrollableList is a mock implementation of IScrollableList for testing
type mockScrollableList struct {
	items         []string
	cursor        int
	firstRowIndex int
	lastRowIndex  int
	listName      string
}

func (m *mockScrollableList) Len() int {
	return len(m.items)
}

func (m *mockScrollableList) Cursor() int {
	return m.cursor
}

func (m *mockScrollableList) SetCursor(index int) {
	m.cursor = index
}

func (m *mockScrollableList) VisibleRange() (int, int) {
	return m.firstRowIndex, m.lastRowIndex
}

func (m *mockScrollableList) ListName() string {
	return m.listName
}

func (m *mockScrollableList) GetItemRenderer(index int) IItemRenderer {
	return nil
}

// mockStreamableList extends mockScrollableList to implement IStreamableList
type mockStreamableList struct {
	mockScrollableList
	hasMore bool
}

func (m *mockStreamableList) HasMore() bool {
	return m.hasMore
}

func TestScroll_EmptyList(t *testing.T) {
	mock := &mockScrollableList{
		items:         []string{},
		cursor:        0,
		firstRowIndex: 0,
		lastRowIndex:  10,
		listName:      "test list",
	}

	result := Scroll(mock, 1, false)

	assert.Equal(t, 0, result.NewCursor)
	assert.Nil(t, result.NavigateMessage)
	assert.False(t, result.RequestMore)
}

func TestScroll_SingleItemList(t *testing.T) {
	tests := []struct {
		name       string
		delta      int
		wantCursor int
		wantMsg    string
	}{
		{
			name:       "scroll down on single item",
			delta:      1,
			wantCursor: 0,
			wantMsg:    "Already at the bottom of test list",
		},
		{
			name:       "scroll up on single item",
			delta:      -1,
			wantCursor: 0,
			wantMsg:    "Already at the top of test list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockScrollableList{
				items:         []string{"item1"},
				cursor:        0,
				firstRowIndex: 0,
				lastRowIndex:  10,
				listName:      "test list",
			}

			result := Scroll(mock, tt.delta, false)

			assert.Equal(t, tt.wantCursor, result.NewCursor)
			assert.NotNil(t, result.NavigateMessage)
			assert.Equal(t, tt.wantMsg, result.NavigateMessage.Output)
		})
	}
}

func TestScroll_BoundaryMessages(t *testing.T) {
	tests := []struct {
		name        string
		startCursor int
		delta       int
		listName    string
		wantMsg     string
	}{
		{
			name:        "top boundary",
			startCursor: 0,
			delta:       -1,
			listName:    "revset `main`",
			wantMsg:     "Already at the top of revset `main`",
		},
		{
			name:        "bottom boundary",
			startCursor: 9,
			delta:       1,
			listName:    "revset `@`",
			wantMsg:     "Already at the bottom of revset `@`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]string, 10)
			mock := &mockScrollableList{
				items:         items,
				cursor:        tt.startCursor,
				firstRowIndex: 0,
				lastRowIndex:  10,
				listName:      tt.listName,
			}

			result := Scroll(mock, tt.delta, false)

			assert.NotNil(t, result.NavigateMessage)
			assert.Equal(t, tt.wantMsg, result.NavigateMessage.Output)
			assert.Nil(t, result.NavigateMessage.Err)
		})
	}
}

func TestScroll_EnsureCursorViewAlwaysTrue(t *testing.T) {
	mock := &mockScrollableList{
		items:         []string{"item1", "item2", "item3"},
		cursor:        1,
		firstRowIndex: 0,
		lastRowIndex:  10,
		listName:      "test list",
	}

	tests := []struct {
		name   string
		delta  int
		isPage bool
	}{
		{"scroll down", 1, false},
		{"scroll up", -1, false},
		{"page down", 1, true},
		{"page up", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Scroll(mock, tt.delta, tt.isPage)
			assert.True(t, result.EnsureCursorView)
		})
	}
}
