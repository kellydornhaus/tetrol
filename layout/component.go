package layout

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Component is the minimal interface for a layoutable and renderable object.
// Units are logical (dp) for measurement and layout; rendering is done in pixels via Context.Renderer.
type Component interface {
	// Measure returns the desired size within constraints (dp units).
	Measure(ctx *Context, cs Constraints) Size
	// Layout assigns bounds (dp) to this component and its children.
	// parent is the node's immediate parent (nil for the root).
	Layout(ctx *Context, parent Component, bounds Rect)
	// Dirty reports whether the visual output may have changed and needs re-render.
	Dirty() bool
	// DrawTo draws this component (and its subtree) onto dst using internal caching.
	DrawTo(ctx *Context, dst Surface)
	// Render draws the component (and its children) into the provided surface.
	// The surface's pixel size equals the assigned bounds times the device scale.
	Render(ctx *Context, dst Surface)
	// Bounds returns this component's local bounds (relative to its parent).
	Bounds() Rect
	// GlobalBounds returns this component's bounds in root coordinates.
	GlobalBounds() Rect
}

// Base provides common state for components: bounds, dirtiness, and cached surface.
// Embed Base in your components to get caching behavior via DrawCached.
type Base struct {
	bounds               Rect
	dirty                bool
	cache                Surface
	cacheScale           float64
	disableCache         bool
	parent               Component
	flexWeight           float64
	visibility           Visibility
	visibilityTransition VisibilityTransition
	visTransitionEnabled bool
	visAnim              visibilityTransitionState
	position             PositionMode
	posOffsets           PositionOffsets
	zIndex               int
	zIndexSet            bool
	alignSelf            TextAlign
	alignSelfSet         bool
	dialogVars           map[string]string
}

func (b *Base) Bounds() Rect { return b.bounds }

// GlobalBounds walks up the parent chain to accumulate the absolute position.
func (b *Base) GlobalBounds() Rect {
	if b.parent == nil {
		return b.bounds
	}
	parentRect := b.parent.GlobalBounds()
	return Rect{
		X: parentRect.X + b.bounds.X,
		Y: parentRect.Y + b.bounds.Y,
		W: b.bounds.W,
		H: b.bounds.H,
	}
}

// SetDirty marks the component as needing re-render.
func (b *Base) SetDirty() { b.dirty = true }

// Dirty returns whether the component is dirty.
func (b *Base) Dirty() bool { return b.dirty }

// SetFrame assigns the parent and bounds, marking dirty if the size changed.
func (b *Base) SetFrame(parent Component, bounds Rect) {
	if b.parent != parent {
		b.parent = parent
	}
	if b.bounds != bounds {
		b.bounds = bounds
		b.dirty = true
	}
}

// releaseCache frees the cached surface by letting it be GC'd.
func (b *Base) releaseCache() { b.cache = nil }

// SetCacheEnabled toggles caching for this component only (children unaffected).
func (b *Base) SetCacheEnabled(enabled bool) {
	if b == nil {
		return
	}
	disable := !enabled
	if b.disableCache == disable {
		return
	}
	b.disableCache = disable
	if disable {
		b.releaseCache()
	}
	b.SetDirty()
}

// CacheEnabled reports whether this component caches its own surface.
func (b *Base) CacheEnabled() bool {
	return b != nil && !b.disableCache
}

// DrawCached manages per-component caching in an adapter-agnostic way.
// It allocates (or reuses) an offscreen surface sized to the component's bounds in pixels,
// calls render() when dirty, then composites the cached surface into dst at the bounds position.
func (b *Base) DrawCached(ctx *Context, dst Surface, render func(target Surface)) {
	b.drawCached(ctx, dst, nil, render)
}

// DrawCachedWithOwner is like DrawCached but tags debug logs with the owning component.
func (b *Base) DrawCachedWithOwner(ctx *Context, dst Surface, owner Component, render func(target Surface)) {
	b.drawCached(ctx, dst, owner, render)
}

