package layout

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// ---- fakes ----

type fakeSurface struct {
	w, h   int
	clears int
}

func (s *fakeSurface) SizePx() (int, int) { return s.w, s.h }
func (s *fakeSurface) Clear()             { s.clears++ }

type drawCall struct {
	x, y int
	src  *fakeSurface
	dst  *fakeSurface
}

type fillCall struct {
	rect  PxRect
	color Color
	dst   *fakeSurface
}

type tintCall struct {
	rect  PxRect
	color Color
	dst   *fakeSurface
}

type imageCall struct {
	rect PxRect
	dst  *fakeSurface
	img  *fakeImage
}

type fakeRenderer struct {
	news   []*fakeSurface
	draws  []drawCall
	fills  []fillCall
	tints  []tintCall
	images []imageCall
}

func (r *fakeRenderer) NewSurface(w, h int) Surface {
	s := &fakeSurface{w: w, h: h}
	r.news = append(r.news, s)
	return s
}
func (r *fakeRenderer) DrawSurface(dst Surface, src Surface, x, y int) {
	r.draws = append(r.draws, drawCall{x: x, y: y, src: src.(*fakeSurface), dst: dst.(*fakeSurface)})
}
func (r *fakeRenderer) FillRect(dst Surface, rect PxRect, color Color) {
	r.fills = append(r.fills, fillCall{rect: rect, color: color, dst: dst.(*fakeSurface)})
}
func (r *fakeRenderer) TintRect(dst Surface, rect PxRect, color Color) {
	r.tints = append(r.tints, tintCall{rect: rect, color: color, dst: dst.(*fakeSurface)})
}
func (r *fakeRenderer) DrawImage(dst Surface, img Image, rect PxRect) {
	r.images = append(r.images, imageCall{rect: rect, dst: dst.(*fakeSurface), img: img.(*fakeImage)})
}

type roundedFillCall struct {
	rect  PxRect
	radii PxCornerRadii
	color Color
	dst   *fakeSurface
}

type roundedStrokeCall struct {
	rect        PxRect
	radii       PxCornerRadii
	strokeWidth float64
	color       Color
	dst         *fakeSurface
}

type fakeRoundedRenderer struct {
	fakeRenderer
	roundedFills   []roundedFillCall
	roundedTints   []roundedFillCall
	roundedStrokes []roundedStrokeCall
}

func (r *fakeRoundedRenderer) FillRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, color Color) {
	fs := dst.(*fakeSurface)
	r.roundedFills = append(r.roundedFills, roundedFillCall{rect: rect, radii: radii, color: color, dst: fs})
}
func (r *fakeRoundedRenderer) TintRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, color Color) {
	fs := dst.(*fakeSurface)
	r.roundedTints = append(r.roundedTints, roundedFillCall{rect: rect, radii: radii, color: color, dst: fs})
}

func (r *fakeRoundedRenderer) StrokeRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, strokeWidth float64, color Color) {
	fs := dst.(*fakeSurface)
	r.roundedStrokes = append(r.roundedStrokes, roundedStrokeCall{rect: rect, radii: radii, strokeWidth: strokeWidth, color: color, dst: fs})
}

type fakeCanvas struct{ fakeSurface }

type fakeText struct {
	lastText string
	lastMax  int
	w, h     int
}

func (t *fakeText) Measure(text string, style TextStyle, maxWidthPx int) (int, int) {
	t.lastText = text
	t.lastMax = maxWidthPx
	if t.w == 0 && t.h == 0 {
		return 10, 10
	}
	return t.w, t.h
}
func (t *fakeText) Draw(dst Surface, text string, rect PxRect, style TextStyle) {}

type fakeImage struct {
	w, h int
}

func (i *fakeImage) SizePx() (int, int) { return i.w, i.h }

// stub component that embeds Base
type stubComp struct {
	Base
	size    Size
	laid    Rect
	renders int
}

func (c *stubComp) Measure(ctx *Context, cs Constraints) Size { return cs.clamp(c.size) }
func (c *stubComp) Layout(ctx *Context, parent Component, b Rect) {
	c.SetFrame(parent, b)
	c.laid = b
}
func (c *stubComp) DrawTo(ctx *Context, dst Surface) {
	c.Base.DrawCached(ctx, dst, func(s Surface) { c.Render(ctx, s) })
}
func (c *stubComp) Render(ctx *Context, dst Surface) { c.renders++ }

type fillStub struct {
	stubComp
	fillW bool
	fillH bool
}

func (f *fillStub) FillWidth() bool  { return f.fillW }
func (f *fillStub) FillHeight() bool { return f.fillH }

type maxConstraintComp struct {
	stubComp
}

func (c *maxConstraintComp) Measure(ctx *Context, cs Constraints) Size {
	return Size{W: cs.Max.W, H: cs.Max.H}
}

// helper panel wrappers used in tests
type textPanel struct {
	*Panel
}

func newTextPanel(content string, style TextStyle) *textPanel {
	p := NewPanel()
	p.SetText(content)
	p.SetTextStyle(style)
	return &textPanel{Panel: p}
}

func (t *textPanel) Measure(ctx *Context, cs Constraints) Size {
	return t.resolveSize(ctx, cs, Size{})
}

func (t *textPanel) Layout(ctx *Context, parent Component, bounds Rect) {
	t.SetFrame(parent, bounds)
	t.ContentBounds()
}

