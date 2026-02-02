package render

// Z-index constants for layered rendering. Higher values render on top.
// Components should use these named constants instead of magic numbers.
const (
	// ZBase is for base content:
	// revisions list, diff view, oplog, details, describe, bookmark operations
	ZBase = 0

	// ZFuzzyInput is for fuzzy input fields, text inputs, and revset list items
	ZFuzzyInput = 1

	// ZPreview is for preview panels and split views
	ZPreview = 10

	// ZRevsetOverlay is for revset overlay content (above preview)
	ZRevsetOverlay = 15

	// ZDialogs is for dialogs (undo/redo confirmation, input fields)
	// that should appear above the preview panel
	ZDialogs = 50

	// ZMenuBorder is for menu borders (git, bookmarks, choose, custom_commands)
	ZMenuBorder = 100

	// ZMenuContent is for menu content items
	ZMenuContent = 101

	// ZOverlay is for overlays like sequence overlay and flash messages
	ZOverlay = 200

	// ZExpandedStatus is for the expanded status bar help overlay (highest UI overlay)
	ZExpandedStatus = 275

	// ZPassword is for password input (highest priority modal)
	ZPassword = 300
)