func (b *Base) drawCached(ctx *Context, dst Surface, owner Component, render func(target Surface)) {
	if !b.ShouldRender() {
		if ctx != nil && ctx.DebugSurfaceAllocEnabled() {
			ctx.logSurfaceRelease(b, owner, "skip: should not render")
		}
		b.releaseCache()
		return
	}
	if ctx == nil || ctx.Renderer == nil || dst == nil {
		return
	}
	px := b.bounds.ToPx(ctx.Scale)
	// guard sizes
	if px.W <= 0 || px.H <= 0 {
		if ctx != nil && ctx.DebugSurfaceAllocEnabled() {
			ctx.logSurfaceRelease(b, owner, "skip: non-positive bounds")
		}
		return
	}
	size := PxSize{W: px.W, H: px.H}
	prevSize := PxSize{}
	if b.cache != nil {
		if cw, ch := b.cache.SizePx(); cw > 0 || ch > 0 {
			prevSize = PxSize{W: cw, H: ch}
		}
	}
	if ctx.DisableCaching || b.disableCache {
		if b.cache != nil {
			if ctx.DebugSurfaceAllocEnabled() {
				ctx.logSurfaceRelease(b, owner, "cache disabled")
			}
			b.releaseCache()
		}
		cache := ctx.Renderer.NewSurface(px.W, px.H)
		ctx.logSurfaceAllocation(b, owner, size, prevSize)
		cache.Clear()
		restore := ctx.pushZeroOffset()
		if render != nil {
			render(cache)
		}
		restore()
		b.dirty = false

		scale := 1.0
		if b.visTransitionEnabled {
			now := time.Now()
			scale = b.visibilityScale(now)
		}

		note := "cache-disabled"
		if scale <= 0 {
			if b.visTransitionEnabled {
				note = appendNote(note, "skip draw (scale<=0)")
			}
			ctx.logSurfaceDraw(b, owner, size, true, note)
			return
		}
		if !nearlyEqual(scale, 1) {
			note = appendNote(note, fmt.Sprintf("scale=%.2f", scale))
		}
		ctx.logSurfaceDraw(b, owner, size, true, note)

		offsetX, offsetY := ctx.drawOffset()
		px.X += offsetX
		px.Y += offsetY

		if !nearlyEqual(scale, 1) {
			if scaler, ok := ctx.Renderer.(SurfaceScaler); ok {
				scaler.DrawSurfaceScaled(dst, cache, px, scale, scale)
				return
			}
			if rectScaler, ok := ctx.Renderer.(SurfaceRectRenderer); ok {
				target := scalePxRect(px, scale)
				if target.W <= 0 || target.H <= 0 {
					return
				}
				rectScaler.DrawSurfaceRect(dst, cache, target)
				return
			}
		}
		ctx.Renderer.DrawSurface(dst, cache, px.X, px.Y)
		return
	}
	recreate := false
	if b.cache == nil || b.cacheScale != ctx.Scale {
		recreate = true
	} else {
		if cw, ch := b.cache.SizePx(); cw != px.W || ch != px.H {
			recreate = true
		}
	}
	if recreate {
		b.cache = ctx.Renderer.NewSurface(px.W, px.H)
		b.cacheScale = ctx.Scale
		b.dirty = true
		ctx.logSurfaceAllocation(b, owner, size, prevSize)
	}

	rendered := false
	if b.dirty {
		b.cache.Clear()
		restore := ctx.pushZeroOffset()
		if render != nil {
			render(b.cache)
		}
		restore()
		b.dirty = false
		rendered = true
	}

	scale := 1.0
	if b.visTransitionEnabled {
		now := time.Now()
		scale = b.visibilityScale(now)
	}

	note := ""
	if recreate {
		note = appendNote(note, "recreated")
	}
	if rendered {
		note = appendNote(note, "rendered")
	} else {
		note = appendNote(note, "cache-hit")
	}
	if scale <= 0 {
		if b.visTransitionEnabled {
			note = appendNote(note, "skip draw (scale<=0)")
		}
		ctx.logSurfaceDraw(b, owner, size, rendered, note)
		b.releaseCache()
		return
	}
	if !nearlyEqual(scale, 1) {
		note = appendNote(note, fmt.Sprintf("scale=%.2f", scale))
	}
	ctx.logSurfaceDraw(b, owner, size, rendered, note)

	offsetX, offsetY := ctx.drawOffset()
	px.X += offsetX
	px.Y += offsetY

	// Composite into destination at pixel position.
	if !nearlyEqual(scale, 1) {
		if scaler, ok := ctx.Renderer.(SurfaceScaler); ok {
			scaler.DrawSurfaceScaled(dst, b.cache, px, scale, scale)
			return
		}
		if rectScaler, ok := ctx.Renderer.(SurfaceRectRenderer); ok {
			target := scalePxRect(px, scale)
			if target.W <= 0 || target.H <= 0 {
				return
			}
			rectScaler.DrawSurfaceRect(dst, b.cache, target)
			return
		}
	}
	ctx.Renderer.DrawSurface(dst, b.cache, px.X, px.Y)
}

