package diff

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// FileList represents the file list panel in the diff viewer
type FileList struct {
	files         []*DiffFile
	selectedIndex int
	startLine     int
	height        int
	width         int
}

// NewFileList creates a new file list from parsed diff files
func NewFileList(files []*DiffFile) *FileList {
	return &FileList{
		files:         files,
		selectedIndex: 0,
		startLine:     0,
	}
}

// SelectedFile returns the currently selected file
func (fl *FileList) SelectedFile() *DiffFile {
	if fl.selectedIndex >= 0 && fl.selectedIndex < len(fl.files) {
		return fl.files[fl.selectedIndex]
	}
	return nil
}

// SelectedIndex returns the currently selected file index
func (fl *FileList) SelectedIndex() int {
	return fl.selectedIndex
}

// SetSelectedIndex sets the selected file index
func (fl *FileList) SetSelectedIndex(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(fl.files) {
		index = len(fl.files) - 1
	}
	if index < 0 {
		index = 0
	}
	fl.selectedIndex = index
	fl.ensureVisible()
}

// MoveUp moves selection up
func (fl *FileList) MoveUp() {
	if fl.selectedIndex > 0 {
		fl.selectedIndex--
		fl.ensureVisible()
	}
}

// MoveDown moves selection down
func (fl *FileList) MoveDown() {
	if fl.selectedIndex < len(fl.files)-1 {
		fl.selectedIndex++
		fl.ensureVisible()
	}
}

// FileCount returns the number of files
func (fl *FileList) FileCount() int {
	return len(fl.files)
}

// ensureVisible adjusts scrolling to keep selection visible
func (fl *FileList) ensureVisible() {
	if fl.height <= 0 {
		return
	}

	if fl.selectedIndex < fl.startLine {
		fl.startLine = fl.selectedIndex
	}
	if fl.selectedIndex >= fl.startLine+fl.height {
		fl.startLine = fl.selectedIndex - fl.height + 1
	}
}

// ViewRect renders the file list to the display context
func (fl *FileList) ViewRect(dl *render.DisplayContext, box layout.Box) {
	fl.height = box.R.Dy()
	fl.width = box.R.Dx()

	if fl.height <= 0 || fl.width <= 0 {
		return
	}

	// Styles
	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Reverse(true)

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // Green
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	renamedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
	copiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // Cyan

	// Render visible files
	y := box.R.Min.Y
	for i := fl.startLine; i < len(fl.files) && y < box.R.Max.Y; i++ {
		file := fl.files[i]
		isSelected := i == fl.selectedIndex

		// Build status indicator
		var statusStyle lipgloss.Style
		switch file.Status {
		case FileAdded:
			statusStyle = addedStyle
		case FileDeleted:
			statusStyle = deletedStyle
		case FileRenamed:
			statusStyle = renamedStyle
		case FileCopied:
			statusStyle = copiedStyle
		default:
			statusStyle = normalStyle
		}

		status := file.Status.String()
		path := file.Path()

		// Truncate path if too long
		maxPathLen := fl.width - 3 // "M " + path + padding
		if len(path) > maxPathLen && maxPathLen > 3 {
			path = "..." + path[len(path)-maxPathLen+3:]
		}

		// Build the line
		lineStyle := normalStyle
		if isSelected {
			lineStyle = selectedStyle
		}

		// Create the content
		tb := dl.Text(box.R.Min.X, y, 0)

		if isSelected {
			// When selected, render the whole line with reverse style
			line := status + " " + path
			// Pad to full width
			for len(line) < fl.width {
				line += " "
			}
			tb.Styled(line, lineStyle)
		} else {
			// Render status with color, path with normal style
			tb.Styled(status, statusStyle)
			tb.Write(" ")
			tb.Styled(path, lineStyle)
		}

		// Add click interaction
		lineRect := cellbuf.Rect(box.R.Min.X, y, fl.width, 1)
		dl.AddInteraction(lineRect, FileSelectedMsg{Index: i}, render.InteractionClick, 0)

		tb.Done()
		y++
	}

	// Fill remaining space with empty lines
	emptyStyle := lipgloss.NewStyle()
	for y < box.R.Max.Y {
		dl.AddFill(cellbuf.Rect(box.R.Min.X, y, fl.width, 1), ' ', emptyStyle, 0)
		y++
	}
}

// FileSelectedMsg is sent when a file is clicked in the file list
type FileSelectedMsg struct {
	Index int
}
