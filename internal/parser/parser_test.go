package parser

import (
	"testing"
)

func TestSimpleParse(t *testing.T) {
	input := `
mindmap
  root((Test Root))
    Child1
      - SubChild1
    Child2
`
	root, err := Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if root.Text != "Test Root" {
		t.Errorf("expected root 'Test Root', got '%s'", root.Text)
	}
	if len(root.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(root.Children))
	}
}
