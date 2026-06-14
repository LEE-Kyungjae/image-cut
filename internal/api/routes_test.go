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
	"strings"
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

func TestPricing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/pricing", nil)
	rec := httptest.NewRecorder()

	handlePricing(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", got)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"model":"gpt-image-2"`)) {
		t.Fatalf("body = %s", rec.Body.String())
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

func TestHandleCutCanOutputJPEG(t *testing.T) {
	body, contentType := multipartBodyWithFields(t, 120, 120, map[string]string{
		"output_format": "jpeg",
		"jpeg_quality":  "80",
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
	if len(reader.File) != 9 {
		t.Fatalf("zip entries = %d, want 9", len(reader.File))
	}
	if got := reader.File[0].Name; !bytes.HasSuffix([]byte(got), []byte(".jpg")) {
		t.Fatalf("zip entry name = %q, want .jpg", got)
	}
}

func TestHandleCutCanBatchOutputPNGAndJPEG(t *testing.T) {
	body, contentType := multipartBodyWithFields(t, 120, 120, map[string]string{
		"batch_formats": "png,jpeg",
		"jpeg_quality":  "80",
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
	if len(reader.File) != 18 {
		t.Fatalf("zip entries = %d, want 18", len(reader.File))
	}
	entries := map[string]bool{}
	for _, file := range reader.File {
		entries[file.Name] = true
	}
	if !entries["png/imagecut_r01_c01.png"] {
		t.Fatal("expected png batch entry")
	}
	if !entries["jpeg/imagecut_r01_c01.jpg"] {
		t.Fatal("expected jpeg batch entry")
	}
}

func TestHandleCutUsesProjectNameForZipFilename(t *testing.T) {
	body, contentType := multipartBodyWithFields(t, 120, 120, map[string]string{
		"project_name": "sticker set",
	})
	req := httptest.NewRequest(http.MethodPost, "/cut", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handleCut(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="sticker_set_cuts.zip"; filename*=UTF-8''sticker_set_cuts.zip` {
		t.Fatalf("content-disposition = %q", got)
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

func TestHandleGenerateReturnsPNG(t *testing.T) {
	body := bytes.NewBufferString("prompt=robot&rows=2&cols=2&margin=24&gutter=24")
	req := httptest.NewRequest(http.MethodPost, "/generate", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handleGenerate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type = %q, want image/png", got)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatal("expected PNG body")
	}
}

func TestHandleGenerateRejectsOpenAIWhenDisabled(t *testing.T) {
	body := bytes.NewBufferString("provider=openai&prompt=robot&rows=2&cols=2&openai_confirm=ALLOW_COST")
	req := httptest.NewRequest(http.MethodPost, "/generate", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handleGenerate(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestBuildGridPrompt(t *testing.T) {
	prompt := buildGridPrompt("robot", imageproc.GridOptions{Rows: 3, Cols: 4})
	if !bytes.Contains([]byte(prompt), []byte("3x4 contact sheet")) {
		t.Fatalf("prompt = %q, want grid instruction", prompt)
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

func TestParseOutputOptions(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/cut", nil)
	req.Form = map[string][]string{
		"output_format": {"original"},
	}

	opts, err := parseOutputOptions(req, "png")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Format != "png" || opts.Ext != "png" {
		t.Fatalf("opts = %+v, want png", opts)
	}
}

func TestParseOutputOptionsRejectsBadQuality(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/cut", nil)
	req.Form = map[string][]string{
		"output_format": {"jpeg"},
		"jpeg_quality":  {"101"},
	}

	if _, err := parseOutputOptions(req, "png"); err == nil {
		t.Fatal("expected jpeg quality error")
	}
}

func TestParseExportFormatsKeepsOriginalFolder(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/cut", nil)
	req.Form = map[string][]string{
		"batch_formats": {"original", "png", "jpeg"},
	}

	outputs, err := parseExportFormats(req, "png")
	if err != nil {
		t.Fatal(err)
	}
	if len(outputs) != 3 {
		t.Fatalf("outputs = %d, want 3", len(outputs))
	}
	if outputs[0].Dir != "original" || outputs[0].Options.Format != "png" {
		t.Fatalf("original output = %+v", outputs[0])
	}
	if outputs[1].Dir != "png" || outputs[1].Options.Format != "png" {
		t.Fatalf("png output = %+v", outputs[1])
	}
	if outputs[2].Dir != "jpeg" || outputs[2].Options.Format != "jpeg" {
		t.Fatalf("jpeg output = %+v", outputs[2])
	}
}

func TestDownloadBaseName(t *testing.T) {
	tests := []struct {
		name       string
		project    string
		uploadName string
		want       string
	}{
		{name: "project name wins", project: "robot sheet", uploadName: "upload.png", want: "robot_sheet"},
		{name: "upload fallback", project: "", uploadName: "my-grid.png", want: "my-grid"},
		{name: "unicode kept", project: "스티커 3x3", uploadName: "upload.png", want: "스티커_3x3"},
		{name: "empty fallback", project: "///", uploadName: "###.png", want: "imagecut"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := downloadBaseName(tt.project, tt.uploadName); got != tt.want {
				t.Fatalf("downloadBaseName() = %q, want %q", got, tt.want)
			}
		})
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
		if key == "batch_formats" {
			continue
		}
		fields[key] = value
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if value, ok := extra["batch_formats"]; ok {
		for _, format := range strings.Split(value, ",") {
			if err := writer.WriteField("batch_formats", strings.TrimSpace(format)); err != nil {
				t.Fatal(err)
			}
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
