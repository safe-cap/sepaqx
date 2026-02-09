package qr

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	xdraw "golang.org/x/image/draw"
)

func OverlayLogoImage(qrPNG []byte, logoImg image.Image, ratio float64, bgShape string) ([]byte, error) {
	if logoImg == nil || ratio <= 0 {
		return qrPNG, nil
	}

	qrImg, err := png.Decode(bytes.NewReader(qrPNG))
	if err != nil {
		return nil, err
	}

	qrRGBA := image.NewRGBA(qrImg.Bounds())
	draw.Draw(qrRGBA, qrRGBA.Bounds(), qrImg, image.Point{}, draw.Src)

	qrW := qrRGBA.Bounds().Dx()

	target := int(math.Round(float64(qrW) * ratio))
	if target < 40 {
		target = 40
	}

	lb := logoImg.Bounds()
	lw := lb.Dx()
	lh := lb.Dy()
	if lw == 0 || lh == 0 {
		return qrPNG, nil
	}

	scale := math.Min(float64(target)/float64(lw), float64(target)/float64(lh))
	newW := int(math.Round(float64(lw) * scale))
	newH := int(math.Round(float64(lh) * scale))

	logoResized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(logoResized, logoResized.Bounds(), logoImg, lb, draw.Over, nil)

	qrH := qrRGBA.Bounds().Dy()

	x := (qrW - newW) / 2
	y := (qrH - newH) / 2

	pad := int(math.Round(float64(qrW) * 0.02))
	if pad < 8 {
		pad = 8
	}
	bg := image.Rect(x-pad, y-pad, x+newW+pad, y+newH+pad)
	if bgShape == "circle" {
		fillCircle(qrRGBA, bg, color.White)
	} else {
		draw.Draw(qrRGBA, bg, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	}
	draw.Draw(qrRGBA, image.Rect(x, y, x+newW, y+newH), logoResized, image.Point{}, draw.Over)

	var out bytes.Buffer
	if err := png.Encode(&out, qrRGBA); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func fillCircle(img *image.RGBA, rect image.Rectangle, c color.Color) {
	w := rect.Dx()
	h := rect.Dy()
	size := w
	if h < w {
		size = h
	}
	r := size / 2
	cx := rect.Min.X + w/2
	cy := rect.Min.Y + h/2
	r2 := r * r
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				img.Set(x, y, c)
			}
		}
	}
}
