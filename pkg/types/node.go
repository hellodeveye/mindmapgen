package types

type NodeStyle struct {
	FillColor   [3]float64
	StrokeColor [3]float64
	TextColor   [3]float64
}

type Node struct {
	Text     string
	Children []*Node
	X, Y     float64
	Style    *NodeStyle // Optional custom style for this node
}

// NewNode creates a new node with default style
func NewNode(text string) *Node {
	return &Node{
		Text: text,
	}
}

// AddChild adds a child node to the current node
func (n *Node) AddChild(child *Node) {
	n.Children = append(n.Children, child)
}
