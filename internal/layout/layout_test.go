package layout

import (
	"testing"

	"github.com/hellodeveye/mindmapgen/pkg/types"
)

func TestLayoutSimple(t *testing.T) {
	root := &types.Node{
		Text: "Root",
		Children: []*types.Node{
			{Text: "Child1"},
			{Text: "Child2"},
		},
	}
	Layout(root)

	if root.Children[0].X <= root.X {
		t.Errorf("child1 should be placed to the right of root")
	}
}
