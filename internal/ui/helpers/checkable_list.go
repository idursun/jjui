package helpers

import "github.com/idursun/jjui/internal/ui/context/models"

type CheckableList[T models.Checkable] struct {
	*List[T]
}

func NewCheckableList[T models.Checkable]() *CheckableList[T] {
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
