package layout

// Context is provided to components for measurement, layout, and drawing.
// It provides access to device pixel scale, rendering, and text services.
type Context struct {
	// Scale is the device scale factor (logical dp -> pixels).
	Scale float64
	// Renderer supplies offscreen creation and compositing.
	Renderer Renderer
	// Text is a text shaper/renderer (backed by etxt via adapter).
	Text TextEngine
	// Debug controls optional logging hooks for layout/styling.
	Debug DebugOptions
	// DisableCaching forces DrawCached to render into a fresh surface each draw.
	DisableCaching bool

	surfaceLog surfaceLogState
	frameLog   frameLogState
	frameIndex int64

	viewport    Size
	hasViewport bool

	drawOffsetX int
	drawOffsetY int
}

// NewContext creates a layout Context from a scale provider and dependencies.
func NewContext(scaleProvider PixelScaleProvider, renderer Renderer, text TextEngine) *Context {
	scale := 1.0
	if scaleProvider != nil {
		scale = scaleProvider.DeviceScaleFactor()
		if scale <= 0 {
			scale = 1.0
		}
	}
	debug := DebugOptions{}
	if envDebugEnabled("LAYOUT_DEBUG_LAYOUT") {
		debug.LogLayoutDecisions = true
	}
	if envDebugEnabled("LAYOUT_DEBUG_CSS") {
		debug.LogCSSQueries = true
	}
	if envDebugEnabled("LAYOUT_DEBUG_SURFACES") {
		debug.LogSurfaceAllocations = true
		debug.LogSurfaceUsage = true
	}
	if envDebugEnabled("LAYOUT_DEBUG_SURFACE_ALLOC") {
		debug.LogSurfaceAllocations = true
	}
	if envDebugEnabled("LAYOUT_DEBUG_SURFACE_USAGE") {
		debug.LogSurfaceUsage = true
	}
	return &Context{Scale: scale, Renderer: renderer, Text: text, Debug: debug}
}

// BeginFrameLog starts buffering debug output for a frame with the given index.
// Logs are emitted at EndFrameLog only if the buffered lines differ from the previous frame.
func (c *Context) BeginFrameLog(frame int64) {
	if c == nil {
		return
	}
	if c.frameLog.active {
		c.EndFrameLog()
	}
	if !(c.DebugLayoutEnabled() || c.DebugCSSEnabled() || c.DebugSurfacesEnabled()) {
		c.frameLog.active = false
		return
	}
	c.frameLog.begin(frame)
}

// BeginFrameLogAuto increments and starts a new frame log if logging is enabled.
func (c *Context) BeginFrameLogAuto() {
	if c == nil {
		return
	}
	c.frameIndex++
	c.BeginFrameLog(c.frameIndex)
}

// EndFrameLog flushes buffered debug output for the current frame.
func (c *Context) EndFrameLog() {
	if c == nil {
		return
	}
	c.logSurfaceMemoryIfChanged()
	c.frameLog.end(c)
}

// ViewportSize returns the current viewport dimensions (dp). Zero values mean unset.
func (c *Context) ViewportSize() Size {
	if c == nil || !c.hasViewport {
		return Size{}
	}
	return c.viewport
}

// HasViewport reports whether a viewport size is currently set.
func (c *Context) HasViewport() bool {
	return c != nil && c.hasViewport
}

// ViewportWidth returns the active viewport width in dp.
func (c *Context) ViewportWidth() float64 {
	if c == nil || !c.hasViewport {
		return 0
	}
	return c.viewport.W
}

// ViewportHeight returns the active viewport height in dp.
func (c *Context) ViewportHeight() float64 {
	if c == nil || !c.hasViewport {
		return 0
	}
	return c.viewport.H
}

// SetViewportSize updates the viewport dimensions. Non-positive values clear it.
func (c *Context) SetViewportSize(sz Size) {
	if c == nil {
		return
	}
	if sz.W <= 0 && sz.H <= 0 {
		c.viewport = Size{}
		c.hasViewport = false
		return
	}
	if sz.W < 0 {
		sz.W = 0
	}
	if sz.H < 0 {
		sz.H = 0
	}
	c.viewport = sz
	c.hasViewport = true
}

// pushOffsetPx temporarily adjusts the draw offset (pixels) used during composition.
// Call the returned function to restore the previous offset.
func (c *Context) pushOffsetPx(x, y int) func() {
	if c == nil {
		return func() {}
	}
	prevX, prevY := c.drawOffsetX, c.drawOffsetY
	c.drawOffsetX += x
	c.drawOffsetY += y
	return func() {
		c.drawOffsetX = prevX
		c.drawOffsetY = prevY
	}
}

// PushOffsetPx temporarily adjusts the draw offset (pixels) used during composition.
// Call the returned function to restore the previous offset.
func (c *Context) PushOffsetPx(x, y int) func() {
	return c.pushOffsetPx(x, y)
}

// DrawOffsetPx returns the current draw offset in pixels.
func (c *Context) DrawOffsetPx() (int, int) {
	if c == nil {
		return 0, 0
	}
	return c.drawOffset()
}

// DrawOffsetDp returns the current draw offset in dp units.
func (c *Context) DrawOffsetDp() (float64, float64) {
	x, y := c.DrawOffsetPx()
	scale := c.Scale
	if scale <= 0 {
		scale = 1
	}
	return float64(x) / scale, float64(y) / scale
}

// pushZeroOffset resets the draw offset to zero for the duration of a render call.
func (c *Context) pushZeroOffset() func() {
	if c == nil {
		return func() {}
	}
	prevX, prevY := c.drawOffsetX, c.drawOffsetY
	c.drawOffsetX = 0
	c.drawOffsetY = 0
	return func() {
		c.drawOffsetX = prevX
		c.drawOffsetY = prevY
	}
}

// drawOffset returns the current draw offset in pixels.
func (c *Context) drawOffset() (int, int) {
	if c == nil {
		return 0, 0
	}
	return c.drawOffsetX, c.drawOffsetY
}