func (t *textPanel) DrawTo(ctx *Context, dst Surface) {
	t.DrawPanel(ctx, dst, func(surface Surface) { t.Render(ctx, surface) })
}

func (t *textPanel) Render(ctx *Context, dst Surface) {}

type panelWrapper struct {
	*Panel
	Child Component
}

func newPanelWrapper(child Component, padding EdgeInsets) *panelWrapper {
	p := NewPanel()
	p.SetPadding(padding)
	return &panelWrapper{Panel: p, Child: child}
}

func (w *panelWrapper) Measure(ctx *Context, cs Constraints) Size {
	var content Size
	inner := w.contentConstraints(ctx, cs)
	if w.Child != nil {
		content = w.Child.Measure(ctx, inner)
	}
	return w.resolveSize(ctx, cs, content)
}

func (w *panelWrapper) Layout(ctx *Context, parent Component, bounds Rect) {
	w.SetFrame(parent, bounds)
	content := w.ContentBounds()
	if w.Child == nil {
		return
	}
	w.Child.Layout(ctx, w, Rect{X: content.X, Y: content.Y, W: content.W, H: content.H})
	if w.Child.Dirty() {
		w.SetDirty()
	}
}

func (w *panelWrapper) DrawTo(ctx *Context, dst Surface) {
	w.DrawPanelChildren(ctx, dst, func(surface Surface) { w.Render(ctx, surface) })
}

func (w *panelWrapper) Render(ctx *Context, dst Surface) {
	if w.Child != nil {
		w.Child.DrawTo(ctx, dst)
	}
}

func TestRelativePositionOffsets(t *testing.T) {
	child1 := &stubComp{size: Size{W: 20, H: 10}}
	child2 := &stubComp{size: Size{W: 30, H: 12}}
	stack := NewVStack(child1, child2)
	SetPositionMode(child2, PositionRelative)
	SetPositionOffsets(child2, PositionOffsets{
		Top:  PositionValue{Value: 5, Defined: true},
		Left: PositionValue{Value: 4, Defined: true},
	})
	ctx := &Context{Scale: 1}
	stack.Measure(ctx, Infinite())
	stack.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 100})

	if !almostEqual(child1.laid.Y, 0) {
		t.Fatalf("expected first child Y=0, got %.2f", child1.laid.Y)
	}
	if !almostEqual(child2.laid.Y, child1.size.H+5) {
		t.Fatalf("expected second child Y=%.2f, got %.2f", child1.size.H+5, child2.laid.Y)
	}
	if !almostEqual(child2.laid.X, 4) {
		t.Fatalf("expected relative X offset 4, got %.2f", child2.laid.X)
	}
	if !almostEqual(child2.laid.W, child2.size.W) {
		t.Fatalf("expected width %.2f preserved for relative child, got %.2f", child2.size.W, child2.laid.W)
	}
}

func TestAbsolutePositionRemovedFromFlow(t *testing.T) {
	child1 := &stubComp{size: Size{W: 20, H: 10}}
	child2 := &stubComp{size: Size{W: 30, H: 15}}
	child3 := &stubComp{size: Size{W: 25, H: 11}}
	stack := NewVStack(child1, child2, child3)
	SetPositionMode(child2, PositionAbsolute)
	SetPositionOffsets(child2, PositionOffsets{
		Top:  PositionValue{Value: 9, Defined: true},
		Left: PositionValue{Value: 7, Defined: true},
	})
	ctx := &Context{Scale: 1}
	stack.Measure(ctx, Infinite())
	stack.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 120, H: 120})

	if !almostEqual(child3.laid.Y, child1.size.H) {
		t.Fatalf("expected third child Y=%.2f unaffected by absolute sibling, got %.2f", child1.size.H, child3.laid.Y)
	}
	if !almostEqual(child2.laid.X, 7) || !almostEqual(child2.laid.Y, 9) {
		t.Fatalf("expected absolute child at (7,9), got (%.2f, %.2f)", child2.laid.X, child2.laid.Y)
	}
	if !almostEqual(child2.laid.W, child2.size.W) || !almostEqual(child2.laid.H, child2.size.H) {
		t.Fatalf("expected absolute child size preserved, got %.2fx%.2f", child2.laid.W, child2.laid.H)
	}
}

func TestFixedPositionUsesViewport(t *testing.T) {
	child := &stubComp{size: Size{W: 40, H: 20}}
	stack := NewZStack(child)
	SetPositionMode(child, PositionFixed)
	SetPositionOffsets(child, PositionOffsets{
		Top:  PositionValue{Value: 18, Defined: true},
		Left: PositionValue{Value: 12, Defined: true},
	})
	ctx := &Context{Scale: 1}
	ctx.SetViewportSize(Size{W: 200, H: 150})
	stack.Measure(ctx, Tight(Size{W: 120, H: 90}))
	stack.Layout(ctx, nil, Rect{X: 20, Y: 30, W: 120, H: 90})

	if !almostEqual(child.laid.X, 12) || !almostEqual(child.laid.Y, 18) {
		t.Fatalf("expected fixed child at viewport coordinates (12,18), got (%.2f, %.2f)", child.laid.X, child.laid.Y)
	}
	if !almostEqual(child.laid.W, child.size.W) || !almostEqual(child.laid.H, child.size.H) {
		t.Fatalf("expected fixed child size preserved, got %.2fx%.2f", child.laid.W, child.laid.H)
	}
}

