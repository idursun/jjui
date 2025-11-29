package common

type ViewRange struct {
	*ViewNode
	Start         int
	End           int
	FirstRowIndex int
	LastRowIndex  int
}
