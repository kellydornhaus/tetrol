package layout

import "sort"

// PositionMode controls how a component participates in layout positioning.
type PositionMode int

const (
	// PositionStatic participates in normal flow (default).
	PositionStatic PositionMode = iota
	// PositionRelative participates in flow but applies offsets after layout.
	PositionRelative
	// PositionAbsolute is removed from flow and positioned within the containing block.
	PositionAbsolute
	// PositionFixed is removed from flow and positioned relative to the viewport.
	PositionFixed
)

// String returns the lower-case identifier for a position mode.
func (m PositionMode) String() string {
	switch m {
	case PositionRelative:
		return "relative"
	case PositionAbsolute:
		return "absolute"
	case PositionFixed:
		return "fixed"
	default:
		return "static"
	}
}

// PositionValue represents an optional positional offset in dp units.
type PositionValue struct {
	Value   float64
	Defined bool
}

// PositionOffsets groups optional offsets for each side.
type PositionOffsets struct {
	Top    PositionValue
	Right  PositionValue
	Bottom PositionValue
	Left   PositionValue
}

// positionAccessor exposes stored positioning information.
type positionAccessor interface {
	PositionMode() PositionMode
	PositionOffsets() PositionOffsets
	ZIndex() (int, bool)
}

// positionSetter allows assigning positioning information.
type positionSetter interface {
	SetPositionMode(PositionMode)
	SetPositionOffsets(PositionOffsets)
	SetZIndex(int, bool)
}

// PositionModeOf reports the current position mode for the component (defaults to static).
func PositionModeOf(c Component) PositionMode {
	if c == nil {
		return PositionStatic
	}
	if accessor, ok := c.(positionAccessor); ok {
		return accessor.PositionMode()
	}
	return PositionStatic
}

// SetPositionMode assigns the position mode when supported.
func SetPositionMode(c Component, mode PositionMode) {
	if setter, ok := c.(positionSetter); ok {
		setter.SetPositionMode(mode)
	}
}

// PositionOffsetsOf reports the stored offsets for the component (zero values when unset).
func PositionOffsetsOf(c Component) PositionOffsets {
	if c == nil {
		return PositionOffsets{}
	}
	if accessor, ok := c.(positionAccessor); ok {
		return accessor.PositionOffsets()
	}
	return PositionOffsets{}
}

// SetPositionOffsets assigns the offsets when supported.
func SetPositionOffsets(c Component, offsets PositionOffsets) {
	if setter, ok := c.(positionSetter); ok {
		setter.SetPositionOffsets(offsets)
	}
}

// ZIndexOf reports the z-index value when present.
func ZIndexOf(c Component) (int, bool) {
	if c == nil {
		return 0, false
	}
	if accessor, ok := c.(positionAccessor); ok {
		return accessor.ZIndex()
	}
	return 0, false
}

// SetZIndex assigns the z-index when supported (define=false clears).
func SetZIndex(c Component, value int, defined bool) {
	if setter, ok := c.(positionSetter); ok {
		setter.SetZIndex(value, defined)
	}
}

// isOutOfFlow returns true when the component should be removed from normal layout flow.
func isOutOfFlow(c Component) bool {
	switch PositionModeOf(c) {
	case PositionAbsolute, PositionFixed:
		return true
	default:
		return false
	}
}

type orderedComponent struct {
	component Component
	zIndex    int
	hasZ      bool
	index     int
}

// orderByZIndex sorts components by z-index (ascending), falling back to slice order.
func orderByZIndex(children []Component) []Component {
	if len(children) == 0 {
		return nil
	}
	items := make([]orderedComponent, 0, len(children))
	anyZ := false
	for idx, child := range children {
		if child == nil {
			continue
		}
		z, hasZ := ZIndexOf(child)
		if hasZ {
			anyZ = true
		}
		items = append(items, orderedComponent{component: child, zIndex: z, hasZ: hasZ, index: idx})
	}
	if !anyZ {
		out := make([]Component, 0, len(items))
		for _, item := range items {
			out = append(out, item.component)
		}
		return out
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].zIndex == items[j].zIndex {
			if items[i].hasZ == items[j].hasZ {
				return items[i].index < items[j].index
			}
			if items[i].hasZ {
				return false
			}
			return true
		}
		return items[i].zIndex < items[j].zIndex
	})
	out := make([]Component, 0, len(items))
	for _, item := range items {
		out = append(out, item.component)
	}
	return out
}