func TestZStackRespectsChildSize(t *testing.T) {
	track := &stubComp{size: Size{W: 300, H: 20}}
	fill := &stubComp{size: Size{W: 150, H: 12}}
	stack := NewZStack(track, fill)
	ctx := &Context{Scale: 1}
	stack.Measure(ctx, Infinite())
	stack.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 300, H: 20})

	if !almostEqual(fill.laid.W, fill.size.W) {
		t.Fatalf("expected fill width %.2f, got %.2f", fill.size.W, fill.laid.W)
	}
	if !almostEqual(fill.laid.H, fill.size.H) {
		t.Fatalf("expected fill height %.2f, got %.2f", fill.size.H, fill.laid.H)
	}
	if !almostEqual(fill.laid.X, 0) || !almostEqual(fill.laid.Y, 0) {
		t.Fatalf("expected fill positioned at origin, got (%.2f, %.2f)", fill.laid.X, fill.laid.Y)
	}
}

func TestZStackFillChildStretchesToBounds(t *testing.T) {
	child := &fillStub{stubComp: stubComp{size: Size{W: 80, H: 10}}, fillW: true, fillH: true}
	stack := NewZStack(child)
	ctx := &Context{Scale: 1}
	stack.Measure(ctx, Infinite())
	stack.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 240, H: 60})

	if !almostEqual(child.laid.W, 240) || !almostEqual(child.laid.H, 60) {
		t.Fatalf("expected fill child to stretch to 240x60, got %.2fx%.2f", child.laid.W, child.laid.H)
	}
	if !almostEqual(child.laid.X, 0) || !almostEqual(child.laid.Y, 0) {
		t.Fatalf("expected fill child origin at 0,0, got (%.2f, %.2f)", child.laid.X, child.laid.Y)
	}
}

func TestOrderByZIndex(t *testing.T) {
	a := &stubComp{}
	b := &stubComp{}
	c := &stubComp{}
	SetZIndex(b, 2, true)
	SetZIndex(c, -1, true)
	ordered := orderByZIndex([]Component{a, b, c})
	if len(ordered) != 3 {
		t.Fatalf("expected 3 components, got %d", len(ordered))
	}
	if ordered[0] != c || ordered[1] != a || ordered[2] != b {
		t.Fatalf("unexpected z-index ordering: %#v", ordered)
	}
}

// ---- tests ----

func TestDpPxConversion(t *testing.T) {
	scale := 1.5
	sz := Size{W: 10.5, H: 20.25}
	px := sz.ToPx(scale)
	if px.W != int(math.Round(10.5*1.5)) || px.H != int(math.Round(20.25*1.5)) {
		t.Fatalf("unexpected px size: %+v", px)
	}
	dp := px.ToDp(scale)
	tol := 0.5 / scale
	if math.Abs(dp.W-10.5) > tol || math.Abs(dp.H-20.25) > tol {
		t.Fatalf("unexpected dp size: %+v (tol %.3f)", dp, tol)
	}
}

func TestBaseCacheLifecycle(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	dst := &fakeSurface{w: 300, h: 200}
	c := &stubComp{size: Size{W: 100, H: 50}}
	c.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 50})
	c.DrawTo(ctx, dst)
	if len(r.news) != 1 {
		t.Fatalf("expected one cache surface, got %d", len(r.news))
	}
	if c.renders != 1 {
		t.Fatalf("expected one render, got %d", c.renders)
	}

	c.DrawTo(ctx, dst)
	if len(r.news) != 1 {
		t.Fatalf("unexpected cache recreation")
	}
	if c.renders != 1 {
		t.Fatalf("unexpected re-render without dirty")
	}

	c.SetDirty()
	c.DrawTo(ctx, dst)
	if c.renders != 2 {
		t.Fatalf("expected re-render after dirty")
	}

	ctx.Scale = 2.0
	c.DrawTo(ctx, dst)
	if len(r.news) != 2 {
		t.Fatalf("expected cache recreation on scale change")
	}

	c.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 120, H: 60})
	c.DrawTo(ctx, dst)
	if len(r.news) != 3 {
		t.Fatalf("expected cache recreation on size change")
	}
}

func TestBaseCacheDisabled(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r, DisableCaching: true}
	dst := &fakeSurface{w: 300, h: 200}
	c := &stubComp{size: Size{W: 100, H: 50}}
	c.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 50})

	c.DrawTo(ctx, dst)
	if len(r.news) != 1 {
		t.Fatalf("expected one surface allocation, got %d", len(r.news))
	}
	if c.renders != 1 {
		t.Fatalf("expected one render, got %d", c.renders)
	}
	if c.cache != nil {
		t.Fatalf("expected cache to be released when caching is disabled")
	}

	c.DrawTo(ctx, dst)
	if len(r.news) != 2 {
		t.Fatalf("expected a new surface allocation, got %d", len(r.news))
	}
	if c.renders != 2 {
		t.Fatalf("expected a second render, got %d", c.renders)
	}
}

func TestPanelChildrenSkipCacheWithoutVisuals(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	dst := &fakeSurface{w: 200, h: 100}

	a := &stubComp{size: Size{W: 20, H: 10}}
	b := &stubComp{size: Size{W: 30, H: 15}}
	row := NewHStack(a, b)
	row.Layout(ctx, nil, Rect{W: 60, H: 20})

	row.DrawTo(ctx, dst)
	if len(r.news) != 2 {
		t.Fatalf("expected only child caches when stack has no visuals, got %d", len(r.news))
	}

	row.SetBackgroundColor(Color{A: 255})
	row.DrawTo(ctx, dst)
	if len(r.news) != 2 {
		t.Fatalf("expected background draw without stack cache, got %d", len(r.news))
	}
}

