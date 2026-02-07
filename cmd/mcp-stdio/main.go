package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	mindmapmcp "github.com/hellodeveye/mindmapgen/pkg/mcp"
	sdk "github.com/mark3labs/mcp-go/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mcpServer := mindmapmcp.NewMindmapServer()

	stdioServer := sdk.NewStdioServer(mcpServer)
	stdioServer.SetErrorLogger(log.New(os.Stderr, "mcp-stdio: ", log.LstdFlags))

	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("stdio server error: %v", err)
	}
}
