package etxtadapter

import (
	"image/color"
	"log"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kellydornhaus/layouter/layout"
	"github.com/tinne26/etxt"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

// EbitenTarget is satisfied by surfaces that can expose an *ebiten.Image.
type EbitenTarget interface{ AsEbiten() *ebiten.Image }

// EtxtEngine implements layout.TextEngine using github.com/tinne26/etxt.
// Requires that dst implements EbitenTarget (e.g., adapters/ebiten.Surface).
type EtxtEngine struct {
	scaleProvider layout.PixelScaleProvider
	r             *etxt.Renderer
	fonts         map[string]*sfnt.Font
	pixelScale    float64
	fallbackRune  rune

	missingMu     sync.Mutex
	missingGlyphs map[rune]struct{}
}

func New(scaleProvider layout.PixelScaleProvider) *EtxtEngine {
	r := etxt.NewRenderer()
	r.Utils().SetCache8MiB()
	fnt, _ := sfnt.Parse(goregular.TTF)
	r.SetFont(fnt)
	engine := &EtxtEngine{
		scaleProvider: scaleProvider,
		r:             r,
		fonts:         map[string]*sfnt.Font{"": fnt, "regular": fnt},
		fallbackRune:  '?',
	}
	r.Glyph().SetMissHandler(engine.handleMissingGlyph)
	return engine
}

func (e *EtxtEngine) RegisterFont(key string, f *sfnt.Font) { e.fonts[key] = f }

func (e *EtxtEngine) setStyle(style layout.TextStyle) {
	if f, ok := e.fonts[style.FontKey]; ok && f != nil {
		e.r.SetFont(f)
	}
	scale := 1.0
	if e.scaleProvider != nil {
		if s := e.scaleProvider.DeviceScaleFactor(); s > 0 {
			scale = s
		}
	}
	e.pixelScale = scale
	e.r.SetScale(scale)
	e.r.SetSize(style.SizeDp)
	clr := color.RGBA{R: style.Color.R, G: style.Color.G, B: style.Color.B, A: style.Color.A}
	if clr.R == 0 && clr.G == 0 && clr.B == 0 && clr.A == 0 {
		clr = color.RGBA{255, 255, 255, 255}
	}
	e.r.SetColor(clr)
	e.r.SetAlign(mapAlign(style.AlignH, style.AlignV))
}

func (e *EtxtEngine) handleMissingGlyph(font *sfnt.Font, codePoint rune) (sfnt.GlyphIndex, bool) {
	e.missingMu.Lock()
	if e.missingGlyphs == nil {
		e.missingGlyphs = make(map[rune]struct{})
	}
	if _, seen := e.missingGlyphs[codePoint]; !seen {
		e.missingGlyphs[codePoint] = struct{}{}
		log.Printf("etxt: missing glyph for %q (%U), using fallback", codePoint, codePoint)
	}
	e.missingMu.Unlock()

	if e.fallbackRune != 0 {
		if idx, err := font.GlyphIndex(nil, e.fallbackRune); err == nil && idx != 0 {
			return idx, false
		}
	}
	return 0, true
}

func mapAlign(h layout.TextAlign, v layout.TextAlign) etxt.Align {
	var a etxt.Align
	switch h {
	case layout.AlignStart:
		a |= etxt.Left
	case layout.AlignCenter:
		a |= etxt.HorzCenter
	case layout.AlignEnd:
		a |= etxt.Right
	}
	switch v {
	case layout.AlignStart:
		a |= etxt.Top
	case layout.AlignCenter:
		a |= etxt.VertCenter
	case layout.AlignEnd:
		a |= etxt.Bottom
	}
	if a == 0 {
		a = etxt.Left | etxt.Baseline
	}
	return a
}

func (e *EtxtEngine) Measure(text string, style layout.TextStyle, maxWidthPx int) (int, int) {
	e.setStyle(style)
	rect := e.r.Measure(text)
	if style.Wrap && maxWidthPx > 0 {
		rect = e.r.MeasureWithWrap(text, maxWidthPx)
	}
	return rect.IntWidth(), rect.IntHeight()
}

func (e *EtxtEngine) Draw(dst layout.Surface, text string, rectPx layout.PxRect, style layout.TextStyle) {
	e.setStyle(style)
	target, ok := dst.(EbitenTarget)
	if !ok {
		return
	}
	img := target.AsEbiten()
	ax := rectPx.X
	ay := rectPx.Y
	switch style.AlignH {
	case layout.AlignCenter:
		ax += rectPx.W / 2
	case layout.AlignEnd:
		ax += rectPx.W
	}
	switch style.AlignV {
	case layout.AlignCenter:
		ay += rectPx.H / 2
	case layout.AlignEnd:
		ay += rectPx.H
	}
	if style.BaselineOffset != 0 && e.pixelScale > 0 {
		ay -= int(math.Round(style.BaselineOffset * e.pixelScale))
	}
	if style.Wrap && rectPx.W > 0 {
		e.r.DrawWithWrap(img, text, ax, ay, rectPx.W)
	} else {
		e.r.Draw(img, text, ax, ay)
	}
}