func TestPanelNoCacheDisablesOwnCacheOnly(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	dst := &fakeSurface{w: 200, h: 100}

	child := &stubComp{size: Size{W: 20, H: 10}}
	panel := NewPanelComponent(child)
	panel.SetText("hello")
	panel.Layout(ctx, nil, Rect{W: 40, H: 20})

	panel.DrawTo(ctx, dst)
	if len(r.news) != 2 {
		t.Fatalf("expected parent and child caches, got %d", len(r.news))
	}
	panel.DrawTo(ctx, dst)
	if len(r.news) != 2 {
		t.Fatalf("expected cache reuse before nocache toggle, got %d", len(r.news))
	}

	panel.SetCacheEnabled(false)
	panel.DrawTo(ctx, dst)
	if len(r.news) != 3 {
		t.Fatalf("expected new parent surface after nocache, got %d", len(r.news))
	}
	panel.DrawTo(ctx, dst)
	if len(r.news) != 4 {
		t.Fatalf("expected new parent surface each draw when nocache, got %d", len(r.news))
	}
}

func TestPanelRoundedDecorationsUseCache(t *testing.T) {
	r := &fakeRoundedRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	dst := &fakeSurface{w: 200, h: 100}

	panel := NewPanelComponent(nil)
	panel.SetBackgroundColor(Color{R: 12, G: 24, B: 36, A: 255})
	panel.SetCornerRadius(6)
	panel.Layout(ctx, nil, Rect{W: 60, H: 30})

	panel.DrawTo(ctx, dst)
	if len(r.news) != 1 {
		t.Fatalf("expected cache allocation for rounded panel, got %d", len(r.news))
	}
	if len(r.roundedFills) != 1 {
		t.Fatalf("expected one rounded fill, got %d", len(r.roundedFills))
	}

	panel.DrawTo(ctx, dst)
	if len(r.roundedFills) != 1 {
		t.Fatalf("expected rounded fill to be cached, got %d", len(r.roundedFills))
	}
}

func TestMeasurementOnlyPanelsStayClean(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	child := newPanelWrapper(nil, EdgeInsets{})
	child.SetMeasurementOnly(true)
	parent := newPanelWrapper(child, EdgeInsets{})

	parent.Measure(ctx, Tight(Size{W: 80, H: 20}))
	parent.Layout(ctx, nil, Rect{W: 80, H: 20})
	parent.DrawTo(ctx, &fakeSurface{w: 160, h: 40})

	if parent.Dirty() {
		t.Fatalf("expected parent clean after draw")
	}
	if len(r.news) != 0 {
		t.Fatalf("expected no caches created when drawing measurement-only wrapper, got %d", len(r.news))
	}

	child.SetDirty()
	parent.Layout(ctx, nil, Rect{W: 80, H: 20})

	if child.Dirty() {
		t.Fatalf("expected measurement-only panel to report clean even when marked dirty")
	}
	if parent.Dirty() {
		t.Fatalf("expected parent to stay clean when only measurement-only child is dirty")
	}
}

func TestPanelChildrenRespectParentOffset(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	dst := &fakeSurface{w: 120, h: 80}

	child := &stubComp{size: Size{W: 10, H: 8}}
	panel := NewPanelComponent(child)
	panel.SetPadding(Insets(3))
	panel.Layout(ctx, nil, Rect{X: 12, Y: 20, W: 40, H: 30})

	panel.DrawTo(ctx, dst)

	if len(r.news) != 1 {
		t.Fatalf("expected only child cache, got %d", len(r.news))
	}
	if len(r.draws) != 1 {
		t.Fatalf("expected one draw call for child, got %d", len(r.draws))
	}
	call := r.draws[0]
	if call.x != 15 || call.y != 23 {
		t.Fatalf("expected child draw at (15,23), got (%d,%d)", call.x, call.y)
	}
}

func TestVisibilityHiddenSkipsDraw(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	dst := &fakeSurface{w: 120, h: 80}
	c := &stubComp{size: Size{W: 40, H: 30}}
	c.SetVisibility(VisibilityHidden)
	c.Layout(ctx, nil, Rect{W: 40, H: 30})
	c.DrawTo(ctx, dst)
	if len(r.news) != 0 {
		t.Fatalf("expected no cache surfaces for hidden component, got %d", len(r.news))
	}
	if c.renders != 0 {
		t.Fatalf("expected no renders for hidden component, got %d", c.renders)
	}
}

