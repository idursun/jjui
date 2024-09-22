package dag

import (
	"jjui/internal/jj"
	"sort"
)

const (
	DirectEdge   = 1
	IndirectEdge = 2
)

type Dag struct {
	lookup map[string]*Node
	Nodes  []*Node
}

type Node struct {
	Parents []*Node
	Commit  *jj.Commit
	Edges   []*Edge
}

type Edge struct {
	To   *Node
	Type int
}

func NewDag() *Dag {
	return &Dag{
		lookup: make(map[string]*Node),
		Nodes:  make([]*Node, 0),
	}
}

func Build(commits []jj.Commit) *Node {
	tree := NewDag()
	for _, commit := range commits {
		tree.AddNode(&commit)
	}

	for _, commit := range commits {
		node := tree.GetNode(&commit)
		for _, parent := range commit.Parents {
			if parent := tree.GetNodeByChangeId(parent); parent != nil {
				parent.AddEdge(node, DirectEdge)
			}
		}
	}

	roots := tree.GetRoots()
	for i := 0; i < len(roots)-1; i++ {
		next := roots[i+1]
		next.AddEdge(roots[i], IndirectEdge)
	}
	return roots[len(roots)-1]
}

func (d *Dag) AddNode(c *jj.Commit) (node *Node) {
	node = &Node{
		Commit: c,
		Edges:  make([]*Edge, 0),
	}
	d.Nodes = append(d.Nodes, node)
	d.lookup[c.ChangeId] = node
	return node
}

func (n *Node) AddEdge(other *Node, typ int) {
	e := &Edge{
		To:   other,
		Type: typ,
	}
	other.Parents = append(other.Parents, n)
	n.Edges = append(n.Edges, e)
}

func (d *Dag) GetNode(c *jj.Commit) *Node {
	return d.GetNodeByChangeId(c.ChangeId)
}

func (d *Dag) GetNodeByChangeId(changeId string) *Node {
	return d.lookup[changeId]
}

func (d *Dag) GetRoots() []*Node {
	roots := make([]*Node, 0)
	for _, node := range d.Nodes {
		if node.Parents == nil {
			roots = append(roots, node)
		}
	}
	return roots
}

func Walk(node *Node, renderer Renderer, context RenderContext) {
	sort.Slice(node.Edges, func(a, b int) bool {
		f := node.Edges[a]
		s := node.Edges[b]
		if f.Type ==  s.Type {
			return f.To.Commit.Index < s.To.Commit.Index
		}
		if s.Type == DirectEdge {
			return true
		}
		return true
	})
	for i, edge := range node.Edges {
		nl := context.Level + 1
		if i == 0 {
			nl = context.Level
		}
		Walk(edge.To, renderer, RenderContext{
			Level:        nl,
			Elided:       edge.Type == IndirectEdge,
			IsFirstChild: i == 0,
		})
	}
	renderer(node, context)
}
