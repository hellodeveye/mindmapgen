package drawer

import (
	"os"
	"testing"

	"github.com/hellodeveye/mindmapgen/pkg/types"
)

func TestDrawSimple(t *testing.T) {
	root := &types.Node{
		Text: "Root",
		Children: []*types.Node{
			{Text: "Child1"},
			{Text: "Child2"},
		},
	}

	err := Draw(root, "test_output.png")
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if _, err := os.Stat("test_output.png"); os.IsNotExist(err) {
		t.Fatalf("output file not created")
	}
}