func TestHStackVisibilityCollapse(t *testing.T) {
	ctx := &Context{}
	a := &stubComp{size: Size{W: 40, H: 10}}
	b := &stubComp{size: Size{W: 30, H: 20}}
	c := &stubComp{size: Size{W: 20, H: 15}}
	stack := NewHStack(a, b, c)
	stack.Spacing = 5

	constraints := Constraints{Max: Size{W: 500, H: 500}}
	sz := stack.Measure(ctx, constraints)
	if !almostEqual(sz.W, 40+30+20+5*2) {
		t.Fatalf("expected width 100, got %.2f", sz.W)
	}
	if !almostEqual(sz.H, 20) {
		t.Fatalf("expected height 20, got %.2f", sz.H)
	}
	stack.Layout(ctx, nil, Rect{W: sz.W, H: sz.H})

	b.SetVisibility(VisibilityCollapse)

	szCollapsed := stack.Measure(ctx, constraints)
	if !almostEqual(szCollapsed.W, 40+20+5) {
		t.Fatalf("expected collapsed width 65, got %.2f", szCollapsed.W)
	}
	if !almostEqual(szCollapsed.H, 15) {
		t.Fatalf("expected collapsed height 15, got %.2f", szCollapsed.H)
	}

	stack.Layout(ctx, nil, Rect{W: szCollapsed.W, H: szCollapsed.H})

	if !almostEqual(b.Bounds().W, 0) || !almostEqual(b.Bounds().H, 0) {
		t.Fatalf("expected collapsed child to have zero bounds, got %+v", b.Bounds())
	}
	if !almostEqual(c.Bounds().X, 40+5) {
		t.Fatalf("expected third child x=45, got %.2f", c.Bounds().X)
	}
}

