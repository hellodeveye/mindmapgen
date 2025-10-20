package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mindmapmcp "github.com/hellodeveye/mindmapgen/pkg/mcp"
	sdk "github.com/mark3labs/mcp-go/server"
)

func main() {
	addr := flag.String("addr", ":8082", "address for the MCP SSE server (e.g. :8082 or 0.0.0.0:8082)")
	baseURL := flag.String("base-url", "", "absolute base URL used in SSE endpoint announcements (optional, e.g. https://mindmap.local)")
	basePath := flag.String("base-path", "/mcp", "path prefix for SSE and message endpoints")
	sseEndpoint := flag.String("sse-endpoint", "/sse", "relative path for the SSE stream endpoint")
	messageEndpoint := flag.String("message-endpoint", "/message", "relative path for the JSON-RPC message endpoint")
	keepAlive := flag.Bool("keep-alive", false, "enable periodic keep-alive events on the SSE stream")
	keepAliveInterval := flag.Duration("keep-alive-interval", 10*time.Second, "interval between keep-alive events when enabled")

	flag.Parse()

	mcpServer := mindmapmcp.NewMindmapServer()

	var opts []sdk.SSEOption
	if trimmed := strings.TrimSpace(*baseURL); trimmed != "" {
		opts = append(opts, sdk.WithBaseURL(strings.TrimRight(trimmed, "/")))
	}
	if path := strings.TrimSpace(*basePath); path != "" {
		opts = append(opts, sdk.WithStaticBasePath(path))
	}
	if endpoint := strings.TrimSpace(*sseEndpoint); endpoint != "" {
		opts = append(opts, sdk.WithSSEEndpoint(endpoint))
	}
	if endpoint := strings.TrimSpace(*messageEndpoint); endpoint != "" {
		opts = append(opts, sdk.WithMessageEndpoint(endpoint))
	}
	if *keepAlive {
		opts = append(opts, sdk.WithKeepAlive(true), sdk.WithKeepAliveInterval(*keepAliveInterval))
	}

	sseServer := sdk.NewSSEServer(mcpServer, opts...)

	go func() {
		log.Printf("Starting MCP SSE server on %s (stream=%s message=%s)", *addr, sseServer.CompleteSsePath(), sseServer.CompleteMessagePath())
		if err := sseServer.Start(*addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("MCP SSE server stopped unexpectedly: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received %s, shutting down MCP SSE server", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sseServer.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
}
