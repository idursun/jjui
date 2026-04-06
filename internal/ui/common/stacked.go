package common

import (
	"github.com/idursun/jjui/internal/ui/dispatch"
)

// StackedModel is the contract for models presented in the stacked overlay.
type StackedModel interface {
	ImmediateModel
	dispatch.ScopeProvider
}
