package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIClientGeneratePNG(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("path = %s, want /images/generations", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header missing")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + base64.StdEncoding.EncodeToString([]byte("png-bytes")) + `"}]}`))
	}))
	defer server.Close()

	client := OpenAIClient{APIKey: "test-key", BaseURL: server.URL, HTTP: server.Client()}
	out, err := client.GeneratePNG(context.Background(), OpenAIRequest{
		Prompt:  "3x3 stickers",
		Model:   "gpt-image-2",
		Size:    "1024x1024",
		Quality: "low",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "png-bytes" {
		t.Fatalf("output = %q, want png-bytes", string(out))
	}
	if got["output_format"] != "png" {
		t.Fatalf("output_format = %v, want png", got["output_format"])
	}
}

func TestOpenAIClientRequiresKey(t *testing.T) {
	client := OpenAIClient{}
	if _, err := client.GeneratePNG(context.Background(), OpenAIRequest{Prompt: "x"}); err == nil {
		t.Fatal("expected missing key error")
	}
}
