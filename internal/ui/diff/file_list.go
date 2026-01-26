package diff

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// treeNode represents a node in the file tree
type treeNode struct {
	name      string      // display name (e.g. "util" or "src/ui" if collapsed)
	isDir     bool
	children  []*treeNode // dirs first, then files, alphabetically
	file      *DiffFile   // non-nil only for file leaf nodes
	fileIndex int         // index into original files slice; -1 for dirs
	expanded  bool        // toggle state for directories
}

// visibleItem represents a flattened tree node with its depth
type visibleItem struct {
	node  *treeNode
	depth int
}

// FileList represents the file list panel in the diff viewer
type FileList struct {
	files         []*DiffFile
	root          *treeNode
	visible       []visibleItem
	selectedIndex int // index into visible slice
	startLine     int
	height        int
	width         int
}

// NewFileList creates a new file list from parsed diff files
func NewFileList(files []*DiffFile) *FileList {
	fl := &FileList{
		files:         files,
		selectedIndex: 0,
		startLine:     0,
	}
	fl.root = buildTree(files)
	fl.flatten()
	// Move selection to the first file node (skip any leading dirs)
	fl.selectedIndex = fl.nextFileIndex(-1)
	if fl.selectedIndex < 0 {
		fl.selectedIndex = 0
	}
	return fl
}

// buildTree constructs a tree from a slice of DiffFiles
func buildTree(files []*DiffFile) *treeNode {
	root := &treeNode{
		name:      "",
		isDir:     true,
		fileIndex: -1,
		expanded:  true,
	}

	for i, file := range files {
		path := file.Path()
		parts := strings.Split(path, "/")

		current := root
		for j, part := range parts {
			if j == len(parts)-1 {
				// File leaf node
				current.children = append(current.children, &treeNode{
					name:      part,
					isDir:     false,
					file:      file,
					fileIndex: i,
				})
			} else {
				// Directory node - find or create
				found := false
				for _, child := range current.children {
					if child.isDir && child.name == part {
						current = child
						found = true
						break
					}
				}
				if !found {
					dir := &treeNode{
						name:      part,
						isDir:     true,
						fileIndex: -1,
						expanded:  true,
					}
					current.children = append(current.children, dir)
					current = dir
				}
			}
		}
	}

	sortTree(root)
	collapseSingleChildDirs(root)
	return root
}

// sortTree sorts children: directories first, then files, alphabetically within each group
func sortTree(node *treeNode) {
	if !node.isDir || len(node.children) == 0 {
		return
	}

	sort.SliceStable(node.children, func(i, j int) bool {
		a, b := node.children[i], node.children[j]
		if a.isDir != b.isDir {
			return a.isDir // dirs first
		}
		return a.name < b.name
	})

	for _, child := range node.children {
		sortTree(child)
	}
}

// collapseSingleChildDirs merges single-child directory chains
// e.g., src/ -> ui/ becomes src/ui/
func collapseSingleChildDirs(node *treeNode) {
	if !node.isDir {
		return
	}

	for i, child := range node.children {
		if child.isDir {
			// Collapse chain of single-child dirs
			for len(child.children) == 1 && child.children[0].isDir {
				grandchild := child.children[0]
				child.name = child.name + "/" + grandchild.name
				child.children = grandchild.children
			}
			node.children[i] = child
			collapseSingleChildDirs(child)
		}
	}
}

// flatten walks the tree depth-first, populating the visible slice
func (fl *FileList) flatten() {
	fl.visible = nil
	if fl.root == nil {
		return
	}
	fl.flattenNode(fl.root, -1) // root itself is not shown
}

func (fl *FileList) flattenNode(node *treeNode, depth int) {
	if depth >= 0 {
		fl.visible = append(fl.visible, visibleItem{node: node, depth: depth})
	}

	if node.isDir && (depth < 0 || node.expanded) {
		for _, child := range node.children {
			fl.flattenNode(child, depth+1)
		}
	}
}

// SelectedFile returns the currently selected file
func (fl *FileList) SelectedFile() *DiffFile {
	idx := fl.SelectedIndex()
	if idx >= 0 && idx < len(fl.files) {
		return fl.files[idx]
	}
	return nil
}

// SelectedIndex returns the file index of the currently selected item, or -1 if a directory is selected
func (fl *FileList) SelectedIndex() int {
	if fl.selectedIndex >= 0 && fl.selectedIndex < len(fl.visible) {
		return fl.visible[fl.selectedIndex].node.fileIndex
	}
	return -1
}

// SetSelectedIndex sets the selection to the visible item matching the given file index
func (fl *FileList) SetSelectedIndex(fileIdx int) {
	if fileIdx < 0 {
		fileIdx = 0
	}
	if fileIdx >= len(fl.files) {
		fileIdx = len(fl.files) - 1
	}
	if fileIdx < 0 {
		return
	}
	// Find the visible item with this file index
	for i, item := range fl.visible {
		if item.node.fileIndex == fileIdx {
			fl.selectedIndex = i
			fl.ensureVisible()
			return
		}
	}
	// If not found (collapsed parent), expand the path and retry
	fl.expandPathToFile(fileIdx)
	fl.flatten()
	for i, item := range fl.visible {
		if item.node.fileIndex == fileIdx {
			fl.selectedIndex = i
			fl.ensureVisible()
			return
		}
	}
}

// expandPathToFile ensures all directories leading to the given file index are expanded
func (fl *FileList) expandPathToFile(fileIdx int) {
	if fl.root == nil || fileIdx < 0 || fileIdx >= len(fl.files) {
		return
	}
	fl.expandPathInNode(fl.root, fileIdx)
}