func TestPanelBorderEdges(t *testing.T) {
	panel := NewPanel()
	panel.SetBorder(Color{R: 255, G: 0, B: 0, A: 255}, 2)
	panel.SetBorderTopWidth(4)
	panel.SetBorderRightColor(Color{R: 0, G: 255, B: 0, A: 255})
	panel.SetBorderBottom(3, Color{R: 0, G: 0, B: 255, A: 255})
	panel.ClearBorderLeft()

	if !almostEqual(panel.BorderTopWidth(), 4) {
		t.Fatalf("expected top width 4, got %.2f", panel.BorderTopWidth())
	}
	if !almostEqual(panel.BorderBottomWidth(), 3) {
		t.Fatalf("expected bottom width 3, got %.2f", panel.BorderBottomWidth())
	}
	if panel.BorderLeftWidth() != 0 {
		t.Fatalf("expected left width cleared")
	}
	topColor := panel.BorderTopColor()
	if topColor != (Color{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("unexpected top color %+v", topColor)
	}
	rightColor := panel.BorderRightColor()
	if rightColor != (Color{R: 0, G: 255, B: 0, A: 255}) {
		t.Fatalf("unexpected right color %+v", rightColor)
	}
	if !panel.hasBorder {
		t.Fatalf("expected panel to still report border")
	}
	panel.ClearBorder()
	if panel.hasBorder {
		t.Fatalf("expected clear border to remove borders")
	}
}

func TestPanelCornerRadiiSetter(t *testing.T) {
	r := &fakeRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	panel := NewPanel()
	panel.SetFrame(nil, Rect{W: 40, H: 20})
	dst := &fakeSurface{w: 80, h: 40}

	panel.DrawPanel(ctx, dst, nil)
	if panel.Dirty() {
		t.Fatalf("expected clean panel after draw")
	}

	panel.SetCornerRadii(CornerRadii{TopLeft: -2, TopRight: 6})
	if !panel.Dirty() {
		t.Fatalf("expected dirty after changing corner radii")
	}
	radii := panel.CornerRadii()
	if radii.TopLeft != 0 || !almostEqual(radii.TopRight, 6) {
		t.Fatalf("unexpected sanitized radii %+v", radii)
	}

	panel.DrawPanel(ctx, dst, nil)
	if panel.Dirty() {
		t.Fatalf("expected clean panel after redraw")
	}

	panel.SetCornerRadius(6)
	if !panel.Dirty() {
		t.Fatalf("expected dirty after updating radius")
	}

	panel.DrawPanel(ctx, dst, nil)
	if panel.Dirty() {
		t.Fatalf("expected clean panel after redraw")
	}

	panel.SetCornerRadius(6)
	if panel.Dirty() {
		t.Fatalf("expected no dirty when radius unchanged")
	}
}

func TestPanelRoundedBackgroundUsesRoundedRenderer(t *testing.T) {
	r := &fakeRoundedRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	panel := NewPanel()
	panel.SetBackgroundColor(Color{R: 10, G: 20, B: 30, A: 255})
	panel.SetCornerRadius(5)
	panel.SetFrame(nil, Rect{W: 50, H: 30})
	dst := &fakeSurface{w: 100, h: 60}

	panel.DrawPanel(ctx, dst, nil)

	if len(r.roundedFills) != 1 {
		t.Fatalf("expected one rounded fill, got %d", len(r.roundedFills))
	}
	if len(r.fills) != 0 {
		t.Fatalf("expected no rectangular fills when rounded renderer present")
	}
	call := r.roundedFills[0]
	if call.rect.W != 50 || call.rect.H != 30 {
		t.Fatalf("unexpected rounded fill rect %+v", call.rect)
	}
	if !almostEqual(call.radii.TopLeft, 5) || !almostEqual(call.radii.TopRight, 5) {
		t.Fatalf("unexpected rounded radii %+v", call.radii)
	}
	if call.color != (Color{R: 10, G: 20, B: 30, A: 255}) {
		t.Fatalf("unexpected fill color %+v", call.color)
	}
}

func TestPanelRoundedBorderUsesRoundedRenderer(t *testing.T) {
	r := &fakeRoundedRenderer{}
	ctx := &Context{Scale: 1.0, Renderer: r}
	panel := NewPanel()
	panel.SetBackgroundColor(Color{R: 200, G: 200, B: 200, A: 255})
	panel.SetCornerRadius(8)
	panel.SetBorder(Color{R: 50, G: 60, B: 70, A: 255}, 3)
	panel.SetFrame(nil, Rect{W: 60, H: 40})
	dst := &fakeSurface{w: 120, h: 80}

	panel.DrawPanel(ctx, dst, nil)

	if len(r.roundedStrokes) != 1 {
		t.Fatalf("expected one rounded stroke, got %d", len(r.roundedStrokes))
	}
	if len(r.roundedFills) != 1 {
		t.Fatalf("expected background rounded fill before stroke")
	}
	if len(r.fills) != 0 {
		t.Fatalf("expected no rectangular fills with rounded renderer")
	}
	call := r.roundedStrokes[0]
	if call.rect.W != 60 || call.rect.H != 40 {
		t.Fatalf("unexpected stroke rect %+v", call.rect)
	}
	if !almostEqual(call.radii.TopLeft, 8) || !almostEqual(call.radii.BottomRight, 8) {
		t.Fatalf("unexpected stroke radii %+v", call.radii)
	}
	if !almostEqual(call.strokeWidth, 3) {
		t.Fatalf("unexpected stroke width %.2f", call.strokeWidth)
	}
	if call.color != (Color{R: 50, G: 60, B: 70, A: 255}) {
		t.Fatalf("unexpected stroke color %+v", call.color)
	}
}

func TestPanelPaddingEdges(t *testing.T) {
	panel := NewPanel()
	panel.SetPadding(EdgeInsets{Top: 4, Right: 4, Bottom: 4, Left: 4})
	panel.SetPaddingTop(10)
	panel.SetPaddingRight(6)
	panel.SetPaddingBottom(2)
	panel.SetPaddingLeft(8)
	if !almostEqual(panel.Padding.Top, 10) || !almostEqual(panel.Padding.Right, 6) || !almostEqual(panel.Padding.Bottom, 2) || !almostEqual(panel.Padding.Left, 8) {
		t.Fatalf("unexpected padding %+v", panel.Padding)
	}
	panel.SetPadding(EdgeInsets{Top: 1, Right: 1, Bottom: 1, Left: 1})
	if !almostEqual(panel.Padding.Top, 1) || !almostEqual(panel.Padding.Left, 1) {
		t.Fatalf("set padding should reset overrides %+v", panel.Padding)
	}
}
func TestLengthResolveViewport(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	ctx.SetViewportSize(Size{W: 320, H: 480})
	if got := LengthDP(45).ResolveWidth(ctx, 0); got != 45 {
		t.Fatalf("expected 45dp, got %.2f", got)
	}
	if got := LengthPercent(0.25).ResolveWidth(ctx, 400); !almostEqual(got, 100) {
		t.Fatalf("expected 100dp from percent, got %.2f", got)
	}
	if got := LengthVW(0.5).ResolveWidth(ctx, 0); !almostEqual(got, 160) {
		t.Fatalf("expected 160dp from viewport width, got %.2f", got)
	}
	if got := LengthVH(0.5).ResolveHeight(ctx, 0); !almostEqual(got, 240) {
		t.Fatalf("expected 240dp from viewport height, got %.2f", got)
	}
	if got := LengthVMin(0.5).ResolveWidth(ctx, 0); !almostEqual(got, 160) {
		t.Fatalf("expected 160dp from viewport min, got %.2f", got)
	}
	if got := LengthVMax(0.5).ResolveWidth(ctx, 0); !almostEqual(got, 240) {
		t.Fatalf("expected 240dp from viewport max, got %.2f", got)
	}
	if got := LengthAuto().ResolveWidth(ctx, 200); got != 0 {
		t.Fatalf("auto length should resolve to 0, got %.2f", got)
	}
}

func TestPanelMinWidthViewport(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	ctx.SetViewportSize(Size{W: 200, H: 400})
	panel := NewPanelComponent(nil)
	panel.SetMinWidthLength(LengthVW(0.5))
	size := panel.Measure(ctx, Constraints{Max: Size{W: 500, H: 200}})
	if !almostEqual(size.W, 100) {
		t.Fatalf("expected measured width 100dp, got %.2f", size.W)
	}
}

func TestPanelFixedWidthHeight(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	panel := NewPanelComponent(nil)
	panel.SetWidth(120)
	panel.SetHeight(60)
	size := panel.Measure(ctx, Constraints{})
	if !almostEqual(size.W, 120) || !almostEqual(size.H, 60) {
		t.Fatalf("expected fixed size 120x60, got %+v", size)
	}
	panel.SetWidthPercent(0) // clears
	panel.SetHeight(0)
	size = panel.Measure(ctx, Constraints{})
	if size.W != 0 || size.H != 0 {
		t.Fatalf("expected cleared width/height to fall back to zero, got %+v", size)
	}
}

func TestGridCellMaxWidthViewport(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	ctx.SetViewportSize(Size{W: 400, H: 400})
	grid := NewGrid(0, 0)
	for i := 0; i < 4; i++ {
		grid.Add(&stubComp{size: Size{W: 220, H: 40}})
	}
	grid.SetCellMaxWidthLength(LengthVW(0.25)) // 0.25 * 400 = 100
	size := grid.Measure(ctx, Constraints{Max: Size{W: 400, H: 400}})
	if !almostEqual(size.W, 200) {
		t.Fatalf("expected grid width 200dp, got %.2f", size.W)
	}
	grid.Layout(ctx, nil, Rect{X: 0, Y: 0, W: size.W, H: size.H})
	for i, child := range grid.Children {
		stub, ok := child.(*stubComp)
		if !ok {
			t.Fatalf("child %d is not stub component", i)
		}
		if !almostEqual(stub.laid.W, 100) {
			t.Fatalf("expected child width 100dp, got %.2f", stub.laid.W)
		}
	}
}

func TestGridMeasureRespectsAvailableWidth(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	grid := NewGrid(0, 0)
	for i := 0; i < 4; i++ {
		grid.Add(&stubComp{size: Size{W: 60, H: 10}})
	}
	size := grid.Measure(ctx, Constraints{Max: Size{W: 100, H: 200}})
	if !almostEqual(size.W, 60) {
		t.Fatalf("expected grid width 60dp, got %.2f", size.W)
	}
}

func TestHStackLayoutWithSpacer(t *testing.T) {
	ctx := &Context{Scale: 1.0, Renderer: &fakeRenderer{}}
	a := &stubComp{size: Size{W: 30, H: 10}}
	b := &stubComp{size: Size{W: 20, H: 10}}
	row := NewHStack(a, NewSpacer(1), b)
	row.Spacing = 5
	row.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 20})
	if a.laid.X != 0 || a.laid.W != 30 {
		t.Fatalf("a laid %+v", a.laid)
	}
	if b.laid.X != 80 || b.laid.W != 20 {
		t.Fatalf("b laid %+v", b.laid)
	}
}

