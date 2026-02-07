package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
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

	maxContentSize    = 1 << 20 // 1 MiB
	maxConcurrentDraw = 3
)

var (
	r2Once      sync.Once
	r2Client    *storage.R2Client
	r2ClientErr error

	validLayouts = map[string]bool{"right": true, "left": true, "both": true}

	renderSem = make(chan struct{}, maxConcurrentDraw)
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
		log.Printf("mindmap MCP server storage init: %v (will use base64 fallback)", r2ClientErr)
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

	srv.AddTool(buildGenerateTool(themeNames), generateMindmapHandler(themeNames))
	srv.AddResource(buildThemesResource(), themesResourceHandler)
	srv.AddResourceTemplate(buildThemeDetailTemplate(), themeDetailHandler)

	return srv
}

func buildGenerateTool(themeNames []string) protocol.Tool {
	description := "Generates a PNG mind map image from indented text or Mermaid mindmap syntax. The tool parses the provided text, converts it into a visual mind map, and returns the generated PNG image."
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
			protocol.Description("Mind map definition in indented text or Mermaid mindmap format."),
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

	opts = append(opts, protocol.WithString(
		"layout",
		protocol.Description("Layout direction. Defaults to 'right'."),
		protocol.Enum("right", "left", "both"),
		protocol.DefaultString("right"),
	))

	return protocol.NewTool(ToolGenerateMindmap, opts...)
}

func generateMindmapHandler(themeNames []string) sdk.ToolHandlerFunc {
	themeSet := make(map[string]bool, len(themeNames))
	for _, t := range themeNames {
		themeSet[t] = true
	}

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

		if len(content) > maxContentSize {
			return protocol.NewToolResultError(fmt.Sprintf("content exceeds maximum size of %d bytes", maxContentSize)), nil
		}

		themeName := "default"
		if rawTheme, ok := args["theme"]; ok {
			if value, ok := rawTheme.(string); ok && strings.TrimSpace(value) != "" {
				themeName = value
			}
		}
		if len(themeSet) > 0 && !themeSet[themeName] {
			return protocol.NewToolResultError(fmt.Sprintf("unknown theme %q; available: %s", themeName, strings.Join(themeNames, ", "))), nil
		}

		layout := "right"
		if rawLayout, ok := args["layout"]; ok {
			if value, ok := rawLayout.(string); ok && strings.TrimSpace(value) != "" {
				layout = value
			}
		}
		if !validLayouts[layout] {
			return protocol.NewToolResultError(fmt.Sprintf("invalid layout %q; must be one of: right, left, both", layout)), nil
		}

		root, err := parser.Parse(content)
		if err != nil {
			return protocol.NewToolResultErrorFromErr("failed to parse mind map outline", err), nil
		}

		// Acquire render semaphore to limit concurrency.
		select {
		case renderSem <- struct{}{}:
		case <-ctx.Done():
			return protocol.NewToolResultError("request cancelled while waiting for render slot"), nil
		}
		defer func() { <-renderSem }()

		var buffer bytes.Buffer
		if err := drawer.Draw(root, &buffer, drawer.WithTheme(themeName), drawer.WithLayout(layout)); err != nil {
			return protocol.NewToolResultErrorFromErr("failed to render mind map", err), nil
		}

		imgBytes := buffer.Bytes()
		b64Data := base64.StdEncoding.EncodeToString(imgBytes)

		// Try R2 upload; fall back to base64-only on failure.
		initR2()
		if r2Client != nil {
			url, err := r2Client.UploadImage(ctx, imgBytes, "image/png")
			if err != nil {
				log.Printf("R2 upload failed, falling back to base64: %v", err)
			} else {
				// Return both URL text and embedded image for maximum compatibility.
				return &protocol.CallToolResult{
					Content: []protocol.Content{
						protocol.TextContent{
							Annotated: protocol.Annotated{},
							Type:      "text",
							Text:      fmt.Sprintf("Mind map uploaded: %s", url),
						},
						protocol.ImageContent{
							Annotated: protocol.Annotated{},
							Type:      "image",
							Data:      b64Data,
							MIMEType:  "image/png",
						},
					},
				}, nil
			}
		}

		// No R2 or upload failed: return base64 image only.
		return &protocol.CallToolResult{
			Content: []protocol.Content{
				protocol.ImageContent{
					Annotated: protocol.Annotated{},
					Type:      "image",
					Data:      b64Data,
					MIMEType:  "image/png",
				},
			},
		}, nil
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

func buildThemeDetailTemplate() protocol.ResourceTemplate {
	return protocol.NewResourceTemplate(
		"mindmapgen://themes/{name}",
		"Theme Detail",
		protocol.WithTemplateDescription("Returns the full configuration for a specific theme."),
		protocol.WithTemplateMIMEType("application/json"),
	)
}

func themeDetailHandler(ctx context.Context, request protocol.ReadResourceRequest) ([]protocol.ResourceContents, error) {
	uri := request.Params.URI
	// Extract theme name from URI: "mindmapgen://themes/{name}"
	const prefix = "mindmapgen://themes/"
	if !strings.HasPrefix(uri, prefix) {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}
	name := strings.TrimPrefix(uri, prefix)
	if name == "" {
		return nil, fmt.Errorf("theme name is required")
	}

	mgr := theme.GetManager()
	themeNames := mgr.ListThemes()
	found := false
	for _, t := range themeNames {
		if t == name {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("theme %q not found", name)
	}

	cfg, err := mgr.GetTheme(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load theme %q: %w", name, err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize theme %q: %w", name, err)
	}

	return []protocol.ResourceContents{
		protocol.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
