package drawer

import (
	"bufio"
	"io"
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

func TestDrawLayoutBothSides(t *testing.T) {
	root := &types.Node{
		Text: "Root",
		Children: []*types.Node{
			{Text: "Child1"},
			{Text: "Child2"},
			{Text: "Child3"},
			{Text: "Child4"},
		},
	}

	if err := Draw(root, io.Discard, WithLayout("both")); err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	var hasLeft, hasRight bool
	for _, child := range root.Children {
		if child.X < root.X {
			hasLeft = true
		}
		if child.X > root.X {
			hasRight = true
		}
	}

	if !hasLeft || !hasRight {
		t.Fatalf("expected children on both sides: left=%v right=%v", hasLeft, hasRight)
	}
}

func TestDrawLayoutDirectional(t *testing.T) {
	tests := []struct {
		name      string
		layout    string
		expectDir int
	}{
		{name: "right", layout: "right", expectDir: 1},
		{name: "left", layout: "left", expectDir: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &types.Node{
				Text: "Root",
				Children: []*types.Node{
					{Text: "Child1"},
					{Text: "Child2"},
				},
			}

			if err := Draw(root, io.Discard, WithLayout(tt.layout)); err != nil {
				t.Fatalf("draw failed: %v", err)
			}

			for _, child := range root.Children {
				if tt.expectDir > 0 && child.X <= root.X {
					t.Fatalf("expected child to be on right side, got child.X=%v root.X=%v", child.X, root.X)
				}
				if tt.expectDir < 0 && child.X >= root.X {
					t.Fatalf("expected child to be on left side, got child.X=%v root.X=%v", child.X, root.X)
				}
			}
		})
	}
}
