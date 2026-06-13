package generator

import (
	"testing"

	"imagecut/internal/imageproc"
)

func TestMockGrid(t *testing.T) {
	img, err := MockGrid("robot stickers", imageproc.GridOptions{Rows: 3, Cols: 3, Margin: 24, Gutter: 24})
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Dx(); got != 1200 {
		t.Fatalf("width = %d, want 1200", got)
	}
	if got := img.Bounds().Dy(); got != 1200 {
		t.Fatalf("height = %d, want 1200", got)
	}
}

func TestMockGridRejectsBadOptions(t *testing.T) {
	if _, err := MockGrid("bad", imageproc.GridOptions{Rows: 0, Cols: 3}); err == nil {
		t.Fatal("expected options error")
	}
}
