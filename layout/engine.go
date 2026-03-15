package layout

// LayoutAndDraw lays out the root to fill the canvas and draws it.
// It converts the canvas pixel size into logical dp using the context scale.
func LayoutAndDraw(ctx *Context, root Component, canvas Canvas) {
	if ctx == nil || root == nil || canvas == nil || ctx.Renderer == nil {
		return
	}
	ctx.BeginFrameLogAuto()
	defer ctx.EndFrameLog()
	w, h := canvas.SizePx()
	dpSize := PxSize{W: w, H: h}.ToDp(ctx.Scale)
	prevSize := ctx.ViewportSize()
	prevHad := ctx.HasViewport()
	ctx.SetViewportSize(Size{W: dpSize.W, H: dpSize.H})
	defer func() {
		if prevHad {
			ctx.SetViewportSize(prevSize)
		} else {
			ctx.SetViewportSize(Size{})
		}
	}()
	// Give the root the full canvas.
	_ = measureComponent(ctx, nil, root, Tight(dpSize))
	rootBounds := Rect{X: 0, Y: 0, W: dpSize.W, H: dpSize.H}
	logLayoutBounds(ctx, root, nil, rootBounds)
	root.Layout(ctx, nil, rootBounds)
	// Draw the tree into the canvas.
	root.DrawTo(ctx, canvas)
}
