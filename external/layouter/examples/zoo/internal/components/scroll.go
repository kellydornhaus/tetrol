package components

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	"github.com/kellydornhaus/layouter/layout"
)

// Scroll is a vertical scrolling container that clips its child to the
// allocated bounds and allows manual offset adjustments.
type Scroll struct {
	*layout.Panel
	Child layout.Component

	TailPadDp float64

	offsetDp     float64
	contentSize  layout.Size
	viewportSize layout.Size

	childSurface layout.Surface
	childScale   float64

	minWidth  float64
	minHeight float64
}

func NewScroll(child layout.Component) *Scroll {
	p := layout.NewPanel()
	p.SetFillWidth(true)
	return &Scroll{Panel: p, Child: child, TailPadDp: 24}
}

func (s *Scroll) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size {
	padW := s.Padding.Left + s.Padding.Right
	padH := s.Padding.Top + s.Padding.Bottom

	if s.Child == nil {
		s.contentSize = layout.Size{}
		s.viewportSize = layout.Size{}
		width := padW
		height := padH
		if s.minWidth > 0 && width < s.minWidth {
			width = s.minWidth
		}
		if s.minHeight > 0 && height < s.minHeight {
			height = s.minHeight
		}
		return clampSizeToConstraints(layout.Size{W: width, H: height}, cs)
	}

	inner := adjustConstraintsForPadding(cs, padW, padH, s.minWidth, s.minHeight)

	childCs := inner
	childCs.Max.H = 0 // unbounded vertically so child reports full height
	childSize := s.Child.Measure(ctx, childCs)

	viewport := childSize
	if inner.Max.W > 0 && viewport.W > inner.Max.W {
		viewport.W = inner.Max.W
	}
	if inner.Max.H > 0 && viewport.H > inner.Max.H {
		viewport.H = inner.Max.H
	}
	if viewport.W < inner.Min.W {
		viewport.W = inner.Min.W
	}
	if viewport.H < inner.Min.H {
		viewport.H = inner.Min.H
	}

	s.contentSize = childSize
	s.viewportSize = viewport
	s.clampOffset()

	width := viewport.W + padW
	height := viewport.H + padH
	if s.minWidth > 0 && width < s.minWidth {
		width = s.minWidth
	}
	if s.minHeight > 0 && height < s.minHeight {
		height = s.minHeight
	}
	return clampSizeToConstraints(layout.Size{W: width, H: height}, cs)
}

func (s *Scroll) Layout(ctx *layout.Context, parent layout.Component, bounds layout.Rect) {
	s.SetFrame(parent, bounds)
	content := s.ContentBounds()
	s.viewportSize = layout.Size{W: content.W, H: content.H}
	if s.Child == nil {
		return
	}
	childHeight := s.contentSize.H
	if childHeight < content.H {
		childHeight = content.H
	}
	s.Child.Layout(ctx, s, layout.Rect{X: 0, Y: 0, W: content.W, H: childHeight})
	s.clampOffset()
	if s.Child.Dirty() {
		s.SetDirty()
	}
}

func (s *Scroll) DrawTo(ctx *layout.Context, dst layout.Surface) {
	if s.Child != nil && s.Child.Dirty() {
		s.SetDirty()
	}
	s.DrawPanel(ctx, dst, func(target layout.Surface) { s.renderContent(ctx, target) })
}

func (s *Scroll) prepareCache(ctx *layout.Context) *ebiten.Image {
	px := s.contentSize.ToPx(ctx.Scale)
	if px.W <= 0 || px.H <= 0 {
		return nil
	}
	s.ensureChildSurface(ctx, px.W, px.H)
	s.childSurface.Clear()
	s.Child.DrawTo(ctx, s.childSurface)
	if surf, ok := s.childSurface.(*adp.Surface); ok {
		return surf.Img
	}
	return nil
}

