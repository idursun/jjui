package diff

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type diffTreeNode struct {
	name          string
	path          string
	isDir         bool
	children      []*diffTreeNode
	childrenNodes []render.TreeNode
	file          *DiffFile
	fileIndex     int
}

func (n *diffTreeNode) ID() string {
	return n.path
}

func (n *diffTreeNode) Name() string {
	return n.name
}

func (n *diffTreeNode) Children() []render.TreeNode {
	if len(n.children) == 0 {
		return nil
	}
	if n.childrenNodes == nil {
		children := make([]render.TreeNode, 0, len(n.children))
		for _, child := range n.children {
			children = append(children, child)
		}
		n.childrenNodes = children
	}
	return n.childrenNodes
}

// FileList represents the file list panel in the diff viewer.
type FileList struct {
	files     []*DiffFile
	root      *diffTreeNode
	tree      *render.TreeList
	fileNodes map[int]*diffTreeNode
}

// NewFileList creates a new file list from parsed diff files.
func NewFileList(files []*DiffFile) *FileList {
	fl := &FileList{
		files: files,
	}
	fl.root = buildTree(files)
	fl.fileNodes = make(map[int]*diffTreeNode)
	fl.indexFiles(fl.root)
	fl.tree = render.NewTreeList(render.TreeListConfig{
		Root:            fl.root,
		DefaultExpanded: true,
		Selectable: func(node render.TreeNode) bool {
			return len(node.Children()) == 0
		},
	})
	fl.tree.SetRenderNode(fl.renderNode)
	fl.tree.SetClickMessage(fl.clickMessage)
	fl.tree.SelectFirstSelectable()
	return fl
}

