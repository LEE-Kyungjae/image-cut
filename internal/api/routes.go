package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"imagecut/internal/generator"
	"imagecut/internal/imageproc"
	"imagecut/internal/pricing"
)

const maxUploadBytes = 20 << 20
const maxImagePixels = 25_000_000

var indexTmpl = template.Must(template.ParseFiles(projectPath("web/templates/index.html")))

type pageData struct {
	Error string
}

type OutputOptions struct {
	Format      string
	Ext         string
	JPEGQuality int
}

type ExportFormat struct {
	Dir     string
	Options OutputOptions
}

type ExportManifest struct {
	Version     int                   `json:"version"`
	ExportedAt  string                `json:"exported_at"`
	ProjectName string                `json:"project_name,omitempty"`
	Source      ManifestSource        `json:"source"`
	Grid        imageproc.GridOptions `json:"grid"`
	Outputs     []ManifestOutput      `json:"outputs"`
	Cuts        []ManifestCut         `json:"cuts"`
}

type ManifestSource struct {
	Filename string `json:"filename"`
	Format   string `json:"format"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type ManifestOutput struct {
	Dir         string `json:"dir,omitempty"`
	Format      string `json:"format"`
	Ext         string `json:"ext"`
	JPEGQuality int    `json:"jpeg_quality,omitempty"`
}

type ManifestCut struct {
	Row int `json:"row"`
	Col int `json:"col"`
	X   int `json:"x"`
	Y   int `json:"y"`
	W   int `json:"w"`
	H   int `json:"h"`
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/cut", handleCut)
	mux.HandleFunc("/generate", handleGenerate)
	mux.HandleFunc("/pricing", handlePricing)
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

func handlePricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(pricing.Current()); err != nil {
		http.Error(w, "pricing error", http.StatusInternalServerError)
	}
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

	img, inputFormat, err := decodeImage(file)
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	opts, err := parseOptions(r)
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	outputs, err := parseExportFormats(r, inputFormat)
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	cuts, err := cutImage(img, opts, r.FormValue("crop_rects"))
	if err != nil {
		renderIndex(w, err.Error())
		return
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	if err := writeManifest(zw, buildManifest(r, header.Filename, inputFormat, img.Bounds(), opts, outputs, cuts)); err != nil {
		renderIndex(w, "manifest 파일을 만들 수 없습니다.")
		return
	}
	for _, output := range outputs {
		for _, cut := range cuts {
			name := cutFilename(output, cut)
			fw, err := zw.Create(name)
			if err != nil {
				renderIndex(w, "ZIP 파일을 만들 수 없습니다.")
				return
			}
			if err := encodeImage(fw, cut.Image, output.Options); err != nil {
				renderIndex(w, "잘라낸 이미지를 인코딩할 수 없습니다.")
				return
			}
		}
	}
	if err := zw.Close(); err != nil {
		renderIndex(w, "ZIP 파일을 마무리할 수 없습니다.")
		return
	}

	base := downloadBaseName(r.FormValue("project_name"), header.Filename)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", zipContentDisposition(base))
	w.Header().Set("Content-Length", strconv.Itoa(out.Len()))
	_, _ = w.Write(out.Bytes())
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	opts, err := parseOptions(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prompt := strings.TrimSpace(r.FormValue("prompt"))
	if prompt == "" {
		prompt = "mock grid"
	}

	provider := strings.TrimSpace(r.FormValue("provider"))
	if provider == "" {
		provider = "mock"
	}

	if provider == "openai" {
		handleOpenAIGenerate(w, r, prompt, opts)
		return
	}
	if provider != "mock" {
		http.Error(w, "지원하지 않는 생성 provider입니다.", http.StatusBadRequest)
		return
	}

	img, err := generator.MockGrid(prompt, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `inline; filename="imagecut_mock_grid.png"`)
	if err := png.Encode(w, img); err != nil {
		http.Error(w, "이미지를 생성할 수 없습니다.", http.StatusInternalServerError)
	}
}

func handleOpenAIGenerate(w http.ResponseWriter, r *http.Request, prompt string, opts imageproc.GridOptions) {
	if strings.ToLower(os.Getenv("IMAGECUT_OPENAI_ENABLED")) != "true" {
		http.Error(w, "OpenAI 생성은 기본 비활성화 상태입니다. IMAGECUT_OPENAI_ENABLED=true 설정이 필요합니다.", http.StatusForbidden)
		return
	}
	if r.FormValue("openai_confirm") != "ALLOW_COST" {
		http.Error(w, "비용 발생 확인 문구 ALLOW_COST가 필요합니다.", http.StatusForbidden)
		return
	}

	fullPrompt := buildGridPrompt(prompt, opts)
	client := generator.OpenAIClient{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	}
	imageBytes, err := client.GeneratePNG(r.Context(), generator.OpenAIRequest{
		Prompt:  fullPrompt,
		Model:   r.FormValue("model"),
		Size:    r.FormValue("size"),
		Quality: r.FormValue("quality"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `inline; filename="imagecut_openai_grid.png"`)
	_, _ = w.Write(imageBytes)
}

func buildGridPrompt(prompt string, opts imageproc.GridOptions) string {
	return fmt.Sprintf(
		"%s\n\nCreate a clean %dx%d contact sheet. Use equal square cells, clear white gutters, centered complete subjects, and no text or labels. Do not let any subject cross cell boundaries.",
		prompt,
		opts.Rows,
		opts.Cols,
	)
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

func cutImage(img image.Image, opts imageproc.GridOptions, cropRects string) ([]imageproc.Cut, error) {
	cropRects = strings.TrimSpace(cropRects)
	if cropRects == "" {
		return imageproc.CutGrid(img, opts)
	}

	var rects []imageproc.CropRect
	if err := json.Unmarshal([]byte(cropRects), &rects); err != nil {
		return nil, fmt.Errorf("crop rect JSON이 올바르지 않습니다.")
	}
	return imageproc.CutRects(img, rects)
}

func parseOutputOptions(r *http.Request, inputFormat string) (OutputOptions, error) {
	format := strings.TrimSpace(r.FormValue("output_format"))
	if format == "" || format == "original" {
		format = inputFormat
	}

	quality, err := intField(r, "jpeg_quality", 92)
	if err != nil {
		return OutputOptions{}, err
	}
	if quality < 1 || quality > 100 {
		return OutputOptions{}, fmt.Errorf("jpeg_quality 값은 1부터 100 사이여야 합니다.")
	}

	switch format {
	case "png":
		return OutputOptions{Format: "png", Ext: "png", JPEGQuality: quality}, nil
	case "jpeg":
		return OutputOptions{Format: "jpeg", Ext: "jpg", JPEGQuality: quality}, nil
	default:
		return OutputOptions{}, fmt.Errorf("output_format은 original, png, jpeg 중 하나여야 합니다.")
	}
}

func parseExportFormats(r *http.Request, inputFormat string) ([]ExportFormat, error) {
	batch := r.Form["batch_formats"]
	if len(batch) == 0 {
		output, err := parseOutputOptions(r, inputFormat)
		if err != nil {
			return nil, err
		}
		return []ExportFormat{{Options: output}}, nil
	}

	quality, err := intField(r, "jpeg_quality", 92)
	if err != nil {
		return nil, err
	}
	if quality < 1 || quality > 100 {
		return nil, fmt.Errorf("jpeg_quality 값은 1부터 100 사이여야 합니다.")
	}

	seen := map[string]bool{}
	var outputs []ExportFormat
	for _, raw := range batch {
		requested := strings.TrimSpace(raw)
		if requested == "" {
			continue
		}
		if seen[requested] {
			continue
		}
		seen[requested] = true
		format := requested
		if requested == "original" {
			format = inputFormat
		}
		switch format {
		case "png":
			outputs = append(outputs, ExportFormat{Dir: requested, Options: OutputOptions{Format: "png", Ext: "png", JPEGQuality: quality}})
		case "jpeg":
			outputs = append(outputs, ExportFormat{Dir: requested, Options: OutputOptions{Format: "jpeg", Ext: "jpg", JPEGQuality: quality}})
		default:
			return nil, fmt.Errorf("batch_formats는 original, png, jpeg 중 하나여야 합니다.")
		}
	}
	if len(outputs) == 0 {
		return nil, fmt.Errorf("batch_formats를 하나 이상 선택하세요.")
	}
	return outputs, nil
}

func cutFilename(output ExportFormat, cut imageproc.Cut) string {
	name := fmt.Sprintf("imagecut_r%02d_c%02d.%s", cut.Row+1, cut.Col+1, output.Options.Ext)
	if output.Dir == "" {
		return name
	}
	return output.Dir + "/" + name
}

func buildManifest(r *http.Request, filename string, inputFormat string, bounds image.Rectangle, opts imageproc.GridOptions, outputs []ExportFormat, cuts []imageproc.Cut) ExportManifest {
	manifest := ExportManifest{
		Version:     1,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		ProjectName: strings.TrimSpace(r.FormValue("project_name")),
		Source: ManifestSource{
			Filename: filepath.Base(filename),
			Format:   inputFormat,
			Width:    bounds.Dx(),
			Height:   bounds.Dy(),
		},
		Grid: opts,
	}
	for _, output := range outputs {
		item := ManifestOutput{
			Dir:         output.Dir,
			Format:      output.Options.Format,
			Ext:         output.Options.Ext,
			JPEGQuality: output.Options.JPEGQuality,
		}
		if item.Format != "jpeg" {
			item.JPEGQuality = 0
		}
		manifest.Outputs = append(manifest.Outputs, item)
	}
	for _, cut := range cuts {
		rect := cut.Rect
		manifest.Cuts = append(manifest.Cuts, ManifestCut{
			Row: cut.Row,
			Col: cut.Col,
			X:   rect.Min.X,
			Y:   rect.Min.Y,
			W:   rect.Dx(),
			H:   rect.Dy(),
		})
	}
	return manifest
}

func writeManifest(zw *zip.Writer, manifest ExportManifest) error {
	fw, err := zw.Create("manifest.json")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(fw)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
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

func encodeImage(w io.Writer, img image.Image, opts OutputOptions) error {
	switch opts.Format {
	case "png":
		return png.Encode(w, img)
	case "jpeg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: opts.JPEGQuality})
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}
}

func downloadBaseName(projectName, uploadName string) string {
	base := strings.TrimSpace(projectName)
	if base == "" {
		base = strings.TrimSuffix(filepath.Base(uploadName), filepath.Ext(uploadName))
	}
	return filenameBase(base)
}

func filenameBase(name string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
			lastUnderscore = false
		case unicode.IsSpace(r) && !lastUnderscore && b.Len() > 0:
			b.WriteRune('_')
			lastUnderscore = true
		}
		if b.Len() >= 80 {
			break
		}
	}
	base := strings.Trim(b.String(), "_-")
	if base == "" {
		return "imagecut"
	}
	return base
}

func asciiFilenameBase(name string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_':
			b.WriteRune(r)
			lastUnderscore = false
		case unicode.IsSpace(r) && !lastUnderscore && b.Len() > 0:
			b.WriteRune('_')
			lastUnderscore = true
		}
	}
	base := strings.Trim(b.String(), "_-")
	if base == "" {
		return "imagecut"
	}
	return base
}

func zipContentDisposition(base string) string {
	name := base + "_cuts.zip"
	fallback := asciiFilenameBase(base) + "_cuts.zip"
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, url.PathEscape(name))
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