func appendNote(note, part string) string {
	if part == "" {
		return note
	}
	if note == "" {
		return part
	}
	return note + "; " + part
}

// SetFlexWeight stores the preferred flex weight for layout stacks.
func (b *Base) SetFlexWeight(weight float64) {
	if weight < 0 {
		weight = 0
	}
	if nearlyEqual(b.flexWeight, weight) {
		return
	}
	b.flexWeight = weight
	b.dirty = true
}

// FlexWeight reports the preferred flex weight.
func (b *Base) FlexWeight() float64 { return b.flexWeight }

// SetVisibility stores the current visibility state.
func (b *Base) SetVisibility(visibility Visibility) {
	if visibility < VisibilityVisible || visibility > VisibilityCollapse {
		visibility = VisibilityVisible
	}
	if b.visibility == visibility {
		return
	}
	prev := b.visibility
	b.visibility = visibility
	b.handleVisibilityTransition(prev, visibility)
	b.SetDirty()
	if b.visibility == VisibilityCollapse {
		b.bounds = Rect{}
	}
	if !b.ShouldRender() {
		b.releaseCache()
	}
}

// Visibility reports the component's visibility mode.
func (b *Base) Visibility() Visibility { return b.visibility }

// SetVisibilityTransition configures the transition used for visibility changes.
func (b *Base) SetVisibilityTransition(transition VisibilityTransition) {
	enabled := transition.Enabled()
	if enabled {
		transition.Scale = clampFraction(transition.Scale)
	} else {
		transition = VisibilityTransition{}
	}
	if b.visibilityTransition == transition && b.visTransitionEnabled == enabled {
		return
	}
	b.visibilityTransition = transition
	b.visTransitionEnabled = enabled
	if !enabled {
		b.visAnim.reset()
	}
	b.SetDirty()
}

// VisibilityTransition reports the stored transition configuration.
func (b *Base) VisibilityTransition() (VisibilityTransition, bool) {
	if !b.visTransitionEnabled {
		return VisibilityTransition{}, false
	}
	return b.visibilityTransition, true
}

// ShouldRender reports whether the component should draw this frame.
func (b *Base) ShouldRender() bool {
	if b.visibility == VisibilityCollapse {
		return false
	}
	if b.visibility.renders() {
		return true
	}
	return b.visTransitionEnabled && b.visAnim.active
}

// PositionMode reports the stored positioning mode.
func (b *Base) PositionMode() PositionMode { return b.position }

// SetPositionMode updates the positioning mode.
func (b *Base) SetPositionMode(mode PositionMode) {
	if mode < PositionStatic || mode > PositionFixed {
		mode = PositionStatic
	}
	if b.position == mode {
		return
	}
	b.position = mode
	b.SetDirty()
}

// PositionOffsets returns the stored offsets.
func (b *Base) PositionOffsets() PositionOffsets { return b.posOffsets }

// SetPositionOffsets assigns new offsets (undefined sides cleared when Defined=false).
func (b *Base) SetPositionOffsets(offsets PositionOffsets) {
	if b.posOffsets == offsets {
		return
	}
	b.posOffsets = offsets
	b.SetDirty()
}

// ZIndex reports the stored z-index (defined=false when unset).
func (b *Base) ZIndex() (int, bool) {
	return b.zIndex, b.zIndexSet
}

