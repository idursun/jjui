package list

import (
	"fmt"
	"io"
	"testing"

	"github.com/idursun/jjui/internal/ui/view"
	"github.com/stretchr/testify/assert"
)

type item struct {
}

var _ IItemRenderer = (*testRenderer)(nil)

type testRenderer struct{}

func (t testRenderer) RenderItem(w io.Writer, index int) {
	fmt.Fprintf(w, "%d\n", index)
}

func (t testRenderer) GetItemHeight(index int) int {
	return 1
}

func TestListRenderer_Render(t *testing.T) {
	type args struct {
		itemCount      int
		cursor         int
		viewRangeStart int
		viewRangeEnd   int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "fills the view range",
			args: args{
				itemCount: 5,
			},
			want: "0\n1\n2\n3\n4",
		},
		{
			name: "view range is limited by the height",
			args: args{
				itemCount: 10,
			},
			want: "0\n1\n2\n3\n4",
		},
		{
			name: "doesn't modify the view range when cursor is in the view range ",
			args: args{
				itemCount: 5,
				cursor:    1,
			},
			want: "0\n1\n2\n3\n4",
		},
		{
			name: "update the view range according to cursor position (cursor at the end)",
			args: args{
				itemCount: 10,
				cursor:    9,
			},
			want: "5\n6\n7\n8\n9",
		},
		{
			name: "moves the view range if the cursor is moved out of the view range",
			args: args{
				itemCount:      10,
				cursor:         0,
				viewRangeStart: 5,
				viewRangeEnd:   9,
			},
			want: "0\n1\n2\n3\n4",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list := NewList[item]()
			for i := 0; i < test.args.itemCount; i++ {
				list.AppendItems(item{})
			}
			list.SetCursor(test.args.cursor)
			renderer := NewRenderer[item](list, testRenderer{}, view.NewSizeable(20, 5))
			renderer.Start = test.args.viewRangeStart
			renderer.End = test.args.viewRangeEnd
			rendered := renderer.Render()
			assert.Equal(t, test.want, rendered)
		})
	}
}

func TestListRenderer_Render_CursorMovement(t *testing.T) {
	type args struct {
		cursor []int
		start  int
		end    int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "view range doesn't change if the cursor moves within the view range",
			args: args{
				cursor: []int{1, 2, 3},
			},
			want: "0\n1\n2\n3\n4",
		},
		{
			name: "view range updates when the cursor moves out of the view range",
			args: args{
				cursor: []int{1, 2, 3, 4, 5},
			},
			want: "1\n2\n3\n4\n5",
		},
		{
			name: "view range updates when cursor jumps",
			args: args{
				cursor: []int{9, 0},
			},
			want: "0\n1\n2\n3\n4",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list := NewList[item]()
			for i := 0; i < 10; i++ {
				list.AppendItems(item{})
			}
			renderer := NewRenderer[item](list, testRenderer{}, view.NewSizeable(20, 5))
			renderer.Start = test.args.start
			renderer.End = test.args.end
			var rendered string
			for _, cursor := range test.args.cursor {
				list.SetCursor(cursor)
				rendered = renderer.Render()
			}
			assert.Equal(t, test.want, rendered)
		})
	}
}
