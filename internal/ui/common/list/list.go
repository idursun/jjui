package list

import (
	"io"

	"github.com/idursun/jjui/internal/ui/common"
)

type IList interface {
	Len() int
	GetItemRenderer(index int) IItemRenderer
}

type IListCursor interface {
	Cursor() int
	SetCursor(index int)
}

type IItemRenderer interface {
	Render(w io.Writer, width int)
	Height() int
}

// IScrollableList defines the interface for list models that support scrolling navigation.
type IScrollableList interface {
	IList
	IListCursor

	// VisibleRange returns the first and last visible row indices,
	// used to calculate page-based scrolling distances.
	VisibleRange() (firstRowIndex, lastRowIndex int)

	// ListName returns a name for the list, used in boundary
	// messages like "Already at the top of {ListName}".
	ListName() string
}

// IStreamableList is list interface that can dynamically load more data
// If HasMore() returns true, the scroll function will request more items when
// scrolling past the current end of the list.
type IStreamableList interface {
	HasMore() bool
}

type ScrollResult struct {
	NewCursor        int
	EnsureCursorView bool
	NavigateMessage  *common.CommandCompletedMsg
	RequestMore      bool
}

// Scroll calculates the new cursor position for list navigation.
// It returns a ScrollResult containing the new cursor position and any messages.
// The caller is responsible for updating the cursor using the returned NewCursor value.
func Scroll(nav IScrollableList, delta int, isPage bool) ScrollResult {
	currentCursor := nav.Cursor()
	totalItems := nav.Len()
	firstRowIndex, lastRowIndex := nav.VisibleRange()
	contextName := nav.ListName()

	result := ScrollResult{
		NewCursor:        currentCursor,
		EnsureCursorView: true,
	}

	if totalItems == 0 {
		return result
	}

	// Check if more items are available
	streamable, isStreamable := nav.(IStreamableList)
	hasMore := isStreamable && streamable.HasMore()

	step := delta
	if isPage {
		span := max(lastRowIndex-firstRowIndex-1, 1)
		if step < 0 {
			step = -span
		} else {
			step = span
		}
	}

	if step > 0 {
		// Moving down
		if currentCursor == totalItems-1 && !hasMore {
			result.NavigateMessage = &common.CommandCompletedMsg{
				Output: "Already at the bottom of " + contextName,
				Err:    nil,
			}
			return result
		}
		if currentCursor+step < totalItems {
			result.NewCursor = currentCursor + step
		} else if isStreamable && hasMore {
			// Request more data only if the list implements streaming AND there's more data
			result.RequestMore = true
			result.NewCursor = currentCursor
		} else if totalItems > 0 {
			// no further scroll: clamp to the last item
			result.NewCursor = totalItems - 1
		}
	} else {
		// Moving up
		if currentCursor == 0 {
			result.NavigateMessage = &common.CommandCompletedMsg{
				Output: "Already at the top of " + contextName,
				Err:    nil,
			}
			return result
		}
		amount := -step
		if currentCursor > 0 {
			result.NewCursor = max(currentCursor-amount, 0)
		}
	}

	return result
}