func TestHStackJustifySpaceBetweenFlex(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	fixed := &stubComp{size: Size{W: 20, H: 10}}
	flex := &maxConstraintComp{}
	flex.SetFlexWeight(1)
	row := NewHStack(fixed, flex)
	row.Spacing = 10
	row.Justify = JustifySpaceBetween
	row.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 20})
	if !almostEqual(flex.laid.W, 70) {
		t.Fatalf("expected flex width 70dp, got %.2f", flex.laid.W)
	}
}

func TestVStackJustifySpaceBetweenFlex(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	fixed := &stubComp{size: Size{W: 20, H: 10}}
	flex := &maxConstraintComp{}
	flex.SetFlexWeight(1)
	stack := NewVStack(fixed, flex)
	stack.Spacing = 10
	stack.Justify = JustifySpaceBetween
	stack.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 100})
	if !almostEqual(flex.laid.H, 80) {
		t.Fatalf("expected flex height 80dp, got %.2f", flex.laid.H)
	}
}

func TestHStackLayoutsOutOfFlowChildren(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	abs := &stubComp{size: Size{W: 10, H: 5}}
	SetPositionMode(abs, PositionAbsolute)
	row := NewHStack(abs)
	row.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 100, H: 20})
	if !almostEqual(abs.laid.W, 10) || !almostEqual(abs.laid.H, 5) {
		t.Fatalf("expected absolute child 10x5, got %+v", abs.laid)
	}
}

func TestGridLayoutsOutOfFlowChildren(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	abs := &stubComp{size: Size{W: 12, H: 6}}
	SetPositionMode(abs, PositionAbsolute)
	grid := NewGrid(0, 0)
	grid.Add(abs)
	grid.Layout(ctx, nil, Rect{X: 0, Y: 0, W: 120, H: 60})
	if !almostEqual(abs.laid.W, 12) || !almostEqual(abs.laid.H, 6) {
		t.Fatalf("expected absolute child 12x6, got %+v", abs.laid)
	}
}

func TestPanelAspectRatio(t *testing.T) {
	ctx := &Context{Scale: 1.0}
	panel := NewPanelComponent(nil)
	panel.SetAspectRatio(2.0) // width:height = 2:1
	size := panel.Measure(ctx, Constraints{Max: Size{W: 300, H: 300}})
	if !almostEqual(size.W, 300) {
		t.Fatalf("expected width 300, got %.2f", size.W)
	}
	if !almostEqual(size.H, 150) {
		t.Fatalf("expected height 150, got %.2f", size.H)
	}

	panel2 := NewPanelComponent(nil)
	panel2.SetAspectRatio(0.5) // width:height = 0.5:1 => height twice width
	panel2.SetMinWidth(80)
	size2 := panel2.Measure(ctx, Constraints{})
	if !almostEqual(size2.W, 80) {
		t.Fatalf("expected width 80, got %.2f", size2.W)
	}
	if !almostEqual(size2.H, 160) {
		t.Fatalf("expected height 160, got %.2f", size2.H)
	}
}

func TestPanelTextMaxWidth(t *testing.T) {
	fr := &fakeRenderer{}
	ft := &fakeText{w: 100, h: 20}
	ctx := &Context{Scale: 1.5, Renderer: fr, Text: ft}
	label := newTextPanel("hello", TextStyle{SizeDp: 10})
	label.SetTextMaxWidth(50)
	sz := label.Measure(ctx, Constraints{Max: Size{W: 100, H: 100}})
	label.Layout(ctx, nil, Rect{W: sz.W, H: sz.H})
	label.DrawTo(ctx, &fakeSurface{w: 200, h: 100})
	expected := int(math.Round(50 * 1.5))
	if ft.lastMax != expected {
		t.Fatalf("expected maxWidthPx=%d, got %d", expected, ft.lastMax)
	}
}