func (s *Scroll) renderContent(ctx *layout.Context, dst layout.Surface) {
	if s.Child == nil {
		return
	}
	vpSurface, ok := dst.(*adp.Surface)
	if !ok {
		s.Child.DrawTo(ctx, dst)
		return
	}

	childImg := s.prepareCache(ctx)
	if childImg == nil {
		return
	}

	dstImg := vpSurface.Img
	vw, vh := dstImg.Size()
	if vw <= 0 || vh <= 0 {
		return
	}

	tailPx := int(math.Round(s.TailPadDp * ctx.Scale))
	padLeftPx := int(math.Round(s.Padding.Left * ctx.Scale))
	padTopPx := int(math.Round(s.Padding.Top * ctx.Scale))
	padBottomPx := int(math.Round(s.Padding.Bottom * ctx.Scale))
	visibleHeightPx := int(math.Round(s.viewportSize.H * ctx.Scale))
	if visibleHeightPx <= 0 {
		visibleHeightPx = vh - padTopPx - padBottomPx
	}
	if visibleHeightPx < 0 {
		visibleHeightPx = 0
	}
	maxOffsetPx := childImg.Bounds().Dy() - visibleHeightPx + tailPx
	if maxOffsetPx < 0 {
		maxOffsetPx = 0
	}
	offsetPx := int(math.Round(s.offsetDp * ctx.Scale))
	if offsetPx < 0 {
		offsetPx = 0
	}
	if offsetPx > maxOffsetPx {
		offsetPx = maxOffsetPx
	}

	dstImg.Clear()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(padLeftPx), float64(padTopPx-offsetPx))
	dstImg.DrawImage(childImg, op)
}

func (s *Scroll) Render(ctx *layout.Context, dst layout.Surface) {
	s.renderContent(ctx, dst)
}

func (s *Scroll) ensureChildSurface(ctx *layout.Context, w, h int) {
	if s.childSurface != nil {
		if s.childScale != ctx.Scale {
			s.childSurface = nil
		} else if cw, ch := s.childSurface.SizePx(); cw != w || ch != h {
			s.childSurface = nil
		}
	}
	if s.childSurface == nil {
		s.childSurface = ctx.Renderer.NewSurface(w, h)
		s.childScale = ctx.Scale
		s.SetDirty()
	}
}

// ScrollBy scrolls vertically by the provided delta in dp units.
func (s *Scroll) ScrollBy(deltaDp float64) {
	if deltaDp == 0 {
		return
	}
	s.offsetDp += deltaDp
	s.clampOffset()
	s.SetDirty()
}

func (s *Scroll) SnapshotImage(ctx *layout.Context) *ebiten.Image {
	return s.prepareCache(ctx)
}

func (s *Scroll) Offset() float64 { return s.offsetDp }

func (s *Scroll) ViewportSize() layout.Size { return s.viewportSize }

func (s *Scroll) ContentSize() layout.Size { return s.contentSize }

func (s *Scroll) TailPad() float64 { return s.TailPadDp }

func (s *Scroll) clampOffset() {
	maxOffset := s.contentSize.H - s.viewportSize.H + s.TailPadDp
	if maxOffset < 0 {
		maxOffset = 0
	}
	if s.offsetDp < 0 {
		s.offsetDp = 0
	}
	if s.offsetDp > maxOffset {
		s.offsetDp = maxOffset
	}
}

func (s *Scroll) FillWidth() bool { return s.Panel.FillWidth() }

func (s *Scroll) SetMinWidth(width float64) {
	s.Panel.SetMinWidth(width)
	s.minWidth = width
}

func (s *Scroll) SetMinHeight(height float64) {
	s.Panel.SetMinHeight(height)
	s.minHeight = height
}

func clampSizeToConstraints(sz layout.Size, cs layout.Constraints) layout.Size {
	w := sz.W
	h := sz.H
	if cs.Max.W > 0 && w > cs.Max.W {
		w = cs.Max.W
	}
	if cs.Max.H > 0 && h > cs.Max.H {
		h = cs.Max.H
	}
	if w < cs.Min.W {
		w = cs.Min.W
	}
	if h < cs.Min.H {
		h = cs.Min.H
	}
	return layout.Size{W: w, H: h}
}

func adjustConstraintsForPadding(cs layout.Constraints, padW, padH, minW, minH float64) layout.Constraints {
	inner := cs
	if inner.Min.W > 0 {
		inner.Min.W = math.Max(0, inner.Min.W-padW)
	}
	if inner.Min.H > 0 {
		inner.Min.H = math.Max(0, inner.Min.H-padH)
	}
	if minW > 0 {
		minInner := math.Max(0, minW-padW)
		if minInner > inner.Min.W {
			inner.Min.W = minInner
		}
	}
	if minH > 0 {
		minInner := math.Max(0, minH-padH)
		if minInner > inner.Min.H {
			inner.Min.H = minInner
		}
	}
	if inner.Max.W > 0 {
		inner.Max.W = math.Max(0, inner.Max.W-padW)
	}
	if inner.Max.H > 0 {
		inner.Max.H = math.Max(0, inner.Max.H-padH)
	}
	return inner
}
