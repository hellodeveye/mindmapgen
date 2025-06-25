package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path"

	"github.com/hellodeveye/mindmapgen/pkg/api"
)

// NewServer creates and configures a new HTTP server multiplexer.
func NewServer(staticFS embed.FS) http.Handler {
	mux := http.NewServeMux()

	// Create a sub-filesystem rooted at "static"
	contentStatic, err := fs.Sub(staticFS, "static")
	if err != nil {
		// We can't recover from this during setup, so panic is acceptable
		// or log.Fatal if preferred.
		log.Fatalf("failed to create sub FS for static content: %v", err)
	}

	staticHandler := http.FileServer(http.FS(contentStatic))

	// API endpoints
	mux.HandleFunc("/api/gen", api.GenerateMindmapHandler)
	mux.HandleFunc("/api/themes", api.ListThemesHandler)

	mux.HandleFunc("/", handleIndex(contentStatic, staticHandler))
	return mux
}

func handleIndex(contentStatic fs.FS, staticHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for the root path
		if r.URL.Path == "/" {
			indexPath := path.Join("index.html") // Path within the embedded FS
			indexContent, err := fs.ReadFile(contentStatic, indexPath)
			if err != nil {
				log.Printf("Error reading index.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexContent)
			return
		}
		staticHandler.ServeHTTP(w, r)
	}
}
