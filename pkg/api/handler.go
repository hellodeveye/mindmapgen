package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/hellodeveye/mindmapgen/internal/drawer"
	"github.com/hellodeveye/mindmapgen/internal/parser"
	"github.com/hellodeveye/mindmapgen/internal/storage"
)

var r2Client *storage.R2Client

func InitR2Client(cfg storage.R2Config) error {
	var err error
	r2Client, err = storage.NewR2Client(cfg)
	return err
}

func GenerateMindmapHandler(w http.ResponseWriter, r *http.Request) {
	//根据参数media决定是否上传
	media := r.URL.Query().Get("media")

	//返回原始图片
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

	switch media {
	case "raw":
		// 设置响应头，返回图像
		w.Header().Set("Content-Type", "image/png")
		// 生成思维导图
		err = drawer.Draw(root, w)
		if err != nil {
			log.Println("Error generating mindmap:", err)
			http.Error(w, "Failed to generate mindmap", http.StatusInternalServerError)
			return
		}
	case "url":
		// Generate mindmap to buffer
		var buf bytes.Buffer
		err = drawer.Draw(root, &buf)
		if err != nil {
			log.Println("Error generating mindmap:", err)
			http.Error(w, "Failed to generate mindmap", http.StatusInternalServerError)
			return
		}
		//上传图片
		url, err := r2Client.UploadImage(r.Context(), buf.Bytes(), "image/png")
		if err != nil {
			log.Println("Error uploading to R2:", err)
			http.Error(w, "Failed to upload mindmap", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			URL string `json:"url"`
		}{URL: url})
	}

}
