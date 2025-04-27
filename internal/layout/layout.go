package layout

import (
	"github.com/hellodeveye/mindmapgen/pkg/types"
)

var nodeWidth = 120.0
var nodeHeight = 50.0
var horizontalSpacing = 30.0
var verticalSpacing = 100.0

func Layout(root *types.Node) {
	layout(root, 0, 0)
}

func layout(node *types.Node, x, y float64) float64 {
	if node == nil {
		return y
	}

	node.X = x
	node.Y = y

	if len(node.Children) == 0 {
		return y + verticalSpacing
	}

	childY := y
	for _, child := range node.Children {
		childY = layout(child, x+nodeWidth+horizontalSpacing, childY)
	}

	return childY
}
