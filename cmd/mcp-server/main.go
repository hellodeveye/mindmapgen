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
	addr := flag.String("addr", ":8082", "address for the MCP HTTP server (e.g. :8082 or 0.0.0.0:8082)")
	basePath := flag.String("base-path", "/mcp", "path prefix for the MCP endpoint")
	keepAlive := flag.Bool("keep-alive", false, "enable periodic keep-alive heartbeat events")
	keepAliveInterval := flag.Duration("keep-alive-interval", 10*time.Second, "interval between keep-alive events when enabled")

	flag.Parse()

	mcpServer := mindmapmcp.NewMindmapServer()

	var opts []sdk.StreamableHTTPOption
	if path := strings.TrimSpace(*basePath); path != "" {
		opts = append(opts, sdk.WithEndpointPath(path))
	}
	if *keepAlive {
		opts = append(opts, sdk.WithHeartbeatInterval(*keepAliveInterval))
	}

	httpServer := sdk.NewStreamableHTTPServer(mcpServer, opts...)

	go func() {
		log.Printf("Starting MCP Streamable HTTP server on %s (endpoint=%s)", *addr, *basePath)
		if err := httpServer.Start(*addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("MCP HTTP server stopped unexpectedly: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received %s, shutting down MCP HTTP server", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
}