// SetZIndex updates the stored z-index (defined=false clears).
func (b *Base) SetZIndex(value int, defined bool) {
	if !defined {
		if !b.zIndexSet {
			return
		}
		b.zIndex = 0
		b.zIndexSet = false
		b.SetDirty()
		return
	}
	if b.zIndexSet && b.zIndex == value {
		return
	}
	b.zIndex = value
	b.zIndexSet = true
	b.SetDirty()
}

// AlignSelf reports an optional cross-axis alignment override for container layouts.
func (b *Base) AlignSelf() (TextAlign, bool) {
	return b.alignSelf, b.alignSelfSet
}

// SetAlignSelf updates the cross-axis alignment override. When defined=false the override is cleared.
func (b *Base) SetAlignSelf(align TextAlign, defined bool) {
	if !defined {
		if !b.alignSelfSet {
			return
		}
		b.alignSelfSet = false
		b.SetDirty()
		return
	}
	if align < AlignStart || align > AlignEnd {
		align = AlignStart
	}
	if b.alignSelfSet && b.alignSelf == align {
		return
	}
	b.alignSelf = align
	b.alignSelfSet = true
	b.SetDirty()
}

func (b *Base) visibilityScale(now time.Time) float64 {
	if !b.visTransitionEnabled {
		return 1
	}
	if !b.visAnim.active {
		if b.visibility == VisibilityVisible {
			return 1
		}
		return clampFraction(b.visibilityTransition.Scale)
	}
	scale, ongoing := b.visAnim.value(now)
	if ongoing {
		b.dirty = true
	}
	if !ongoing {
		if b.visibility == VisibilityVisible {
			return 1
		}
		return clampFraction(b.visibilityTransition.Scale)
	}
	return scale
}

func (b *Base) handleVisibilityTransition(prev, next Visibility) {
	if !b.visTransitionEnabled {
		b.visAnim.reset()
		return
	}
	if prev == VisibilityCollapse || next == VisibilityCollapse {
		b.visAnim.reset()
		return
	}

	var direction int
	switch {
	case next == VisibilityVisible && prev != VisibilityVisible:
		if !b.visibilityTransition.AnimateIn() {
			b.visAnim.reset()
			return
		}
		direction = 1
	case prev == VisibilityVisible && next != VisibilityVisible:
		if !b.visibilityTransition.AnimateOut() {
			b.visAnim.reset()
			return
		}
		direction = -1
	default:
		b.visAnim.reset()
		return
	}

	now := time.Now()
	current := 1.0
	if b.visAnim.active {
		if scale, ongoing := b.visAnim.value(now); ongoing {
			current = scale
		} else {
			if prev == VisibilityVisible {
				current = 1
			} else {
				current = clampFraction(b.visibilityTransition.Scale)
			}
		}
	} else {
		if prev == VisibilityVisible {
			current = 1
		} else {
			current = clampFraction(b.visibilityTransition.Scale)
		}
	}
	b.visAnim.begin(b.visibilityTransition, direction, now, current)
}

func scalePxRect(rect PxRect, scale float64) PxRect {
	if scale <= 0 {
		return PxRect{}
	}
	if nearlyEqual(scale, 1) {
		return rect
	}
	width := float64(rect.W)
	height := float64(rect.H)
	scaledW := int(math.Round(width * scale))
	scaledH := int(math.Round(height * scale))
	if rect.W > 0 && scaledW == 0 && scale > 0 {
		scaledW = 1
	}
	if rect.H > 0 && scaledH == 0 && scale > 0 {
		scaledH = 1
	}
	offsetX := rect.X
	offsetY := rect.Y
	if scaledW != rect.W {
		offsetX = rect.X + int(math.Round((width-float64(scaledW))/2))
	}
	if scaledH != rect.H {
		offsetY = rect.Y + int(math.Round((height-float64(scaledH))/2))
	}
	return PxRect{X: offsetX, Y: offsetY, W: scaledW, H: scaledH}
}

func clampScale(value, min, max float64) float64 {
	value = maxFloat(value, min)
	return minFloat(value, max)
}