func TestGlobalBoundsPropagation(t *testing.T) {
	ctx := &Context{Scale: 1.0, Renderer: &fakeRenderer{}}
	child := &stubComp{size: Size{W: 20, H: 10}}
	panel := newPanelWrapper(child, Insets(4))

	panel.Layout(ctx, nil, Rect{X: 10, Y: 20, W: 60, H: 40})

	childLocal := child.Bounds()
	if childLocal.X != 4 || childLocal.Y != 4 || childLocal.W != 52 || childLocal.H != 32 {
		t.Fatalf("unexpected child local bounds %+v", childLocal)
	}

	childGlobal := child.GlobalBounds()
	if childGlobal.X != 14 || childGlobal.Y != 24 || childGlobal.W != 52 || childGlobal.H != 32 {
		t.Fatalf("unexpected child global bounds %+v", childGlobal)
	}
}

func TestFlowStackWeightsDistributeExtraSpace(t *testing.T) {
	a := &stubComp{size: Size{W: 60, H: 10}}
	b := &stubComp{size: Size{W: 40, H: 10}}
	c := &stubComp{size: Size{W: 40, H: 10}}
	SetFlexWeight(b, 1)
	SetFlexWeight(c, 2)

	stack := NewFlowStack(a, b, c)
	stack.Spacing = 0

	ctx := NewContext(nil, nil, nil)
	bounds := Rect{W: 300, H: 50}
	stack.Layout(ctx, nil, bounds)

	if !almostEqual(a.laid.W, 60) {
		t.Fatalf("expected fixed child width 60, got %.2f", a.laid.W)
	}
	leftover := 300.0 - (60.0 + 40.0 + 40.0)
	expectedB := 40.0 + leftover*(1.0/3.0)
	expectedC := 40.0 + leftover*(2.0/3.0)
	if !almostEqual(b.laid.W, expectedB) {
		t.Fatalf("expected weight child width %.2f, got %.2f", expectedB, b.laid.W)
	}
	if !almostEqual(c.laid.W, expectedC) {
		t.Fatalf("expected weight child width %.2f, got %.2f", expectedC, c.laid.W)
	}
}

func TestGridAlignSelfOverridesAlignment(t *testing.T) {
	g := NewGrid(2, 0)
	a := &stubComp{size: Size{W: 40, H: 10}}
	b := &stubComp{size: Size{W: 200, H: 50}}
	g.Add(a)
	g.Add(b)
	g.AlignH = AlignStart
	g.AlignV = AlignStart
	SetAlignSelf(a, AlignEnd, true)

	ctx := NewContext(nil, nil, nil)
	bounds := Rect{W: 400, H: 200}
	g.Layout(ctx, nil, bounds)

	if !almostEqual(a.laid.X, 160) {
		t.Fatalf("expected align-self end to move child to x=160, got %.2f", a.laid.X)
	}
	if !almostEqual(a.laid.Y, 40) {
		t.Fatalf("expected align-self end to move child to y=40, got %.2f", a.laid.Y)
	}
	if !almostEqual(a.laid.W, 40) {
		t.Fatalf("expected child width 40, got %.2f", a.laid.W)
	}
	if !almostEqual(a.laid.H, 10) {
		t.Fatalf("expected child height 10, got %.2f", a.laid.H)
	}
}

func TestGridWeightsExpandShortRow(t *testing.T) {
	g := NewGrid(3, 0)
	children := make([]*stubComp, 4)
	for i := range children {
		children[i] = &stubComp{size: Size{W: 80, H: 20}}
		g.Add(children[i])
	}
	SetFlexWeight(children[3], 1)

	ctx := NewContext(nil, nil, nil)
	bounds := Rect{W: 240, H: 200}
	g.Layout(ctx, nil, bounds)

	for i := 0; i < 3; i++ {
		if !almostEqual(children[i].laid.W, 80) {
			t.Fatalf("expected first row child width 80, got %.2f", children[i].laid.W)
		}
	}
	if !almostEqual(children[3].laid.W, 240) {
		t.Fatalf("expected weighted child to expand to 240 width, got %.2f", children[3].laid.W)
	}
	if !almostEqual(children[3].laid.X, 0) {
		t.Fatalf("expected weighted child to start at x=0, got %.2f", children[3].laid.X)
	}
}

func TestDialogVariableResolution(t *testing.T) {
	parent := NewPanelComponent(nil)
	child := NewPanelComponent(nil)
	child.PanelRef().SetText("Hello, {{Name}}!")
	parent.SetChild(child)

	SetDialogVariable(parent, "Name", "Joseph")
	if got := child.PanelRef().Text(); got != "Hello, Joseph!" {
		t.Fatalf("expected inherited variable, got %q", got)
	}

	SetDialogVariable(child, "Name", "Ada")
	if got := child.PanelRef().Text(); got != "Hello, Ada!" {
		t.Fatalf("expected child override, got %q", got)
	}

	ClearDialogVariable(child, "Name")
	if got := child.PanelRef().Text(); got != "Hello, Joseph!" {
		t.Fatalf("expected fallback to parent value, got %q", got)
	}

	ClearDialogVariable(parent, "Name")
	if got := child.PanelRef().Text(); got != "Hello, !" {
		t.Fatalf("expected empty string after clearing variable, got %q", got)
	}
}
