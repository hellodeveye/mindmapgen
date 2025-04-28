package api

import (
	"io"
	"log"
	"net/http"

	"github.com/hellodeveye/mindmapgen/internal/drawer"
	"github.com/hellodeveye/mindmapgen/internal/parser"
)

func GenerateMindmapHandler(w http.ResponseWriter, r *http.Request) {
	var content string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	content = string(body)

	root, err := parser.Parse(content)
	if err != nil {
		log.Fatalf("Failed to parse input file '%s': %v", content, err)
	}

	// 设置响应头，返回图像
	w.Header().Set("Content-Type", "image/png")
	// 生成思维导图
	err = drawer.Draw(root, w)
	if err != nil {
		log.Println("Error generating mindmap:", err)
		http.Error(w, "Failed to generate mindmap", http.StatusInternalServerError)
		return
	}
}
