package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hellodeveye/mindmapgen/internal/drawer"
	"github.com/hellodeveye/mindmapgen/internal/parser"
	"github.com/hellodeveye/mindmapgen/internal/storage"
	"github.com/hellodeveye/mindmapgen/internal/theme"
)

var r2Client *storage.R2Client

const maxMindmapInputBytes = 1 << 20 // 1 MiB

type apiErrorResponse struct {
	Error string `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{Error: message})
}

func InitR2Client(cfg storage.R2Config) error {
	var err error
	r2Client, err = storage.NewR2Client(cfg)
	return err
}

func GenerateMindmapHandler(w http.ResponseWriter, r *http.Request) {
	// 获取参数
	media := r.URL.Query().Get("media")
	themeName := r.URL.Query().Get("theme")
	layout := r.URL.Query().Get("layout")

	// 如果没有指定主题，使用默认主题
	if themeName == "" {
		themeName = "default"
	}
	if layout == "" {
		layout = "right"
	}

	// 读取请求内容
	var content string
	r.Body = http.MaxBytesReader(w, r.Body, maxMindmapInputBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "Input too large")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Failed to read request body")
		return
	}
	content = string(body)
	if strings.TrimSpace(content) == "" {
		writeAPIError(w, http.StatusBadRequest, "Empty input content")
		return
	}

	// 解析内容
	root, err := parser.Parse(content)
	if err != nil {
		log.Printf("Failed to parse input: %v", err)
		writeAPIError(w, http.StatusBadRequest, "Failed to parse input content")
		return
	}

	switch media {
	case "raw":
		// 设置响应头，返回图像
		w.Header().Set("Content-Type", "image/png")

		// 使用指定主题生成思维导图
		err = drawer.Draw(root, w, drawer.WithTheme(themeName), drawer.WithLayout(layout))
		if err != nil {
			log.Println("Error generating mindmap:", err)
			writeAPIError(w, http.StatusInternalServerError, "Failed to generate mindmap")
			return
		}

	case "url":
		if r2Client == nil {
			writeAPIError(w, http.StatusServiceUnavailable, "R2 client not configured. Set R2_* environment variables and restart the server.")
			return
		}
		// Generate mindmap to buffer
		var buf bytes.Buffer
		err = drawer.Draw(root, &buf, drawer.WithTheme(themeName), drawer.WithLayout(layout))
		if err != nil {
			log.Println("Error generating mindmap:", err)
			writeAPIError(w, http.StatusInternalServerError, "Failed to generate mindmap")
			return
		}

		// 上传图片
		url, err := r2Client.UploadImage(r.Context(), buf.Bytes(), "image/png")
		if err != nil {
			log.Println("Error uploading to R2:", err)
			writeAPIError(w, http.StatusInternalServerError, "Failed to upload mindmap")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			URL string `json:"url"`
		}{URL: url})

	default:
		// 默认返回原始图片
		w.Header().Set("Content-Type", "image/png")
		err = drawer.Draw(root, w, drawer.WithTheme(themeName), drawer.WithLayout(layout))
		if err != nil {
			log.Println("Error generating mindmap:", err)
			writeAPIError(w, http.StatusInternalServerError, "Failed to generate mindmap")
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
