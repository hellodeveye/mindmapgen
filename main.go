package main

import (
	"embed"
	"os"

	"log"
	"net/http"

	"github.com/hellodeveye/mindmapgen/api"
	"github.com/hellodeveye/mindmapgen/internal/storage"
	"github.com/hellodeveye/mindmapgen/pkg/server"
)

//go:embed all:static
var staticFiles embed.FS

func main() {
	// Create the server mux with all handlers configured
	handler := server.NewServer(staticFiles)
	api.InitR2Client(storage.R2Config{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("R2_ACCESS_KEY_SECRET"),
		BucketName:      os.Getenv("R2_BUCKET_NAME"),
		Domain:          os.Getenv("R2_DOMAIN"),
	})

	log.Println("Starting server on :8080")
	// Use the handler returned by NewServer
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err) // Slightly improved error logging
	}
}
