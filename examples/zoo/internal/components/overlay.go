package components

import "github.com/kellydornhaus/layouter/layout"

// NewOverlay composes child and painter layers using a ZStack.
func NewOverlay(child layout.Component, painter func(ctx *layout.Context, dst layout.Surface)) layout.Component {
	if painter == nil {
		return child
	}
	paintLayer := &painterComponent{Painter: painter}
	return layout.NewZStack(child, paintLayer)
}

type painterComponent struct {
	layout.Base
	Painter func(ctx *layout.Context, dst layout.Surface)
}

func (p *painterComponent) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size {
	return cs.Min
}

func (p *painterComponent) Layout(ctx *layout.Context, parent layout.Component, bounds layout.Rect) {
	p.SetFrame(parent, bounds)
}

func (p *painterComponent) DrawTo(ctx *layout.Context, dst layout.Surface) {
	p.Render(ctx, dst)
}

func (p *painterComponent) Render(ctx *layout.Context, dst layout.Surface) {
	if p.Painter != nil {
		p.Painter(ctx, dst)
	}
}
