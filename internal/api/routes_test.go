package api

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}

func TestHandleCutReturnsZip(t *testing.T) {
	body, contentType := multipartBody(t, 120, 120)
	req := httptest.NewRequest(http.MethodPost, "/cut", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handleCut(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("content-type = %q, want application/zip", got)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected zip body")
	}
}

func TestHandleCutRejectsMissingImage(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("rows", "2"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/cut", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handleCut(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want rendered page status", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("자를 이미지를 선택하세요.")) {
		t.Fatal("expected missing image error")
	}
}

func multipartBody(t *testing.T, width, height int) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fw, err := writer.CreateFormFile("image", "grid.png")
	if err != nil {
		t.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	if err := png.Encode(fw, img); err != nil {
		t.Fatal(err)
	}

	fields := map[string]string{
		"rows":   "3",
		"cols":   "3",
		"margin": "0",
		"gutter": "0",
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return &body, writer.FormDataContentType()
}
