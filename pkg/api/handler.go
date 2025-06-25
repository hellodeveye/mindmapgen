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
	"github.com/hellodeveye/mindmapgen/internal/theme"
)

var r2Client *storage.R2Client

func InitR2Client(cfg storage.R2Config) error {
	var err error
	r2Client, err = storage.NewR2Client(cfg)
	return err
}

func GenerateMindmapHandler(w http.ResponseWriter, r *http.Request) {
	// 获取参数
	media := r.URL.Query().Get("media")
	themeName := r.URL.Query().Get("theme")

	// 如果没有指定主题，使用默认主题
	if themeName == "" {
		themeName = "default"
	}

	// 读取请求内容
	var content string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	content = string(body)

	// 解析内容
	root, err := parser.Parse(content)
	if err != nil {
		log.Printf("Failed to parse input: %v", err)
		http.Error(w, "Failed to parse input content", http.StatusBadRequest)
		return
	}

	switch media {
	case "raw":
		// 设置响应头，返回图像
		w.Header().Set("Content-Type", "image/png")

		// 使用指定主题生成思维导图
		err = drawer.DrawWithTheme(root, w, themeName)
		if err != nil {
			log.Println("Error generating mindmap:", err)
			http.Error(w, "Failed to generate mindmap", http.StatusInternalServerError)
			return
		}

	case "url":
		// Generate mindmap to buffer
		var buf bytes.Buffer
		err = drawer.DrawWithTheme(root, &buf, themeName)
		if err != nil {
			log.Println("Error generating mindmap:", err)
			http.Error(w, "Failed to generate mindmap", http.StatusInternalServerError)
			return
		}

		// 上传图片
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

	default:
		// 默认返回原始图片
		w.Header().Set("Content-Type", "image/png")
		err = drawer.DrawWithTheme(root, w, themeName)
		if err != nil {
			log.Println("Error generating mindmap:", err)
			http.Error(w, "Failed to generate mindmap", http.StatusInternalServerError)
			return
		}
	}
}

// ListThemesHandler 列出所有可用主题
func ListThemesHandler(w http.ResponseWriter, r *http.Request) {
	manager := theme.GetManager()
	themes := manager.ListThemes()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Themes []string `json:"themes"`
	}{Themes: themes})
}
