package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateMindmapHandler_URLWithoutR2Client(t *testing.T) {
	prevClient := r2Client
	r2Client = nil
	t.Cleanup(func() {
		r2Client = prevClient
	})

	req := httptest.NewRequest(http.MethodPost, "/api/gen?media=url", bytes.NewBufferString("root\n  child"))
	rec := httptest.NewRecorder()

	GenerateMindmapHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "R2 client not configured") {
		t.Fatalf("expected error message to mention R2 client not configured, got %q", rec.Body.String())
	}
}

func TestGenerateMindmapHandler_EmptyInput(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/gen", bytes.NewBufferString("   \n\t"))
	rec := httptest.NewRecorder()

	GenerateMindmapHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Empty input content") {
		t.Fatalf("expected empty input error, got %q", rec.Body.String())
	}
}

func TestGenerateMindmapHandler_InputTooLarge(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxMindmapInputBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/api/gen", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()

	GenerateMindmapHandler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Input too large") {
		t.Fatalf("expected input too large error, got %q", rec.Body.String())
	}
}

func TestGenerateMindmapHandler_LayoutParam(t *testing.T) {
	tests := []struct {
		name   string
		layout string
	}{
		{name: "both", layout: "both"},
		{name: "left", layout: "left"},
		{name: "right", layout: "right"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/gen?media=raw&layout="+tt.layout, bytes.NewBufferString("root\n  child"))
			rec := httptest.NewRecorder()

			GenerateMindmapHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/png") {
				t.Fatalf("expected Content-Type image/png, got %q", got)
			}

			body := rec.Body.Bytes()
			if len(body) < 8 {
				t.Fatalf("expected PNG body, got %d bytes", len(body))
			}
			pngSig := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
			if !bytes.Equal(body[:8], pngSig) {
				t.Fatalf("response is not PNG data")
			}
		})
	}
}