// buildTree constructs a tree from a slice of DiffFiles.
func buildTree(files []*DiffFile) *diffTreeNode {
	root := &diffTreeNode{
		name:      "",
		path:      "",
		isDir:     true,
		fileIndex: -1,
	}

	for i, file := range files {
		path := file.Path()
		parts := strings.Split(path, "/")

		current := root
		currentPath := ""
		for j, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			if j == len(parts)-1 {
				current.children = append(current.children, &diffTreeNode{
					name:      part,
					path:      currentPath,
					isDir:     false,
					file:      file,
					fileIndex: i,
				})
			} else {
				found := false
				for _, child := range current.children {
					if child.isDir && child.name == part {
						current = child
						found = true
						break
					}
				}
				if !found {
					dir := &diffTreeNode{
						name:      part,
						path:      currentPath,
						isDir:     true,
						fileIndex: -1,
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

// sortTree sorts children: directories first, then files, alphabetically within each group.
func sortTree(node *diffTreeNode) {
	if !node.isDir || len(node.children) == 0 {
		return
	}

	sort.SliceStable(node.children, func(i, j int) bool {
		a, b := node.children[i], node.children[j]
		if a.isDir != b.isDir {
			return a.isDir
		}
		return a.name < b.name
	})

	for _, child := range node.children {
		sortTree(child)
	}
}

// collapseSingleChildDirs merges single-child directory chains.
func collapseSingleChildDirs(node *diffTreeNode) {
	if !node.isDir {
		return
	}

	for i, child := range node.children {
		if child.isDir {
			for len(child.children) == 1 && child.children[0].isDir {
				grandchild := child.children[0]
				child.name = child.name + "/" + grandchild.name
				child.path = grandchild.path
				child.children = grandchild.children
			}
			node.children[i] = child
			collapseSingleChildDirs(child)
		}
	}
}

func (fl *FileList) indexFiles(node *diffTreeNode) {
	if node == nil {
		return
	}
	if !node.isDir && node.fileIndex >= 0 {
		fl.fileNodes[node.fileIndex] = node
		return
	}
	for _, child := range node.children {
		fl.indexFiles(child)
	}
}

// SelectedFile returns the currently selected file.
func (fl *FileList) SelectedFile() *DiffFile {
	if fl.tree == nil {
		return nil
	}
	node, ok := fl.tree.SelectedNode().(*diffTreeNode)
	if !ok || node == nil {
		return nil
	}
	return node.file
}

// SelectedIndex returns the file index of the currently selected item, or -1 if none.
func (fl *FileList) SelectedIndex() int {
	if fl.tree == nil {
		return -1
	}
	node, ok := fl.tree.SelectedNode().(*diffTreeNode)
	if !ok || node == nil || node.isDir {
		return -1
	}
	return node.fileIndex
}

// SetSelectedIndex sets the selection to the visible item matching the given file index.
func (fl *FileList) SetSelectedIndex(fileIdx int) {
	if len(fl.files) == 0 || fl.tree == nil {
		return
	}
	if fileIdx < 0 {
		fileIdx = 0
	}
	if fileIdx >= len(fl.files) {
		fileIdx = len(fl.files) - 1
	}
	node, ok := fl.fileNodes[fileIdx]
	if !ok {
		return
	}
	fl.tree.SetSelectedByID(node.ID())
}

// MoveUp moves selection to the previous file node (skipping directories).
func (fl *FileList) MoveUp() {
	if fl.tree == nil {
		return
	}
	fl.tree.MoveUp()
}

// MoveDown moves selection to the next file node (skipping directories).
func (fl *FileList) MoveDown() {
	if fl.tree == nil {
		return
	}
	fl.tree.MoveDown()
}

// ToggleExpand toggles the expanded state of a directory at the given visible index.
func (fl *FileList) ToggleExpand(visibleIdx int) {
	if fl.tree == nil {
		return
	}
	fl.tree.ToggleExpand(visibleIdx)
}

// FileCount returns the number of files.
func (fl *FileList) FileCount() int {
	return len(fl.files)
}

// ViewRect renders the file list to the display context.
func (fl *FileList) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if fl.tree == nil {
		return
	}
	fl.tree.ViewRect(dl, box)
}

func (fl *FileList) renderNode(
	dl *render.DisplayContext,
	node render.TreeNode,
	depth int,
	isSelected bool,
	rect cellbuf.Rectangle,
) {
	diffNode, ok := node.(*diffTreeNode)
	if !ok || diffNode == nil {
		return
	}

	width := rect.Dx()
	if width <= 0 {
		return
	}

	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Reverse(true)

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	modifiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	renamedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	copiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	indent := strings.Repeat("  ", depth)
	if diffNode.isDir {
		arrow := "▾ "
		if !fl.tree.IsExpanded(node) {
			arrow = "▸ "
		}
		label := indent + arrow + diffNode.name + "/"
		if len(label) > width {
			label = label[:width]
		}

		tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
		if isSelected {
			line := label
			for len(line) < width {
				line += " "
			}
			tb.Styled(line, selectedStyle)
		} else {
			tb.Styled(label, dirStyle)
		}
		tb.Done()
		return
	}

	var fileStyle lipgloss.Style
	switch diffNode.file.Status {
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

	fileName := diffNode.name
	maxLen := width - len(indent)
	if len(fileName) > maxLen && maxLen > 3 {
		fileName = "..." + fileName[len(fileName)-maxLen+3:]
	}

	tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
	if isSelected {
		line := indent + fileName
		for len(line) < width {
			line += " "
		}
		tb.Styled(line, selectedStyle)
	} else {
		if len(indent) > 0 {
			tb.Write(indent)
		}
		tb.Styled(fileName, fileStyle)
	}
	tb.Done()
}

func (fl *FileList) clickMessage(node render.TreeNode, visibleIndex int) render.ClickMessage {
	diffNode, ok := node.(*diffTreeNode)
	if !ok || diffNode == nil {
		return nil
	}
	if diffNode.isDir {
		return TreeToggleMsg{VisibleIndex: visibleIndex}
	}
	return FileSelectedMsg{Index: diffNode.fileIndex}
}

// FileSelectedMsg is sent when a file is clicked in the file list.
type FileSelectedMsg struct {
	Index int
}

// TreeToggleMsg is sent when a directory is clicked in the file list.
type TreeToggleMsg struct {
	VisibleIndex int
}
