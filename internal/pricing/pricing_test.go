package pricing

import "testing"

func TestCurrentIncludesGPTImage2(t *testing.T) {
	table := Current()
	if len(table.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(table.Models))
	}
	model := table.Models[0]
	if model.Model != "gpt-image-2" {
		t.Fatalf("model = %q, want gpt-image-2", model.Model)
	}
	if model.Image.Input != 8 || model.Image.CachedInput != 2 || model.Image.Output != 30 {
		t.Fatalf("image pricing = %+v", model.Image)
	}
	if model.Text.Input != 5 || model.Text.CachedInput != 1.25 {
		t.Fatalf("text pricing = %+v", model.Text)
	}
	if model.SourceURL == "" || model.LastVerified == "" {
		t.Fatal("expected source metadata")
	}
}
