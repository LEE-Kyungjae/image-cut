package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAIClient struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

type OpenAIRequest struct {
	Prompt  string
	Model   string
	Size    string
	Quality string
}

func (c OpenAIClient) GeneratePNG(ctx context.Context, req OpenAIRequest) ([]byte, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY가 설정되어 있지 않습니다.")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt가 비어 있습니다.")
	}

	model := fallback(req.Model, "gpt-image-2")
	size := fallback(req.Size, "1024x1024")
	quality := fallback(req.Quality, "low")
	baseURL := strings.TrimRight(fallback(c.BaseURL, DefaultOpenAIBaseURL), "/")
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}

	payload := map[string]any{
		"model":         model,
		"prompt":        req.Prompt,
		"size":          size,
		"quality":       quality,
		"output_format": "png",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OpenAI image API error: %s", strings.TrimSpace(string(respBody)))
	}

	var decoded struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("OpenAI 응답을 해석할 수 없습니다.")
	}
	if len(decoded.Data) == 0 || decoded.Data[0].B64JSON == "" {
		return nil, fmt.Errorf("OpenAI 응답에 이미지 데이터가 없습니다.")
	}

	imageBytes, err := base64.StdEncoding.DecodeString(decoded.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("OpenAI 이미지 데이터를 디코딩할 수 없습니다.")
	}
	return imageBytes, nil
}

func fallback(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}
