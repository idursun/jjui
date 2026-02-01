package details

import (
	"sort"
	"strings"

	"github.com/idursun/jjui/internal/ui/render"
)

type detailsTreeNode struct {
	name          string
	path          string
	children      []*detailsTreeNode
	childrenNodes []render.TreeNode
	item          *item
	fileIndex     int
}

func (n *detailsTreeNode) ID() string {
	return n.path
}

func (n *detailsTreeNode) Name() string {
	return n.name
}

func (n *detailsTreeNode) Children() []render.TreeNode {
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

func buildDetailsTree(items []*item) *detailsTreeNode {
	root := &detailsTreeNode{
		name:      "",
		path:      "",
		fileIndex: -1,
	}

	for i, it := range items {
		parts := strings.Split(it.fileName, "/")
		current := root
		currentPath := ""
		for j, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			if j == len(parts)-1 {
				current.children = append(current.children, &detailsTreeNode{
					name:      part,
					path:      currentPath,
					item:      it,
					fileIndex: i,
				})
			} else {
				found := false
				for _, child := range current.children {
					if child.item == nil && child.name == part {
						current = child
						found = true
						break
					}
				}
				if !found {
					dir := &detailsTreeNode{
						name:      part,
						path:      currentPath,
						fileIndex: -1,
					}
					current.children = append(current.children, dir)
					current = dir
				}
			}
		}
	}

	sortDetailsTree(root)
	collapseDetailsSingleChildDirs(root)
	return root
}

func sortDetailsTree(node *detailsTreeNode) {
	if len(node.children) == 0 {
		return
	}

	sort.SliceStable(node.children, func(i, j int) bool {
		a, b := node.children[i], node.children[j]
		aDir := len(a.children) > 0
		bDir := len(b.children) > 0
		if aDir != bDir {
			return aDir
		}
		return a.name < b.name
	})

	for _, child := range node.children {
		sortDetailsTree(child)
	}
}

func collapseDetailsSingleChildDirs(node *detailsTreeNode) {
	for i, child := range node.children {
		if len(child.children) > 0 && child.item == nil {
			for len(child.children) == 1 && child.children[0].item == nil {
				grandchild := child.children[0]
				child.name = child.name + "/" + grandchild.name
				child.path = grandchild.path
				child.children = grandchild.children
				child.childrenNodes = nil
			}
			node.children[i] = child
			collapseDetailsSingleChildDirs(child)
		}
	}
}

func indexDetailsTreeFiles(node *detailsTreeNode, files map[string]*detailsTreeNode) {
	if node == nil {
		return
	}
	if node.item != nil {
		files[node.item.fileName] = node
		return
	}
	for _, child := range node.children {
		indexDetailsTreeFiles(child, files)
	}
}

// collectDescendantItems collects all file items under a directory node.
func collectDescendantItems(node *detailsTreeNode) []*item {
	if node == nil {
		return nil
	}
	if node.item != nil {
		return []*item{node.item}
	}
	var items []*item
	for _, child := range node.children {
		items = append(items, collectDescendantItems(child)...)
	}
	return items
}