func (fl *FileList) expandPathInNode(node *treeNode, fileIdx int) bool {
	if !node.isDir {
		return node.fileIndex == fileIdx
	}
	for _, child := range node.children {
		if fl.expandPathInNode(child, fileIdx) {
			node.expanded = true
			return true
		}
	}
	return false
}

// MoveUp moves selection to the previous file node (skipping directories)
func (fl *FileList) MoveUp() {
	prev := fl.prevFileIndex(fl.selectedIndex)
	if prev >= 0 {
		fl.selectedIndex = prev
		fl.ensureVisible()
	}
}

// MoveDown moves selection to the next file node (skipping directories)
func (fl *FileList) MoveDown() {
	next := fl.nextFileIndex(fl.selectedIndex)
	if next >= 0 {
		fl.selectedIndex = next
		fl.ensureVisible()
	}
}

// nextFileIndex returns the index of the next file node after the given visible index, or -1
func (fl *FileList) nextFileIndex(from int) int {
	for i := from + 1; i < len(fl.visible); i++ {
		if !fl.visible[i].node.isDir {
			return i
		}
	}
	return -1
}

// prevFileIndex returns the index of the previous file node before the given visible index, or -1
func (fl *FileList) prevFileIndex(from int) int {
	for i := from - 1; i >= 0; i-- {
		if !fl.visible[i].node.isDir {
			return i
		}
	}
	return -1
}

// ToggleExpand toggles the expanded state of a directory at the given visible index
func (fl *FileList) ToggleExpand(visibleIdx int) {
	if visibleIdx < 0 || visibleIdx >= len(fl.visible) {
		return
	}
	node := fl.visible[visibleIdx].node
	if !node.isDir {
		return
	}

	// Remember current file selection
	currentFileIdx := fl.SelectedIndex()

	node.expanded = !node.expanded
	fl.flatten()

	// Restore file selection
	if currentFileIdx >= 0 {
		for i, item := range fl.visible {
			if item.node.fileIndex == currentFileIdx {
				fl.selectedIndex = i
				fl.ensureVisible()
				return
			}
		}
	}
	// If the previously selected file is now hidden, find nearest file
	fl.selectedIndex = fl.nearestFileIndex(visibleIdx)
	fl.ensureVisible()
}

// nearestFileIndex finds the nearest file node to the given visible index
func (fl *FileList) nearestFileIndex(from int) int {
	if from >= len(fl.visible) {
		from = len(fl.visible) - 1
	}
	// Try forward first
	for i := from; i < len(fl.visible); i++ {
		if !fl.visible[i].node.isDir {
			return i
		}
	}
	// Try backward
	for i := from - 1; i >= 0; i-- {
		if !fl.visible[i].node.isDir {
			return i
		}
	}
	return 0
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

	// Using jj's default colors for file status
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))    // Green (diff added)
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // Red (diff removed)
	modifiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // Cyan (diff modified)
	renamedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // Cyan (diff renamed)
	copiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // Green (diff copied)
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))      // Blue

	// Render visible items
	y := box.R.Min.Y
	for i := fl.startLine; i < len(fl.visible) && y < box.R.Max.Y; i++ {
		item := fl.visible[i]
		node := item.node
		isSelected := i == fl.selectedIndex

		indent := strings.Repeat("  ", item.depth)

		if node.isDir {
			// Directory node
			arrow := "▾ "
			if !node.expanded {
				arrow = "▸ "
			}
			label := indent + arrow + node.name + "/"

			// Truncate if too long
			if len(label) > fl.width {
				label = label[:fl.width]
			}

			tb := dl.Text(box.R.Min.X, y, 0)
			if isSelected {
				line := label
				for len(line) < fl.width {
					line += " "
				}
				tb.Styled(line, selectedStyle)
			} else {
				tb.Styled(label, dirStyle)
			}

			lineRect := cellbuf.Rect(box.R.Min.X, y, fl.width, 1)
			dl.AddInteraction(lineRect, TreeToggleMsg{VisibleIndex: i}, render.InteractionClick, 0)

			tb.Done()
		} else {
			// File node
			file := node.file

			var fileStyle lipgloss.Style
			switch file.Status {
			case FileAdded:
				fileStyle = addedStyle
			case FileDeleted:
				fileStyle = deletedStyle
			case FileRenamed:
				fileStyle = renamedStyle
			case FileCopied:
				fileStyle = copiedStyle
			case FileModified:
				fileStyle = modifiedStyle
			default:
				fileStyle = normalStyle
			}

			fileName := node.name

			// Truncate if too long
			maxLen := fl.width - len(indent)
			if len(fileName) > maxLen && maxLen > 3 {
				fileName = "..." + fileName[len(fileName)-maxLen+3:]
			}

			tb := dl.Text(box.R.Min.X, y, 0)
			if isSelected {
				line := indent + fileName
				for len(line) < fl.width {
					line += " "
				}
				tb.Styled(line, selectedStyle)
			} else {
				if len(indent) > 0 {
					tb.Write(indent)
				}
				tb.Styled(fileName, fileStyle)
			}

			lineRect := cellbuf.Rect(box.R.Min.X, y, fl.width, 1)
			dl.AddInteraction(lineRect, FileSelectedMsg{Index: node.fileIndex}, render.InteractionClick, 0)

			tb.Done()
		}

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

// TreeToggleMsg is sent when a directory is clicked in the file list
type TreeToggleMsg struct {
	VisibleIndex int
}
