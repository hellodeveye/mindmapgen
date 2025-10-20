package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/hellodeveye/mindmapgen/internal/drawer"
	"github.com/hellodeveye/mindmapgen/internal/parser"
	"github.com/hellodeveye/mindmapgen/internal/storage"
	"github.com/hellodeveye/mindmapgen/internal/theme"
	protocol "github.com/mark3labs/mcp-go/mcp"
	sdk "github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "Mindmap Generator"
	serverVersion = "0.1.0"
	// ToolGenerateMindmap is the identifier MCP clients should call to render a mind map.
	ToolGenerateMindmap = "generate_mindmap"
	themesResourceURI   = "mindmapgen://themes"
)

var (
	r2Once      sync.Once
	r2Client    *storage.R2Client
	r2ClientErr error
)

func initR2() {
	r2Once.Do(func() {
		var err error
		r2Client, err = storage.NewR2ClientFromEnv()
		if err != nil {
			if errors.Is(err, storage.ErrMissingR2Config) {
				r2ClientErr = fmt.Errorf("missing R2 storage configuration; ensure R2_* environment variables are set")
			} else {
				r2ClientErr = fmt.Errorf("failed to initialize R2 client: %w", err)
			}
		}
	})
}

// NewMindmapServer constructs an MCP server instance exposing mind map tooling.
func NewMindmapServer() *sdk.MCPServer {
	initR2()
	if r2ClientErr != nil {
		log.Printf("mindmap MCP server storage init failed: %v", r2ClientErr)
	}

	themeNames := theme.GetManager().ListThemes()
	sort.Strings(themeNames)

	srv := sdk.NewMCPServer(
		serverName,
		serverVersion,
		sdk.WithToolCapabilities(true),
		sdk.WithResourceCapabilities(false, false),
		sdk.WithLogging(),
		sdk.WithRecovery(),
		sdk.WithInstructions("Expose tools for turning outline text into rendered mind map PNGs."),
	)

	srv.AddTool(buildGenerateTool(themeNames), generateMindmapHandler())
	srv.AddResource(buildThemesResource(), themesResourceHandler)

	return srv
}

func buildGenerateTool(themeNames []string) protocol.Tool {
	description := "Generate a PNG mind map from indented or Mermaid mindmap text."
	opts := []protocol.ToolOption{
		protocol.WithDescription(description),
		protocol.WithToolAnnotation(protocol.ToolAnnotation{
			Title:           "Generate Mind Map",
			ReadOnlyHint:    protocol.ToBoolPtr(true),
			DestructiveHint: protocol.ToBoolPtr(false),
			IdempotentHint:  protocol.ToBoolPtr(true),
			OpenWorldHint:   protocol.ToBoolPtr(false),
		}),
		protocol.WithString(
			"content",
			protocol.Required(),
			protocol.Description("Mind map outline; supports 'mindmap' headers or indentation."),
			protocol.MinLength(1),
		),
	}

	themeDescription := "Rendering theme. Defaults to 'default'."
	if len(themeNames) > 0 {
		opts = append(opts, protocol.WithString(
			"theme",
			protocol.Description(themeDescription+" Available: "+strings.Join(themeNames, ", ")),
			protocol.Enum(themeNames...),
			protocol.DefaultString("default"),
		))
	} else {
		opts = append(opts, protocol.WithString(
			"theme",
			protocol.Description(themeDescription),
			protocol.DefaultString("default"),
		))
	}

	return protocol.NewTool(ToolGenerateMindmap, opts...)
}

func generateMindmapHandler() sdk.ToolHandlerFunc {
	return func(ctx context.Context, request protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		args := request.GetArguments()
		if len(args) == 0 {
			return protocol.NewToolResultError("missing required argument: content"), nil
		}

		rawContent, ok := args["content"]
		if !ok {
			return protocol.NewToolResultError("missing required argument: content"), nil
		}

		content, ok := rawContent.(string)
		if !ok || strings.TrimSpace(content) == "" {
			return protocol.NewToolResultError("argument 'content' must be a non-empty string"), nil
		}

		themeName := "default"
		if rawTheme, ok := args["theme"]; ok {
			if value, ok := rawTheme.(string); ok && strings.TrimSpace(value) != "" {
				themeName = value
			}
		}

		root, err := parser.Parse(content)
		if err != nil {
			return protocol.NewToolResultErrorFromErr("failed to parse mind map outline", err), nil
		}

		var buffer bytes.Buffer
		if err := drawer.DrawWithTheme(root, &buffer, themeName); err != nil {
			return protocol.NewToolResultErrorFromErr("failed to render mind map", err), nil
		}

		initR2()
		if r2Client == nil {
			return protocol.NewToolResultErrorFromErr("object storage not configured", r2ClientErr), nil
		}

		url, err := r2Client.UploadImage(ctx, buffer.Bytes(), "image/png")
		if err != nil {
			return protocol.NewToolResultErrorFromErr("failed to upload mind map", err), nil
		}

		return protocol.NewToolResultText(fmt.Sprintf("Mind map uploaded: %s", url)), nil
	}
}

func buildThemesResource() protocol.Resource {
	return protocol.NewResource(
		themesResourceURI,
		"Available Themes",
		protocol.WithResourceDescription("Lists renderer themes that MCP clients can select."),
		protocol.WithMIMEType("application/json"),
	)
}

func themesResourceHandler(ctx context.Context, request protocol.ReadResourceRequest) ([]protocol.ResourceContents, error) {
	themeNames := theme.GetManager().ListThemes()
	sort.Strings(themeNames)

	payload := struct {
		Themes []string `json:"themes"`
	}{Themes: themeNames}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return []protocol.ResourceContents{
		protocol.TextResourceContents{
			URI:      themesResourceURI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
