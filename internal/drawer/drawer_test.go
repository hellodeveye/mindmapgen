package drawer

import (
	"bufio"
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

	const fileName = "test_output.png"
	f, err := os.Create(fileName)
	if err != nil {
		t.Fatalf("failed to create test output file: %v", err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	err = Draw(root, w)
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Fatalf("output file not created")
	}
	os.Remove(fileName)
}
