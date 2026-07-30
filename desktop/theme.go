package main

import (
	"bytes"
	_ "embed"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The zlef.fr design language, transposed to a native window: near-black
// surfaces, warm off-white type, olive as the primary accent and grape as the
// secondary. Values are the canonical --zl-* tokens from da.zlef.fr/tokens.css.
var (
	colBG       = rgb(0x06060a)
	colSurface1 = rgb(0x0e0e13)
	colSurface2 = rgb(0x15151c)
	colSurface3 = rgb(0x1d1d25)
	colLine     = color.RGBA{255, 255, 255, 18}
	colLineHot  = color.RGBA{157, 174, 80, 140}
	colText     = rgb(0xe9eae2)
	colTextSoft = rgb(0xb6b7ad)
	colTextMute = rgb(0x7d7e76)
	colTextFain = rgb(0x54554f)
	colOlive    = rgb(0x3e4618)
	colOliveMid = rgb(0x59642a)
	colOliveSft = rgb(0x9dae50)
	colOliveBrt = rgb(0xbdce74)
	colGrapeSft = rgb(0xb095b3)
	colSuccess  = rgb(0x6f9a3a)
	colWarning  = rgb(0xc79a3e)
	colDanger   = rgb(0xc2566a)
)

func rgb(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}

// alpha returns c faded to a (0..1) — used for every entrance/fade, since
// nothing in a zlef UI should pop in abruptly.
func alpha(c color.RGBA, a float64) color.RGBA {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return color.RGBA{
		uint8(float64(c.R) * a), uint8(float64(c.G) * a),
		uint8(float64(c.B) * a), uint8(float64(c.A) * a),
	}
}

// Fonts are embedded (DejaVu, permissive licence in assets/FONT-LICENSE.txt) so
// the binary renders identically on a bare Windows box and a Linux laptop with
// no fontconfig setup. The mono face carries the numbers in the HUD, where
// tabular digits stop the readout from twitching.
//
//go:embed assets/DejaVuSans.ttf
var fontSansTTF []byte

//go:embed assets/DejaVuSansMono.ttf
var fontMonoTTF []byte

type faces struct {
	xl, lg, md, sm, xs *text.GoTextFace
	mono, monoSm       *text.GoTextFace
}

func loadFaces() *faces {
	sans, err := text.NewGoTextFaceSource(bytes.NewReader(fontSansTTF))
	if err != nil {
		log.Fatalf("font: %v", err)
	}
	mono, err := text.NewGoTextFaceSource(bytes.NewReader(fontMonoTTF))
	if err != nil {
		log.Fatalf("font: %v", err)
	}
	f := func(src *text.GoTextFaceSource, size float64) *text.GoTextFace {
		return &text.GoTextFace{Source: src, Size: size}
	}
	// "never too small": body sits at 17px like the web tokens.
	return &faces{
		xl:     f(sans, 34),
		lg:     f(sans, 22),
		md:     f(sans, 17),
		sm:     f(sans, 14.5),
		xs:     f(sans, 13),
		mono:   f(mono, 15),
		monoSm: f(mono, 12.5),
	}
}
