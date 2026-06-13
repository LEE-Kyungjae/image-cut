package api

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"imagecut/internal/imageproc"
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

func TestHandleCutAcceptsCropRects(t *testing.T) {
	body, contentType := multipartBodyWithFields(t, 120, 120, map[string]string{
		"crop_rects": `[{"row":0,"col":0,"x":10,"y":10,"w":25,"h":30}]`,
	})
	req := httptest.NewRequest(http.MethodPost, "/cut", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handleCut(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("zip entries = %d, want 1", len(reader.File))
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

func TestCutImageUsesCropRects(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))

	cuts, err := cutImage(img, imageprocOptions(3, 3), `[{"row":0,"col":0,"x":10,"y":20,"w":30,"h":40}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cuts) != 1 {
		t.Fatalf("len(cuts) = %d, want 1", len(cuts))
	}
	if got := cuts[0].Image.Bounds().Dx(); got != 30 {
		t.Fatalf("width = %d, want 30", got)
	}
	if got := cuts[0].Image.Bounds().Dy(); got != 40 {
		t.Fatalf("height = %d, want 40", got)
	}
}

func TestCutImageRejectsBadCropRects(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	if _, err := cutImage(img, imageprocOptions(3, 3), `not json`); err == nil {
		t.Fatal("expected JSON error")
	}
}

func multipartBody(t *testing.T, width, height int) (*bytes.Buffer, string) {
	return multipartBodyWithFields(t, width, height, nil)
}

func multipartBodyWithFields(t *testing.T, width, height int, extra map[string]string) (*bytes.Buffer, string) {
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
	for key, value := range extra {
		fields[key] = value
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

func imageprocOptions(rows, cols int) imageproc.GridOptions {
	return imageproc.GridOptions{
		Rows: rows,
		Cols: cols,
	}
}
