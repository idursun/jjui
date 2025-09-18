package view

type ViewRange struct {
	*Sizeable
	Start         int
	End           int
	FirstRowIndex int
	LastRowIndex  int
}

//func (v *ViewRange) Reset() {
//	v.Start = 0
//	v.End = 0
//	v.FirstRowIndex = -1
//	v.LastRowIndex = -1
//}
