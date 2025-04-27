package types

import "testing"

func TestNewNode(t *testing.T) {
	root := &Node{Text: "root"}

	if root.Text != "root" {
		t.Errorf("expected text 'root', got %s", root.Text)
	}
	if root.Children == nil {
		t.Errorf("expected initialized children slice")
	}
}
