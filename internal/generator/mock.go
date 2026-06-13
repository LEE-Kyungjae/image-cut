package generator

import (
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"

	"imagecut/internal/imageproc"
)

func MockGrid(prompt string, opts imageproc.GridOptions) (image.Image, error) {
	if _, err := imageproc.CutGrid(image.NewRGBA(image.Rect(0, 0, 1200, 1200)), opts); err != nil {
		return nil, err
	}

	const size = 1200
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 248, G: 250, B: 252, A: 255}), image.Point{}, draw.Src)

	usableW := size - opts.Margin*2 - opts.Gutter*(opts.Cols-1)
	usableH := size - opts.Margin*2 - opts.Gutter*(opts.Rows-1)
	cellW := usableW / opts.Cols
	cellH := usableH / opts.Rows
	seed := hash(prompt)

	for row := 0; row < opts.Rows; row++ {
		for col := 0; col < opts.Cols; col++ {
			index := uint32(row*opts.Cols + col)
			fill := colorFromSeed(seed + index*37)
			ink := darken(fill)
			x0 := opts.Margin + col*(cellW+opts.Gutter)
			y0 := opts.Margin + row*(cellH+opts.Gutter)
			rect := image.Rect(x0, y0, x0+cellW, y0+cellH)
			draw.Draw(img, rect, image.NewUniform(fill), image.Point{}, draw.Src)
			drawBorder(img, rect, ink, 8)
			drawMarker(img, rect, ink, row, col)
		}
	}

	return img, nil
}

func hash(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}

func colorFromSeed(seed uint32) color.RGBA {
	return color.RGBA{
		R: uint8(96 + seed%120),
		G: uint8(96 + (seed/7)%120),
		B: uint8(96 + (seed/13)%120),
		A: 255,
	}
}

func darken(c color.RGBA) color.RGBA {
	return color.RGBA{R: c.R / 2, G: c.G / 2, B: c.B / 2, A: 255}
}

func drawBorder(img *image.RGBA, rect image.Rectangle, c color.RGBA, width int) {
	draw.Draw(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+width), image.NewUniform(c), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(rect.Min.X, rect.Max.Y-width, rect.Max.X, rect.Max.Y), image.NewUniform(c), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+width, rect.Max.Y), image.NewUniform(c), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(rect.Max.X-width, rect.Min.Y, rect.Max.X, rect.Max.Y), image.NewUniform(c), image.Point{}, draw.Src)
}

func drawMarker(img *image.RGBA, rect image.Rectangle, c color.RGBA, row, col int) {
	center := image.Point{X: (rect.Min.X + rect.Max.X) / 2, Y: (rect.Min.Y + rect.Max.Y) / 2}
	radius := min(rect.Dx(), rect.Dy()) / 8
	if radius < 12 {
		radius = 12
	}

	for y := center.Y - radius; y <= center.Y+radius; y++ {
		for x := center.X - radius; x <= center.X+radius; x++ {
			dx := x - center.X
			dy := y - center.Y
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x, y, c)
			}
		}
	}

	step := radius / 3
	for i := 0; i <= row; i++ {
		x := center.X - radius/2 + i*step
		draw.Draw(img, image.Rect(x, center.Y-radius/5, x+step/2+1, center.Y+radius/5), image.NewUniform(color.White), image.Point{}, draw.Src)
	}
	for i := 0; i <= col; i++ {
		y := center.Y - radius/2 + i*step
		draw.Draw(img, image.Rect(center.X-radius/5, y, center.X+radius/5, y+step/2+1), image.NewUniform(color.White), image.Point{}, draw.Src)
	}
}
