package dispatch

import (
	"strings"

	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
)

// DeriveOwner determines the intent owner from generated built-in metadata.
// Non-built-in actions have no owner.
func DeriveOwner(action keybindings.Action) string {
	actionName := strings.TrimSpace(string(action))
	if actionName == "" {
		return ""
	}
	if owners := actionmeta.ActionOwners(actionName); len(owners) > 0 {
		return owners[0]
	}
	return ""
}

// IsRevisionsOwner returns true if the owner routes to the revisions model.
func IsRevisionsOwner(owner string) bool {
	if owner == actions.OwnerRevisions {
		return true
	}
	if strings.HasPrefix(owner, "revisions.") {
		return true
	}
	return actions.IsRevisionsOwner(owner)
}
