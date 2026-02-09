package qr

import (
	qrcode "github.com/skip2/go-qrcode"
)

type Options struct {
	Size int
	ECC  qrcode.RecoveryLevel
}

type Style struct {
	CornerRadius int
	ModuleStyle  string
	ModuleRadius float64
	QuietZone    int
}

func DefaultPublicOptions() Options {
	// Public mode: fixed, boring, highly compatible.
	return Options{
		Size: 512,
		ECC:  qrcode.Medium, // "M"
	}
}

func DefaultAuthOptions(withLogo bool) Options {
	// Auth mode: can be customized; ECC is increased only if a logo is used.
	ecc := qrcode.Medium // "M"
	if withLogo {
		ecc = qrcode.Highest // "H"
	}
	return Options{
		Size: 512,
		ECC:  ecc,
	}
}

func MakeQR(payload string, opt Options) ([]byte, error) {
	qr, err := qrcode.New(payload, opt.ECC)
	if err != nil {
		return nil, err
	}

	img := qr.Image(opt.Size)
	imgTransparent := MakeBackgroundTransparent(img)

	return EncodePNG(imgTransparent)
}

func MakeQRStyled(payload string, opt Options, style Style) ([]byte, error) {
	qr, err := qrcode.New(payload, opt.ECC)
	if err != nil {
		return nil, err
	}
	img := renderStyled(qr.Bitmap(), opt.Size, style)
	return EncodePNG(img)
}
