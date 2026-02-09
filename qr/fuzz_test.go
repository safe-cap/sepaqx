package qr

import "testing"

func FuzzRecolorGradient(f *testing.F) {
	pngBytes, _ := MakeQR("BCD\n001\n1\nSCT\nTESTBIC\nTEST NAME\nDE12500105170648489890\nEUR1.00\nGDDS\n\n\n", DefaultPublicOptions())
	if pngBytes == nil {
		f.Skip("unable to generate seed png")
	}
	f.Add("#000000", "#ffffff", "#7a5cff", "#3aa8ff", float64(45))
	f.Add("#111111", "transparent", "#ff0000", "#00ff00", float64(90))
	f.Fuzz(func(t *testing.T, fg, bg, from, to string, angle float64) {
		_, _ = RecolorGradient(pngBytes, fg, bg, &GradientSpec{From: from, To: to, Angle: angle}, &GradientSpec{From: from, To: to, Angle: angle})
	})
}

func FuzzHexColors(f *testing.F) {
	seeds := []string{"#000000", "#ffffff", "ffffff", "#a1b2c3", "gggggg", "#12345", "transparent", ""}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseHexColor(s)
		_, _, _ = parseBgColor(s)
	})
}
