package api

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"imagecut/internal/imageproc"
)

const maxUploadBytes = 20 << 20
const maxImagePixels = 25_000_000

var indexTmpl = template.Must(template.ParseFiles(projectPath("web/templates/index.html")))

type pageData struct {
	Error string
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/cut", handleCut)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	renderIndex(w, "")
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func handleCut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		renderIndex(w, "20MB 이하의 이미지 파일을 업로드하세요.")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		renderIndex(w, "자를 이미지를 선택하세요.")
		return
	}
	defer file.Close()

	img, format, err := decodeImage(file)
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	opts, err := parseOptions(r)
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	cuts, err := imageproc.CutGrid(img, opts)
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, cut := range cuts {
		name := fmt.Sprintf("imagecut_r%02d_c%02d.%s", cut.Row+1, cut.Col+1, format)
		fw, err := zw.Create(name)
		if err != nil {
			renderIndex(w, "ZIP 파일을 만들 수 없습니다.")
			return
		}
		if err := encodeImage(fw, cut.Image, format); err != nil {
			renderIndex(w, "잘라낸 이미지를 인코딩할 수 없습니다.")
			return
		}
	}
	if err := zw.Close(); err != nil {
		renderIndex(w, "ZIP 파일을 마무리할 수 없습니다.")
		return
	}

	base := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	if base == "" || base == "." {
		base = "imagecut"
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_cuts.zip"`, sanitizeName(base)))
	w.Header().Set("Content-Length", strconv.Itoa(out.Len()))
	_, _ = w.Write(out.Bytes())
}

func renderIndex(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTmpl.Execute(w, pageData{Error: message}); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func parseOptions(r *http.Request) (imageproc.GridOptions, error) {
	rows, err := intField(r, "rows", 3)
	if err != nil {
		return imageproc.GridOptions{}, err
	}
	cols, err := intField(r, "cols", 3)
	if err != nil {
		return imageproc.GridOptions{}, err
	}
	margin, err := intField(r, "margin", 0)
	if err != nil {
		return imageproc.GridOptions{}, err
	}
	gutter, err := intField(r, "gutter", 0)
	if err != nil {
		return imageproc.GridOptions{}, err
	}

	return imageproc.GridOptions{
		Rows:   rows,
		Cols:   cols,
		Margin: margin,
		Gutter: gutter,
	}, nil
}

func intField(r *http.Request, name string, fallback int) (int, error) {
	value := strings.TrimSpace(r.FormValue(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s 값은 숫자여야 합니다.", name)
	}
	return parsed, nil
}

func decodeImage(r io.ReadSeeker) (image.Image, string, error) {
	config, format, err := image.DecodeConfig(r)
	if err != nil {
		return nil, "", fmt.Errorf("PNG 또는 JPEG 이미지만 지원합니다.")
	}
	switch format {
	case "png", "jpeg":
	default:
		return nil, "", fmt.Errorf("PNG 또는 JPEG 이미지만 지원합니다.")
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, "", fmt.Errorf("이미지 크기를 확인할 수 없습니다.")
	}
	if config.Width*config.Height > maxImagePixels {
		return nil, "", fmt.Errorf("이미지가 너무 큽니다. 최대 25MP까지 지원합니다.")
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, "", fmt.Errorf("이미지를 다시 읽을 수 없습니다.")
	}

	img, format, err := image.Decode(r)
	if err != nil {
		return nil, "", fmt.Errorf("PNG 또는 JPEG 이미지만 지원합니다.")
	}
	switch format {
	case "png", "jpeg":
		return img, format, nil
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

func encodeImage(w io.Writer, img image.Image, format string) error {
	switch format {
	case "png":
		return png.Encode(w, img)
	case "jpeg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 92})
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "imagecut"
	}
	return b.String()
}

func projectPath(path string) string {
	dir, err := os.Getwd()
	if err != nil {
		return path
	}

	for {
		candidate := filepath.Join(dir, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return path
		}
		dir = parent
	}
}
