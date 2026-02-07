package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	protocol "github.com/mark3labs/mcp-go/mcp"
)

func callTool(t *testing.T, handler func(context.Context, protocol.CallToolRequest) (*protocol.CallToolResult, error), args map[string]any) *protocol.CallToolResult {
	t.Helper()
	req := protocol.CallToolRequest{
		Params: protocol.CallToolParams{
			Name:      ToolGenerateMindmap,
			Arguments: args,
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	return result
}

func resultText(result *protocol.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(protocol.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func hasImageContent(result *protocol.CallToolResult) bool {
	for _, c := range result.Content {
		if _, ok := c.(protocol.ImageContent); ok {
			return true
		}
	}
	return false
}

func TestGenerateMindmap_MissingContent(t *testing.T) {
	handler := generateMindmapHandler([]string{"default"})
	result := callTool(t, handler, map[string]any{})
	if !result.IsError {
		t.Fatal("expected error result for missing content")
	}
	if !strings.Contains(resultText(result), "content") {
		t.Errorf("error message should mention 'content', got: %s", resultText(result))
	}
}

func TestGenerateMindmap_EmptyContent(t *testing.T) {
	handler := generateMindmapHandler([]string{"default"})
	result := callTool(t, handler, map[string]any{"content": "   "})
	if !result.IsError {
		t.Fatal("expected error result for empty content")
	}
	if !strings.Contains(resultText(result), "non-empty") {
		t.Errorf("error message should mention 'non-empty', got: %s", resultText(result))
	}
}

func TestGenerateMindmap_OversizedContent(t *testing.T) {
	handler := generateMindmapHandler([]string{"default"})
	big := strings.Repeat("a", maxContentSize+1)
	result := callTool(t, handler, map[string]any{"content": big})
	if !result.IsError {
		t.Fatal("expected error result for oversized content")
	}
	if !strings.Contains(resultText(result), "maximum size") {
		t.Errorf("error message should mention 'maximum size', got: %s", resultText(result))
	}
}

func TestGenerateMindmap_InvalidTheme(t *testing.T) {
	handler := generateMindmapHandler([]string{"default", "dark"})
	result := callTool(t, handler, map[string]any{"content": "Root\n  Child", "theme": "nonexistent"})
	if !result.IsError {
		t.Fatal("expected error result for invalid theme")
	}
	if !strings.Contains(resultText(result), "unknown theme") {
		t.Errorf("error message should mention 'unknown theme', got: %s", resultText(result))
	}
}

func TestGenerateMindmap_InvalidLayout(t *testing.T) {
	handler := generateMindmapHandler(nil)
	result := callTool(t, handler, map[string]any{"content": "Root\n  Child", "layout": "diagonal"})
	if !result.IsError {
		t.Fatal("expected error result for invalid layout")
	}
	if !strings.Contains(resultText(result), "invalid layout") {
		t.Errorf("error message should mention 'invalid layout', got: %s", resultText(result))
	}
}

func TestGenerateMindmap_ValidInput_Base64Fallback(t *testing.T) {
	// Without R2 configured, the handler should return a base64 image.
	handler := generateMindmapHandler(nil)
	result := callTool(t, handler, map[string]any{"content": "Root\n  Child"})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(result))
	}
	if !hasImageContent(result) {
		t.Fatal("expected ImageContent in result")
	}
	for _, c := range result.Content {
		if img, ok := c.(protocol.ImageContent); ok {
			if img.MIMEType != "image/png" {
				t.Errorf("expected image/png MIME type, got: %s", img.MIMEType)
			}
			if img.Data == "" {
				t.Error("expected non-empty base64 data")
			}
		}
	}
}

func TestGenerateMindmap_NilArgs(t *testing.T) {
	handler := generateMindmapHandler(nil)
	req := protocol.CallToolRequest{
		Params: protocol.CallToolParams{
			Name: ToolGenerateMindmap,
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for nil args")
	}
}

func TestThemesResource(t *testing.T) {
	req := protocol.ReadResourceRequest{
		Params: protocol.ReadResourceParams{
			URI: themesResourceURI,
		},
	}
	contents, err := themesResourceHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("themes resource handler error: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 resource content, got %d", len(contents))
	}
	tc, ok := contents[0].(protocol.TextResourceContents)
	if !ok {
		t.Fatal("expected TextResourceContents")
	}

	var payload struct {
		Themes []string `json:"themes"`
	}
	if err := json.Unmarshal([]byte(tc.Text), &payload); err != nil {
		t.Fatalf("failed to parse themes JSON: %v", err)
	}
	if len(payload.Themes) == 0 {
		t.Error("expected at least one theme")
	}
}

func TestThemeDetailResource_Valid(t *testing.T) {
	req := protocol.ReadResourceRequest{
		Params: protocol.ReadResourceParams{
			URI: "mindmapgen://themes/default",
		},
	}
	contents, err := themeDetailHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("theme detail handler error: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 resource content, got %d", len(contents))
	}
	tc, ok := contents[0].(protocol.TextResourceContents)
	if !ok {
		t.Fatal("expected TextResourceContents")
	}
	if !strings.Contains(tc.Text, "Name") {
		t.Error("expected theme config JSON to contain 'Name'")
	}
}

func TestThemeDetailResource_NotFound(t *testing.T) {
	req := protocol.ReadResourceRequest{
		Params: protocol.ReadResourceParams{
			URI: "mindmapgen://themes/nonexistent_theme_xyz",
		},
	}
	_, err := themeDetailHandler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent theme")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}
