package helppage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func (h *Model) printKeyBinding(k key.Binding) string {
	return h.printKey(k.Help().Key, k.Help().Desc)
}

func (h *Model) printKey(key string, desc string) string {
	keyAligned := fmt.Sprintf("%9s", key)
	return lipgloss.JoinHorizontal(0, h.styles.shortcut.Render(keyAligned), h.styles.dimmed.Render(desc))
}

func (h *Model) printMode(key key.Binding, name string) string {
	keyAligned := fmt.Sprintf("%9s", key.Help().Key)
	return lipgloss.JoinHorizontal(0, h.styles.shortcut.Render(keyAligned), h.styles.title.Render(name))
}

func (h *Model) newModeItem(binding *key.Binding, name string) helpItem {
	if binding == nil {
		return helpItem{
			display:  h.printMode(key.NewBinding(), name),
			search:   normalizeSearch(name),
		
		}
	}

	help := binding.Help()
	return helpItem{
		display:  h.printMode(*binding, name),
		search:   normalizeSearch(help.Key, help.Desc, name),
	
	}
}

func (h *Model) newBindingItem(binding key.Binding) helpItem {
	help := binding.Help()
	return helpItem{
		display:  h.printKeyBinding(binding),
		search:   normalizeSearch(help.Key, help.Desc),
	
	}
}

func (h *Model) newKeyItem(key string, desc string) helpItem {
	return helpItem{
		display:  h.printKey(key, desc),
		search:   normalizeSearch(key, desc),
	
	}
}

// Joins an entry's keybind, description, and name
func normalizeSearch(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		clean = append(clean, strings.ToLower(part))
	}
	return strings.Join(clean, " ")
}

func (h *Model) setDefaultMenu() {
	h.defaultMenu = itemMenu{
		0, 0,
		h.buildLeftGroups(),
		h.buildMiddleGroups(),
		h.buildRightGroups(),
	}
	h.defaultMenu.width, h.defaultMenu.height = 45, h.defaultMenu.calculateMaxHeight()
	h.searchQuery.Width = h.defaultMenu.width - 10
}

func (h *Model) buildLeftGroups() []itemGroup {
	jumpKeys := fmt.Sprintf("%s/%s/%s",
		h.keyMap.JumpToParent.Help().Key,
		h.keyMap.JumpToChildren.Help().Key,
		h.keyMap.JumpToWorkingCopy.Help().Key,
	)

	uiTitle := h.newModeItem(nil, "UI")
	execTitle := h.newModeItem(nil, "Exec")
	revisionsTitle := h.newModeItem(nil, "Revisions")

	return []itemGroup{
		{
			groupHeader: &uiTitle,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Refresh),
				h.newBindingItem(h.keyMap.Help),
				h.newBindingItem(h.keyMap.Cancel),
				h.newBindingItem(h.keyMap.Quit),
				h.newBindingItem(h.keyMap.Suspend),
				h.newBindingItem(h.keyMap.Revset),
			},
		},
		{
			groupHeader: &execTitle,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.ExecJJ),
				h.newBindingItem(h.keyMap.ExecShell),
			},
		},
		{
			groupHeader: &revisionsTitle,
			groupItems: []helpItem{
				h.newKeyItem(jumpKeys, "jump to parent/child/working-copy"),
				h.newBindingItem(h.keyMap.ToggleSelect),
				h.newBindingItem(h.keyMap.AceJump),
				h.newBindingItem(h.keyMap.QuickSearch),
				h.newBindingItem(h.keyMap.QuickSearchCycle),
				h.newBindingItem(h.keyMap.FileSearch.Toggle),
				h.newBindingItem(h.keyMap.New),
				h.newBindingItem(h.keyMap.Commit),
				h.newBindingItem(h.keyMap.Describe),
				h.newBindingItem(h.keyMap.Edit),
				h.newBindingItem(h.keyMap.Diff),
				h.newBindingItem(h.keyMap.Diffedit),
				h.newBindingItem(h.keyMap.Split),
				h.newBindingItem(h.keyMap.Abandon),
				h.newBindingItem(h.keyMap.Absorb),
				h.newBindingItem(h.keyMap.Undo),
				h.newBindingItem(h.keyMap.Redo),
				h.newBindingItem(h.keyMap.Details.Mode),
				h.newBindingItem(h.keyMap.Bookmark.Set),
				h.newBindingItem(h.keyMap.InlineDescribe.Mode),
				h.newBindingItem(h.keyMap.SetParents),
			},
		},
	}
}

