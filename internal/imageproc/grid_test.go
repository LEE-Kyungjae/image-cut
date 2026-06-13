package imageproc

import (
	"image"
	"image/color"
	"testing"
)

func TestCutGrid(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 80))
	src.Set(10, 10, color.RGBA{R: 255, A: 255})

	cuts, err := CutGrid(src, GridOptions{
		Rows:   2,
		Cols:   2,
		Margin: 10,
		Gutter: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cuts) != 4 {
		t.Fatalf("len(cuts) = %d, want 4", len(cuts))
	}

	first := cuts[0]
	if got := first.Image.Bounds().Dx(); got != 35 {
		t.Fatalf("first width = %d, want 35", got)
	}
	if got := first.Image.Bounds().Dy(); got != 25 {
		t.Fatalf("first height = %d, want 25", got)
	}
	if got := first.Image.At(0, 0); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("first pixel = %v, want red", got)
	}
}

func TestCutGridRejectsInvalidOptions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	if _, err := CutGrid(src, GridOptions{Rows: 0, Cols: 2}); err == nil {
		t.Fatal("expected rows error")
	}
	if _, err := CutGrid(src, GridOptions{Rows: 2, Cols: 2, Margin: 60}); err == nil {
		t.Fatal("expected margin error")
	}
}

func TestCutRects(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 80))
	src.Set(12, 14, color.RGBA{G: 255, A: 255})

	cuts, err := CutRects(src, []CropRect{
		{Row: 0, Col: 0, X: 10, Y: 12, W: 20, H: 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cuts) != 1 {
		t.Fatalf("len(cuts) = %d, want 1", len(cuts))
	}
	if got := cuts[0].Image.Bounds().Dx(); got != 20 {
		t.Fatalf("width = %d, want 20", got)
	}
	if got := cuts[0].Image.Bounds().Dy(); got != 30 {
		t.Fatalf("height = %d, want 30", got)
	}
	if got := cuts[0].Image.At(2, 2); got != (color.RGBA{G: 255, A: 255}) {
		t.Fatalf("pixel = %v, want green", got)
	}
}

func TestCutRectsRejectsOutOfBounds(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 80))
	if _, err := CutRects(src, []CropRect{{Row: 0, Col: 0, X: 90, Y: 70, W: 20, H: 20}}); err == nil {
		t.Fatal("expected out-of-bounds error")
	}
}
