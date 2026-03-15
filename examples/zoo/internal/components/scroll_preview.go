package components

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	"github.com/kellydornhaus/layouter/layout"
)

// ScrollPreview draws a fading minimap thumbnail for a scroll container.
type ScrollPreview struct {
	FadeTicks  int
	FadeMax    int
	lastOffset float64
}

func NewScrollPreview() *ScrollPreview {
	return &ScrollPreview{FadeMax: 120}
}

func (p *ScrollPreview) Trigger(offset float64) {
	if p.FadeMax <= 0 {
		p.FadeMax = 120
	}
	p.FadeTicks = p.FadeMax
	p.lastOffset = offset
}

func (p *ScrollPreview) Tick() {
	if p.FadeTicks > 0 {
		p.FadeTicks--
	}
}

func (p *ScrollPreview) Active() bool { return p.FadeTicks > 0 }

func (p *ScrollPreview) alpha() float64 {
	if p.FadeMax <= 0 || p.FadeTicks <= 0 {
		return 0
	}
	return float64(p.FadeTicks) / float64(p.FadeMax)
}

// Painter returns a function suitable for Overlay painter callbacks.
func (p *ScrollPreview) Painter(scroll *Scroll) func(ctx *layout.Context, dst layout.Surface) {
	return func(ctx *layout.Context, dst layout.Surface) { p.Draw(ctx, dst, scroll) }
}

func (p *ScrollPreview) Draw(ctx *layout.Context, dst layout.Surface, scroll *Scroll) {
	if !p.Active() || scroll == nil {
		return
	}
	img := scroll.SnapshotImage(ctx)
	if img == nil {
		return
	}
	surf, ok := dst.(*adp.Surface)
	if !ok {
		return
	}
	dstImg := surf.Img
	dw, dh := dstImg.Size()
	if dw <= 0 || dh <= 0 {
		return
	}
	sw, sh := img.Size()
	if sw <= 0 || sh <= 0 {
		return
	}

	margin := int(math.Round(12 * ctx.Scale))
	if margin < 8 {
		margin = 8
	}
	maxThumbW := dw / 4
	if maxThumbW > 220 {
		maxThumbW = 220
	}
	if maxThumbW < 120 {
		maxThumbW = 120
	}
	thumbW := sw
	if thumbW > maxThumbW {
		thumbW = maxThumbW
	}
	scale := float64(thumbW) / float64(sw)
	thumbH := int(math.Round(float64(sh) * scale))
	maxThumbH := dh / 3
	if maxThumbH > 200 {
		maxThumbH = 200
	}
	if maxThumbH > 0 && thumbH > maxThumbH {
		scale = float64(maxThumbH) / float64(sh)
		thumbH = maxThumbH
		thumbW = int(math.Round(float64(sw) * scale))
	}
	if thumbH < 40 {
		thumbH = 40
	}
	if thumbW < 80 {
		thumbW = 80
		scale = float64(thumbW) / float64(sw)
		thumbH = int(math.Round(float64(sh) * scale))
		if thumbH < 40 {
			thumbH = 40
		}
	}

	dx := dw - thumbW - margin
	if dx < margin {
		dx = margin
	}
	dy := dh - thumbH - margin
	if dy < margin {
		dy = margin
	}

	alpha := p.alpha()
	bg := ebiten.NewImage(thumbW, thumbH)
	bg.Fill(color.RGBA{0, 0, 0, uint8(160 * alpha)})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(dx), float64(dy))
	dstImg.DrawImage(bg, op)

	op = &ebiten.DrawImageOptions{}
	op.ColorM.Scale(0.9, 0.9, 0.9, alpha)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(dx), float64(dy))
	dstImg.DrawImage(img, op)

	offsetPx := int(math.Round(scroll.Offset() * ctx.Scale))
	viewportPx := int(math.Round(scroll.ViewportSize().H * ctx.Scale))
	tailPx := int(math.Round(scroll.TailPad() * ctx.Scale))
	contentPx := sh + tailPx
	maxOffsetPx := contentPx - viewportPx
	if maxOffsetPx < 1 {
		maxOffsetPx = 1
	}
	if offsetPx > maxOffsetPx {
		offsetPx = maxOffsetPx
	}

	topPx := int(math.Round(float64(offsetPx) * scale))
	heightPx := int(math.Round(float64(viewportPx) * scale))
	if heightPx < 3 {
		heightPx = 3
	}
	if topPx+heightPx > thumbH {
		heightPx = thumbH - topPx
		if heightPx < 3 {
			heightPx = 3
		}
		topPx = thumbH - heightPx
	}

	highlight := ebiten.NewImage(max(thumbW-8, 4), max(heightPx-4, 3))
	highlight.Fill(color.RGBA{120, 180, 255, uint8(170 * alpha)})
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(dx+4), float64(dy+topPx+2))
	dstImg.DrawImage(highlight, op)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