type visibilityTransitionState struct {
	cfg       VisibilityTransition
	from      float64
	to        float64
	start     time.Time
	active    bool
	direction int
}

func (s *visibilityTransitionState) reset() {
	s.cfg = VisibilityTransition{}
	s.from = 0
	s.to = 0
	s.start = time.Time{}
	s.active = false
	s.direction = 0
}

func (s *visibilityTransitionState) begin(cfg VisibilityTransition, direction int, now time.Time, start float64) {
	if !cfg.Enabled() || direction == 0 {
		s.reset()
		return
	}
	minScale := clampFraction(cfg.Scale)
	s.cfg = cfg
	s.direction = direction
	s.start = now
	s.active = true
	if direction > 0 {
		if start <= 0 {
			start = minScale
		}
		s.from = clampScale(start, minScale, 1)
		s.to = 1
	} else {
		if start <= 0 {
			start = 1
		}
		s.from = clampScale(start, minScale, 1)
		s.to = minScale
	}
	if nearlyEqual(s.from, s.to) {
		s.active = false
	}
}

func (s *visibilityTransitionState) value(now time.Time) (float64, bool) {
	if !s.active || !s.cfg.Enabled() {
		s.active = false
		if s.direction > 0 {
			return 1, false
		}
		return clampFraction(s.cfg.Scale), false
	}
	duration := s.cfg.Duration
	if duration <= 0 {
		s.active = false
		return s.to, false
	}
	elapsed := now.Sub(s.start)
	if elapsed <= 0 {
		return s.from, true
	}
	progress := elapsed.Seconds() / duration.Seconds()
	if progress >= 1 {
		s.active = false
		return s.to, false
	}
	current := s.from + (s.to-s.from)*progress
	return current, true
}

type dialogHost interface {
	dialogBase() *Base
}

func dialogBaseOf(c Component) *Base {
	if host, ok := c.(dialogHost); ok {
		return host.dialogBase()
	}
	return nil
}

func parentComponentOf(c Component) Component {
	if base := dialogBaseOf(c); base != nil {
		return base.parent
	}
	return nil
}

func attachDialogParent(child Component, parent Component) {
	if base := dialogBaseOf(child); base != nil {
		base.parent = parent
	}
}

func visitChildren(c Component, fn func(Component)) {
	switch comp := c.(type) {
	case *PanelComponent:
		if comp.Child != nil {
			fn(comp.Child)
		}
	case *VStack:
		for _, ch := range comp.Children {
			if ch != nil {
				fn(ch)
			}
		}
	case *HStack:
		for _, ch := range comp.Children {
			if ch != nil {
				fn(ch)
			}
		}
	case *FlowStack:
		for _, ch := range comp.Children {
			if ch != nil {
				fn(ch)
			}
		}
	case *ZStack:
		for _, ch := range comp.Children {
			if ch != nil {
				fn(ch)
			}
		}
	case *Grid:
		for _, ch := range comp.Children {
			if ch != nil {
				fn(ch)
			}
		}
	default:
		if provider, ok := c.(interface{ DialogChildren() []Component }); ok {
			for _, ch := range provider.DialogChildren() {
				if ch != nil {
					fn(ch)
				}
			}
		}
	}
}

func hostFromComponent(c Component) dialogHost {
	if host, ok := c.(dialogHost); ok {
		return host
	}
	return nil
}

func parentDialogHost(curr dialogHost) dialogHost {
	if curr == nil {
		return nil
	}
	base := curr.dialogBase()
	if base == nil {
		return nil
	}
	if parentHost, ok := base.parent.(dialogHost); ok {
		return parentHost
	}
	return nil
}

func propagateDialogVariableDirty(c Component) {
	if c == nil {
		return
	}
	if base := dialogBaseOf(c); base != nil {
		base.SetDirty()
	}
	visitChildren(c, func(child Component) {
		propagateDialogVariableDirty(child)
	})
}

