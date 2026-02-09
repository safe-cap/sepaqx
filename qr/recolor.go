package qr

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
)

func Recolor(qrPNG []byte, fgHex, bgHex string) ([]byte, error) {
	return RecolorGradient(qrPNG, fgHex, bgHex, nil, nil)
}

func RecolorGradient(qrPNG []byte, fgHex, bgHex string, fgGrad, bgGrad *GradientSpec) ([]byte, error) {
	fg, err := parseHexColor(fgHex)
	if err != nil {
		return nil, fmt.Errorf("invalid fg color: %w", err)
	}

	bg, transparentBG, err := parseBgColor(bgHex)
	if err != nil {
		return nil, fmt.Errorf("invalid bg color: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(qrPNG))
	if err != nil {
		return nil, err
	}

	b := img.Bounds()
	out := image.NewRGBA(b)

	var fgGradFn func(x, y int) color.RGBA
	var bgGradFn func(x, y int) color.RGBA
	if fgGrad != nil {
		fgGradFn = makeGradientFn(b, fgGrad.From, fgGrad.To, fgGrad.Angle)
	}
	if bgGrad != nil {
		bgGradFn = makeGradientFn(b, bgGrad.From, bgGrad.To, bgGrad.Angle)
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bb, a := img.At(x, y).RGBA()
			if a == 0 {
				if transparentBG {
					out.Set(x, y, color.RGBA{0, 0, 0, 0})
				} else if bgGradFn != nil {
					out.Set(x, y, bgGradFn(x, y))
				} else {
					out.Set(x, y, bg)
				}
				continue
			}
			if r > 0xeeee && g > 0xeeee && bb > 0xeeee {
				if transparentBG {
					out.Set(x, y, color.RGBA{0, 0, 0, 0})
				} else if bgGradFn != nil {
					out.Set(x, y, bgGradFn(x, y))
				} else {
					out.Set(x, y, bg)
				}
			} else {
				if fgGradFn != nil {
					out.Set(x, y, fgGradFn(x, y))
				} else {
					out.Set(x, y, fg)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseBgColor(s string) (color.RGBA, bool, error) {
	v := strings.TrimSpace(strings.ToLower(s))
	if v == "" {
		// Default background is white when explicitly recoloring.
		return color.RGBA{255, 255, 255, 255}, false, nil
	}
	if v == "transparent" || v == "none" {
		return color.RGBA{0, 0, 0, 0}, true, nil
	}
	c, err := parseHexColor(v)
	if err != nil {
		return color.RGBA{}, false, err
	}
	return c, false, nil
}

type GradientSpec struct {
	From  string
	To    string
	Angle float64
}

func makeGradientFn(b image.Rectangle, fromHex, toHex string, angle float64) func(x, y int) color.RGBA {
	from, _ := parseHexColor(fromHex)
	to, _ := parseHexColor(toHex)

	// Normalize angle and compute direction.
	rad := angle * (3.141592653589793 / 180.0)
	dx := math.Cos(rad)
	dy := math.Sin(rad)
	if dx == 0 && dy == 0 {
		dx = 1
	}

	// Project corners to get min/max.
	x0 := float64(b.Min.X)
	y0 := float64(b.Min.Y)
	x1 := float64(b.Max.X)
	y1 := float64(b.Max.Y)
	corners := [4][2]float64{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}}
	minp := math.Inf(1)
	maxp := math.Inf(-1)
	for _, c := range corners {
		p := c[0]*dx + c[1]*dy
		if p < minp {
			minp = p
		}
		if p > maxp {
			maxp = p
		}
	}
	den := maxp - minp
	if den == 0 {
		den = 1
	}

	return func(x, y int) color.RGBA {
		p := float64(x)*dx + float64(y)*dy
		t := (p - minp) / den
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		return lerpColor(from, to, t)
	}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}
func parseHexColor(s string) (color.RGBA, error) {
	v := strings.TrimSpace(strings.ToLower(s))
	if v == "" {
		return color.RGBA{0, 0, 0, 255}, nil
	}

	// Trim leading '#' if present (safe even if it's not there)
	v = strings.TrimPrefix(v, "#")

	if len(v) != 6 {
		return color.RGBA{}, fmt.Errorf("expected 6 hex digits")
	}

	var rgb [3]uint8
	for i := 0; i < 3; i++ {
		b, err := parseHexByte(v[i*2 : i*2+2])
		if err != nil {
			return color.RGBA{}, err
		}
		rgb[i] = b
	}

	return color.RGBA{rgb[0], rgb[1], rgb[2], 255}, nil
}

func parseHexByte(s string) (uint8, error) {
	var x uint8
	for _, r := range s {
		x <<= 4
		switch {
		case r >= '0' && r <= '9':
			x |= uint8(r - '0')
		case r >= 'a' && r <= 'f':
			x |= uint8(r-'a') + 10
		default:
			return 0, fmt.Errorf("invalid hex")
		}
	}
	return x, nil
}
