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
			display:    h.printMode(key.NewBinding(), name),
			searchTerm: normalizeSearch(name),
		}
	}

	help := binding.Help()
	return helpItem{
		display:    h.printMode(*binding, name),
		searchTerm: normalizeSearch(help.Key, help.Desc, name),
	}
}

func (h *Model) newBindingItem(binding key.Binding) helpItem {
	help := binding.Help()
	return helpItem{
		display:    h.printKeyBinding(binding),
		searchTerm: normalizeSearch(help.Key, help.Desc),
	}
}

func (h *Model) newKeyItem(key string, desc string) helpItem {
	return helpItem{
		display:    h.printKey(key, desc),
		searchTerm: normalizeSearch(key, desc),
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
	h.defaultMenu = helpMenu{
		0, 0,
		h.buildLeftGroups(),
		h.buildMiddleGroups(),
		h.buildRightGroups(),
	}
	// TODO: 132 is an arbitrary width that allows all column to display properly
	// update to use dynamic width based on column contents
	h.defaultMenu.width, h.defaultMenu.height = 132, h.calculateMaxHeight()
	h.searchQuery.Width = len(h.searchQuery.Placeholder)
}

func (h *Model) buildLeftGroups() menuColumn {
	jumpKeys := fmt.Sprintf("%s/%s/%s",
		h.keyMap.JumpToParent.Help().Key,
		h.keyMap.JumpToChildren.Help().Key,
		h.keyMap.JumpToWorkingCopy.Help().Key,
	)

	return menuColumn{
		itemGroup{
			h.newModeItem(nil, "UI"),
			h.newBindingItem(h.keyMap.Refresh),
			h.newBindingItem(h.keyMap.Help),
			h.newBindingItem(h.keyMap.Cancel),
			h.newBindingItem(h.keyMap.Quit),
			h.newBindingItem(h.keyMap.Suspend),
			h.newBindingItem(h.keyMap.Revset),
		},
		itemGroup{
			h.newModeItem(nil, "Exec"),
			h.newBindingItem(h.keyMap.ExecJJ),
			h.newBindingItem(h.keyMap.ExecShell),
		},
		itemGroup{
			h.newModeItem(nil, "Revisions"),
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
	}
}

func (h *Model) buildMiddleGroups() menuColumn {
	return menuColumn{
		itemGroup{
			h.newModeItem(&h.keyMap.Details.Mode, "Details"),
			h.newBindingItem(h.keyMap.Details.Close),
			h.newBindingItem(h.keyMap.Details.ToggleSelect),
			h.newBindingItem(h.keyMap.Details.Restore),
			h.newBindingItem(h.keyMap.Details.Split),
			h.newBindingItem(h.keyMap.Details.Squash),
			h.newBindingItem(h.keyMap.Details.Diff),
			h.newBindingItem(h.keyMap.Details.RevisionsChangingFile),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Evolog.Mode, "Evolog"),
			h.newBindingItem(h.keyMap.Evolog.Diff),
			h.newBindingItem(h.keyMap.Evolog.Restore),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Squash.Mode, "Squash"),
			h.newBindingItem(h.keyMap.Squash.KeepEmptied),
			h.newBindingItem(h.keyMap.Squash.Interactive),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Revert.Mode, "Revert"),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Rebase.Mode, "Rebase"),
			h.newBindingItem(h.keyMap.Rebase.Revision),
			h.newBindingItem(h.keyMap.Rebase.Source),
			h.newBindingItem(h.keyMap.Rebase.Branch),
			h.newBindingItem(h.keyMap.Rebase.Before),
			h.newBindingItem(h.keyMap.Rebase.After),
			h.newBindingItem(h.keyMap.Rebase.Onto),
			h.newBindingItem(h.keyMap.Rebase.Insert),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Duplicate.Mode, "Duplicate"),
			h.newBindingItem(h.keyMap.Duplicate.Onto),
			h.newBindingItem(h.keyMap.Duplicate.Before),
			h.newBindingItem(h.keyMap.Duplicate.After),
		},
	}
}

func (h *Model) buildRightGroups() menuColumn {
	customCommandItems := []helpItem{h.newModeItem(&h.keyMap.CustomCommands, "Custom Commands")}
	for _, command := range h.context.CustomCommands {
		customCommandItems = append(customCommandItems, h.newBindingItem(command.Binding()))
	}

	return menuColumn{
		itemGroup{
			h.newModeItem(&h.keyMap.Preview.Mode, "Preview"),
			h.newBindingItem(h.keyMap.Preview.ScrollUp),
			h.newBindingItem(h.keyMap.Preview.ScrollDown),
			h.newBindingItem(h.keyMap.Preview.HalfPageDown),
			h.newBindingItem(h.keyMap.Preview.HalfPageUp),
			h.newBindingItem(h.keyMap.Preview.Expand),
			h.newBindingItem(h.keyMap.Preview.Shrink),
			h.newBindingItem(h.keyMap.Preview.ToggleBottom),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Git.Mode, "Git"),
			h.newBindingItem(h.keyMap.Git.Push),
			h.newBindingItem(h.keyMap.Git.Fetch),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.Bookmark.Mode, "Bookmarks"),
			h.newBindingItem(h.keyMap.Bookmark.Move),
			h.newBindingItem(h.keyMap.Bookmark.Delete),
			h.newBindingItem(h.keyMap.Bookmark.Untrack),
			h.newBindingItem(h.keyMap.Bookmark.Track),
			h.newBindingItem(h.keyMap.Bookmark.Forget),
			helpItem{"", ""},
		},
		itemGroup{
			h.newModeItem(&h.keyMap.OpLog.Mode, "Oplog"),
			h.newBindingItem(h.keyMap.Diff),
			h.newBindingItem(h.keyMap.OpLog.Restore),
			helpItem{"", ""},
		},

		itemGroup{
			h.newModeItem(&h.keyMap.Leader, "Leader"),
			helpItem{"", ""},
		},
		customCommandItems,
	}
}
