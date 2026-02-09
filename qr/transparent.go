package qr

import (
	"image"
	"image/color"
)

func MakeBackgroundTransparent(img image.Image) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(b)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b2, a := img.At(x, y).RGBA()

			// White → transparent
			if r>>8 == 255 && g>>8 == 255 && b2>>8 == 255 {
				out.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 0})
			} else {
				out.SetNRGBA(x, y, color.NRGBA{
					uint8(r >> 8),
					uint8(g >> 8),
					uint8(b2 >> 8),
					uint8(a >> 8),
				})
			}
		}
	}
	return out
}