func flowOffsets(offsets PositionOffsets) (dx, dy float64) {
	if offsets.Left.Defined {
		dx += offsets.Left.Value
	} else if offsets.Right.Defined {
		dx -= offsets.Right.Value
	}
	if offsets.Top.Defined {
		dy += offsets.Top.Value
	} else if offsets.Bottom.Defined {
		dy -= offsets.Bottom.Value
	}
	return dx, dy
}

func applyRelativeOffsets(flowRect Rect, offsets PositionOffsets) Rect {
	dx, dy := flowOffsets(offsets)
	flowRect.X += dx
	flowRect.Y += dy
	return flowRect
}

func viewportRect(ctx *Context) Rect {
	if ctx == nil {
		return Rect{}
	}
	return Rect{X: 0, Y: 0, W: ctx.ViewportWidth(), H: ctx.ViewportHeight()}
}

func resolveOutOfFlowRect(ctx *Context, parent Component, child Component, container Rect, mode PositionMode) Rect {
	offsets := PositionOffsetsOf(child)
	target := container
	if mode == PositionFixed {
		vp := viewportRect(ctx)
		if vp.W > 0 || vp.H > 0 {
			target = vp
		}
	}

	widthConstraint := -1.0
	if offsets.Left.Defined && offsets.Right.Defined {
		widthConstraint = target.W
		if target.W > 0 {
			widthConstraint = target.W - offsets.Left.Value - offsets.Right.Value
		}
	}
	heightConstraint := -1.0
	if offsets.Top.Defined && offsets.Bottom.Defined {
		heightConstraint = target.H
		if target.H > 0 {
			heightConstraint = target.H - offsets.Top.Value - offsets.Bottom.Value
		}
	}

	constraints := Constraints{}
	if widthConstraint >= 0 {
		constraints.Max.W = widthConstraint
		constraints.Min.W = maxFloat(0, widthConstraint)
	}
	if heightConstraint >= 0 {
		constraints.Max.H = heightConstraint
		constraints.Min.H = maxFloat(0, heightConstraint)
	}

	size := measureComponent(ctx, parent, child, constraints)
	width := size.W
	if widthConstraint >= 0 {
		width = maxFloat(0, widthConstraint)
	}
	height := size.H
	if heightConstraint >= 0 {
		height = maxFloat(0, heightConstraint)
	}

	x := target.X
	if offsets.Left.Defined {
		x = target.X + offsets.Left.Value
	} else if offsets.Right.Defined {
		x = target.X + target.W - width - offsets.Right.Value
	}
	y := target.Y
	if offsets.Top.Defined {
		y = target.Y + offsets.Top.Value
	} else if offsets.Bottom.Defined {
		y = target.Y + target.H - height - offsets.Bottom.Value
	}

	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return Rect{X: x, Y: y, W: width, H: height}
}

func layoutFlowChild(ctx *Context, parent Component, child Component, flowRect Rect) {
	if child == nil {
		return
	}
	mode := PositionModeOf(child)
	note := mode.String()
	switch PositionModeOf(child) {
	case PositionRelative:
		rect := applyRelativeOffsets(flowRect, PositionOffsetsOf(child))
		logChildLayout(ctx, parent, child, rect, note)
		child.Layout(ctx, parent, rect)
	default:
		logChildLayout(ctx, parent, child, flowRect, note)
		child.Layout(ctx, parent, flowRect)
	}
}

func layoutOutOfFlowChild(ctx *Context, parent Component, child Component, container Rect) {
	if child == nil {
		return
	}
	mode := PositionModeOf(child)
	if mode != PositionAbsolute && mode != PositionFixed {
		return
	}
	rect := resolveOutOfFlowRect(ctx, parent, child, container, mode)
	logChildLayout(ctx, parent, child, rect, mode.String())
	child.Layout(ctx, parent, rect)
}

func layoutOutOfFlowChildren(ctx *Context, parent Component, children []Component, container Rect) {
	for _, child := range children {
		if child == nil {
			continue
		}
		mode := PositionModeOf(child)
		if mode != PositionAbsolute && mode != PositionFixed {
			continue
		}
		layoutOutOfFlowChild(ctx, parent, child, container)
	}
}
