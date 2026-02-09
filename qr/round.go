package qr

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func renderStyled(modules [][]bool, size int, style Style) *image.RGBA {
	n := len(modules)
	if n == 0 || size <= 0 {
		return image.NewRGBA(image.Rect(0, 0, size, size))
	}

	quiet := 4
	if style.QuietZone > 0 {
		quiet = style.QuietZone
	}
	total := n + quiet*2
	if total <= 0 {
		return image.NewRGBA(image.Rect(0, 0, size, size))
	}
	scale := float64(size) / float64(total)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	moduleRadius := style.ModuleRadius
	if moduleRadius <= 0 {
		moduleRadius = 0.25
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !modules[y][x] {
				continue
			}
			radius := 0.0
			if style.ModuleStyle == "rounded" || style.ModuleStyle == "blob" {
				if style.ModuleStyle == "blob" {
					if style.ModuleRadius > 0 {
						radius = style.ModuleRadius
					} else {
						radius = 0.5
					}
				} else {
					radius = moduleRadius
				}
			}

			px := int(math.Round(float64(x+quiet) * scale))
			py := int(math.Round(float64(y+quiet) * scale))
			pw := int(math.Round(float64(x+quiet+1)*scale)) - px
			ph := int(math.Round(float64(y+quiet+1)*scale)) - py
			if pw < 1 {
				pw = 1
			}
			if ph < 1 {
				ph = 1
			}
			if radius <= 0 {
				fillRect(img, px, py, pw, ph, color.Black)
			} else {
				fillRoundedRect(img, px, py, pw, ph, radius, color.Black)
			}
		}
	}

	if style.CornerRadius > 0 {
		applyCornerRadius(img, style.CornerRadius)
	}

	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			img.Set(xx, yy, c)
		}
	}
}

func fillRoundedRect(img *image.RGBA, x, y, w, h int, radiusFrac float64, c color.Color) {
	r := int(math.Round(float64(min(w, h)) * radiusFrac))
	if r <= 0 {
		fillRect(img, x, y, w, h, c)
		return
	}
	r2 := r * r
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			dx := min(xx, w-1-xx)
			dy := min(yy, h-1-yy)
			if dx >= r || dy >= r {
				img.Set(x+xx, y+yy, c)
				continue
			}
			ox := r - dx
			oy := r - dy
			if ox*ox+oy*oy <= r2 {
				img.Set(x+xx, y+yy, c)
			}
		}
	}
}

func applyCornerRadius(img *image.RGBA, radius int) {
	if radius <= 0 {
		return
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	r := radius
	if r*2 > w {
		r = w / 2
	}
	if r*2 > h {
		r = h / 2
	}
	r2 := r * r
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := min(x, w-1-x)
			dy := min(y, h-1-y)
			if dx >= r || dy >= r {
				continue
			}
			ox := r - dx
			oy := r - dy
			if ox*ox+oy*oy > r2 {
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
