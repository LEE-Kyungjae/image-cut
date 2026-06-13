package imageproc

import (
	"fmt"
	"image"
	"image/draw"
)

type GridOptions struct {
	Rows   int
	Cols   int
	Margin int
	Gutter int
}

type Cut struct {
	Row   int
	Col   int
	Image image.Image
	Rect  image.Rectangle
}

type CropRect struct {
	Row int `json:"row"`
	Col int `json:"col"`
	X   int `json:"x"`
	Y   int `json:"y"`
	W   int `json:"w"`
	H   int `json:"h"`
}

func CutGrid(src image.Image, opts GridOptions) ([]Cut, error) {
	if opts.Rows < 1 || opts.Rows > 8 {
		return nil, fmt.Errorf("rows 값은 1부터 8 사이여야 합니다.")
	}
	if opts.Cols < 1 || opts.Cols > 8 {
		return nil, fmt.Errorf("cols 값은 1부터 8 사이여야 합니다.")
	}
	if opts.Margin < 0 || opts.Gutter < 0 {
		return nil, fmt.Errorf("margin/gutter 값은 0 이상이어야 합니다.")
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	usableW := width - opts.Margin*2 - opts.Gutter*(opts.Cols-1)
	usableH := height - opts.Margin*2 - opts.Gutter*(opts.Rows-1)
	if usableW <= 0 || usableH <= 0 {
		return nil, fmt.Errorf("margin/gutter 값이 이미지보다 큽니다.")
	}

	cellW := usableW / opts.Cols
	cellH := usableH / opts.Rows
	if cellW <= 0 || cellH <= 0 {
		return nil, fmt.Errorf("칸 크기가 너무 작습니다.")
	}

	cuts := make([]Cut, 0, opts.Rows*opts.Cols)
	for row := 0; row < opts.Rows; row++ {
		for col := 0; col < opts.Cols; col++ {
			x0 := bounds.Min.X + opts.Margin + col*(cellW+opts.Gutter)
			y0 := bounds.Min.Y + opts.Margin + row*(cellH+opts.Gutter)
			rect := image.Rect(x0, y0, x0+cellW, y0+cellH)
			cuts = append(cuts, Cut{
				Row:   row,
				Col:   col,
				Image: clone(src, rect),
				Rect:  rect,
			})
		}
	}

	return cuts, nil
}

func CutRects(src image.Image, rects []CropRect) ([]Cut, error) {
	if len(rects) == 0 {
		return nil, fmt.Errorf("자를 영역이 없습니다.")
	}
	if len(rects) > 64 {
		return nil, fmt.Errorf("자를 영역은 최대 64개까지 지원합니다.")
	}

	bounds := src.Bounds()
	cuts := make([]Cut, 0, len(rects))
	for _, item := range rects {
		if item.Row < 0 || item.Col < 0 {
			return nil, fmt.Errorf("crop rect의 row/col 값이 올바르지 않습니다.")
		}
		if item.W <= 0 || item.H <= 0 {
			return nil, fmt.Errorf("crop rect의 너비/높이는 1 이상이어야 합니다.")
		}

		rect := image.Rect(
			bounds.Min.X+item.X,
			bounds.Min.Y+item.Y,
			bounds.Min.X+item.X+item.W,
			bounds.Min.Y+item.Y+item.H,
		)
		if !rect.In(bounds) {
			return nil, fmt.Errorf("crop rect가 이미지 범위를 벗어났습니다.")
		}

		cuts = append(cuts, Cut{
			Row:   item.Row,
			Col:   item.Col,
			Image: clone(src, rect),
			Rect:  rect,
		})
	}

	return cuts, nil
}

func clone(src image.Image, rect image.Rectangle) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
	return dst
}