func resolveDialogVariableForHost(host dialogHost, name string) (string, bool) {
	for current := host; current != nil; current = parentDialogHost(current) {
		if base := current.dialogBase(); base != nil && base.dialogVars != nil {
			if val, ok := base.dialogVars[name]; ok {
				return val, true
			}
		}
	}
	return "", false
}

func hasDialogPlaceholder(s string) bool {
	start := strings.Index(s, "{{")
	for start >= 0 {
		if end := strings.Index(s[start+2:], "}}"); end >= 0 {
			return true
		}
		if start+2 >= len(s) {
			break
		}
		next := strings.Index(s[start+2:], "{{")
		if next < 0 {
			break
		}
		start = start + 2 + next
	}
	return false
}

func resolveDialogTemplate(host dialogHost, template string) string {
	if template == "" {
		return ""
	}
	start := strings.Index(template, "{{")
	if start < 0 {
		return template
	}
	var sb strings.Builder
	i := 0
	for i < len(template) {
		open := strings.Index(template[i:], "{{")
		if open < 0 {
			sb.WriteString(template[i:])
			break
		}
		sb.WriteString(template[i : i+open])
		i += open + 2
		close := strings.Index(template[i:], "}}")
		if close < 0 {
			sb.WriteString("{{")
			sb.WriteString(template[i:])
			break
		}
		key := strings.TrimSpace(template[i : i+close])
		if key != "" {
			if val, ok := resolveDialogVariableForHost(host, key); ok {
				sb.WriteString(val)
			}
		}
		i += close + 2
	}
	return sb.String()
}

func setDialogVariableInBase(base *Base, name, value string) bool {
	if base == nil {
		return false
	}
	if base.dialogVars != nil {
		if old, ok := base.dialogVars[name]; ok && old == value {
			return false
		}
	} else {
		base.dialogVars = make(map[string]string)
	}
	base.dialogVars[name] = value
	return true
}

func clearDialogVariableInBase(base *Base, name string) bool {
	if base == nil || base.dialogVars == nil {
		return false
	}
	if _, ok := base.dialogVars[name]; !ok {
		return false
	}
	delete(base.dialogVars, name)
	if len(base.dialogVars) == 0 {
		base.dialogVars = nil
	}
	return true
}

// SetDialogVariable stores a dialog variable on the component. Descendants inherit it unless overridden.
func SetDialogVariable(c Component, name, value string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	host := hostFromComponent(c)
	if host == nil {
		return
	}
	if !setDialogVariableInBase(host.dialogBase(), name, value) {
		return
	}
	propagateDialogVariableDirty(c)
}

// ClearDialogVariable removes a dialog variable override from the component.
func ClearDialogVariable(c Component, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	host := hostFromComponent(c)
	if host == nil {
		return
	}
	if !clearDialogVariableInBase(host.dialogBase(), name) {
		return
	}
	propagateDialogVariableDirty(c)
}

// DialogVariable looks up a dialog variable value, searching up the parent chain.
func DialogVariable(c Component, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	host := hostFromComponent(c)
	if host == nil {
		return "", false
	}
	return resolveDialogVariableForHost(host, name)
}

// SetFlexWeight assigns the flex weight for the given component when supported.
func SetFlexWeight(c Component, weight float64) {
	if setter, ok := c.(interface{ SetFlexWeight(float64) }); ok {
		setter.SetFlexWeight(weight)
	}
}

// FlexWeight retrieves the component's flex weight when available.
func FlexWeight(c Component) float64 {
	if getter, ok := c.(interface{ FlexWeight() float64 }); ok {
		return getter.FlexWeight()
	}
	return 0
}

// AlignSelfOf retrieves a component's cross-axis alignment override when available.
func AlignSelfOf(c Component) (TextAlign, bool) {
	if getter, ok := c.(interface{ AlignSelf() (TextAlign, bool) }); ok {
		return getter.AlignSelf()
	}
	return AlignStart, false
}

// SetAlignSelf assigns the cross-axis alignment override when supported (defined=false clears).
func SetAlignSelf(c Component, align TextAlign, defined bool) {
	if setter, ok := c.(interface{ SetAlignSelf(TextAlign, bool) }); ok {
		setter.SetAlignSelf(align, defined)
	}
}
