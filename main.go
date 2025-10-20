package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/hellodeveye/mindmapgen/api"
	"github.com/hellodeveye/mindmapgen/internal/storage"
	"github.com/hellodeveye/mindmapgen/pkg/server"
)

//go:embed all:static
var staticFiles embed.FS

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()
	addr := fmt.Sprintf(":%d", *port)

	// Create the server mux with all handlers configured
	handler := server.NewServer(staticFiles)
	if cfg, err := storage.LoadR2ConfigFromEnv(); err != nil {
		if !errors.Is(err, storage.ErrMissingR2Config) {
			log.Printf("failed to load R2 config: %v", err)
		}
	} else if err := api.InitR2Client(cfg); err != nil {
		log.Printf("failed to initialize R2 client: %v", err)
	}

	log.Printf("Starting server on %s", addr)
	// Use the handler returned by NewServer
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err) // Slightly improved error logging
	}
}
