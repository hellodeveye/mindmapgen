package main

import (
	"embed"
	// "io/fs" // No longer needed here
	"log"
	"net/http"

	// "path" // No longer needed here

	// "github.com/hellodeveye/mindmapgen/pkg/api" // Handled by server pkg
	"github.com/hellodeveye/mindmapgen/pkg/server" // Import the new server package
)

//go:embed all:static
var staticFiles embed.FS

func main() {
	// Create the server mux with all handlers configured
	handler := server.NewServer(staticFiles)

	log.Println("Starting server on :8080")
	// Use the handler returned by NewServer
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err) // Slightly improved error logging
	}
}
