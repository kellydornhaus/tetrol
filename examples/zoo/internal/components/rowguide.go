package components

import (
	"github.com/hajimehoshi/ebiten/v2"
	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	"github.com/kellydornhaus/layouter/layout"
	"image/color"
)

// RowGuide draws horizontal lines at the top and bottom of its bounds for debugging alignment.
type RowGuide struct {
	layout.Base
	Child     layout.Component
	Color     color.Color
	Thickness int
}

func NewRowGuide(child layout.Component) *RowGuide {
	return &RowGuide{Child: child, Color: color.RGBA{255, 0, 255, 200}, Thickness: 1}
}

func (r *RowGuide) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size {
	if r.Child == nil {
		return cs.Min
	}
	return r.Child.Measure(ctx, cs)
}

func (r *RowGuide) Layout(ctx *layout.Context, parent layout.Component, bounds layout.Rect) {
	r.SetFrame(parent, bounds)
	if r.Child != nil {
		r.Child.Layout(ctx, r, layout.Rect{X: 0, Y: 0, W: bounds.W, H: bounds.H})
		if r.Child.Dirty() {
			r.SetDirty()
		}
	}
}

func (r *RowGuide) DrawTo(ctx *layout.Context, dst layout.Surface) {
	r.Base.DrawCachedWithOwner(ctx, dst, r, func(t layout.Surface) { r.Render(ctx, t) })
}

func (r *RowGuide) Render(ctx *layout.Context, dst layout.Surface) {
	// draw child first
	if r.Child != nil {
		r.Child.DrawTo(ctx, dst)
	}
	// then draw guides on top
	s, ok := dst.(*adp.Surface)
	if !ok {
		return
	}
	w, h := s.SizePx()
	th := r.Thickness
	if th < 1 {
		th = 1
	}
	// top line
	top := ebiten.NewImage(w, th)
	top.Fill(r.Color)
	s.Img.DrawImage(top, &ebiten.DrawImageOptions{})
	// bottom line
	bot := ebiten.NewImage(w, th)
	bot.Fill(r.Color)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, float64(h-th))
	s.Img.DrawImage(bot, op)
}
