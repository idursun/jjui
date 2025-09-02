package list

import "github.com/idursun/jjui/internal/ui/common/models"

type CheckableList[T models.ICheckable] struct {
	*List[T]
}

func NewCheckableList[T models.ICheckable]() *CheckableList[T] {
	return &CheckableList[T]{
		List: NewList[T](),
	}
}

func (c *CheckableList[T]) GetCheckedItems() []T {
	var ret []T
	for _, item := range c.Items {
		if item.IsChecked() {
			ret = append(ret, item)
		}
	}
	return ret
}