func (h *Model) buildMiddleGroups() []itemGroup {
	detailsMode := h.newModeItem(&h.keyMap.Details.Mode, "Details")
	evologMode := h.newModeItem(&h.keyMap.Evolog.Mode, "Evolog")
	squashMode := h.newModeItem(&h.keyMap.Squash.Mode, "Squash")
	revertMode := h.newModeItem(&h.keyMap.Revert.Mode, "Revert")
	rebaseMode := h.newModeItem(&h.keyMap.Rebase.Mode, "Rebase")
	duplicateMode := h.newModeItem(&h.keyMap.Duplicate.Mode, "Duplicate")

	return []itemGroup{
		{
			groupHeader: &detailsMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Details.Close),
				h.newBindingItem(h.keyMap.Details.ToggleSelect),
				h.newBindingItem(h.keyMap.Details.Restore),
				h.newBindingItem(h.keyMap.Details.Split),
				h.newBindingItem(h.keyMap.Details.Squash),
				h.newBindingItem(h.keyMap.Details.Diff),
				h.newBindingItem(h.keyMap.Details.RevisionsChangingFile),
			},
		},
		{
			groupHeader: &evologMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Evolog.Diff),
				h.newBindingItem(h.keyMap.Evolog.Restore),
			},
		},
		{
			groupHeader: &squashMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Squash.KeepEmptied),
				h.newBindingItem(h.keyMap.Squash.Interactive),
			},
		},
		{
			groupHeader: &revertMode,
		},
		{
			groupHeader: &rebaseMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Rebase.Revision),
				h.newBindingItem(h.keyMap.Rebase.Source),
				h.newBindingItem(h.keyMap.Rebase.Branch),
				h.newBindingItem(h.keyMap.Rebase.Before),
				h.newBindingItem(h.keyMap.Rebase.After),
				h.newBindingItem(h.keyMap.Rebase.Onto),
				h.newBindingItem(h.keyMap.Rebase.Insert),
			},
		},
		{
			groupHeader: &duplicateMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Duplicate.Onto),
				h.newBindingItem(h.keyMap.Duplicate.Before),
				h.newBindingItem(h.keyMap.Duplicate.After),
			},
		},
	}
}

func (h *Model) buildRightGroups() []itemGroup {
	previewMode := h.newModeItem(&h.keyMap.Preview.Mode, "Preview")
	gitMode := h.newModeItem(&h.keyMap.Git.Mode, "Git")
	bookmarksMode := h.newModeItem(&h.keyMap.Bookmark.Mode, "Bookmarks")
	oplogMode := h.newModeItem(&h.keyMap.OpLog.Mode, "Oplog")
	leaderMode := h.newModeItem(&h.keyMap.Leader, "Leader")
	customCommandsMode := h.newModeItem(&h.keyMap.CustomCommands, "Custom Commands")

	customCommandItems := make([]helpItem, 0, len(h.context.CustomCommands))
	for _, command := range h.context.CustomCommands {
		customCommandItems = append(customCommandItems, h.newBindingItem(command.Binding()))
	}

	return []itemGroup{
		{
			groupHeader: &previewMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Preview.ScrollUp),
				h.newBindingItem(h.keyMap.Preview.ScrollDown),
				h.newBindingItem(h.keyMap.Preview.HalfPageDown),
				h.newBindingItem(h.keyMap.Preview.HalfPageUp),
				h.newBindingItem(h.keyMap.Preview.Expand),
				h.newBindingItem(h.keyMap.Preview.Shrink),
				h.newBindingItem(h.keyMap.Preview.ToggleBottom),
			},
		},
		{
			groupHeader: &gitMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Git.Push),
				h.newBindingItem(h.keyMap.Git.Fetch),
			},
		},
		{
			groupHeader: &bookmarksMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Bookmark.Move),
				h.newBindingItem(h.keyMap.Bookmark.Delete),
				h.newBindingItem(h.keyMap.Bookmark.Untrack),
				h.newBindingItem(h.keyMap.Bookmark.Track),
				h.newBindingItem(h.keyMap.Bookmark.Forget),
			},
		},
		{
			groupHeader: &oplogMode,
			groupItems: []helpItem{
				h.newBindingItem(h.keyMap.Diff),
				h.newBindingItem(h.keyMap.OpLog.Restore),
			},
		},
		{
			groupHeader: &leaderMode,
		},
		{
			groupHeader: &customCommandsMode,
			groupItems:  customCommandItems,
		},
	}
}
