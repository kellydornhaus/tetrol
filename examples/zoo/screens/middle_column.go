package screens

import (
	"math"

	"github.com/kellydornhaus/layouter/layout"
)

const (
	middleSafeHeight    = 50.0
	middleToolbarHeight = 50.0
	middleInputHeight   = 50.0
	middleSliceSpacing  = 0.0
	middleKeyboardRatio = 0.4375 // width : height = 4 : 1.75

	middleFixedHeight = middleSafeHeight + middleToolbarHeight + middleInputHeight + middleSliceSpacing*4
	middleWidthFactor = 1.0 + middleKeyboardRatio // game area + keyboard

	middleMinimumWidth = 0.0
)

func buildViewportColumn(ctx *layout.Context) Screen {
	column := newMiddleColumn()

	bg := layout.NewPanelComponent(column)
	bg.SetFillWidth(true)
	bg.SetFillHeight(true)
	bg.SetFlexWeight(1)
	bg.SetBackgroundColor(toRGBA(12, 14, 28, 255))

	return &staticScreen{
		name: "Viewport Column",
		root: bg,
	}
}

type middleColumn struct {
	layout.Base

	frame   *layout.PanelComponent
	stack   *layout.VStack
	safe    *layout.PanelComponent
	toolbar *layout.PanelComponent
	game    *layout.PanelComponent
	input   *layout.PanelComponent
	keys    *layout.PanelComponent

	lastWidth  float64
	lastHeight float64
}

func newMiddleColumn() *middleColumn {
	makeSlice := func(label string, bg layout.Color, border layout.Color, minHeight float64) *layout.PanelComponent {
		panel := layout.NewPanelComponent(nil)
		panel.SetFillWidth(true)
		panel.SetPadding(layout.Insets(14))
		panel.SetBackgroundColor(bg)
		panel.SetBorder(border, 2)
		panel.SetText(label)
		panel.SetTextStyle(layout.TextStyle{
			SizeDp: 20,
			AlignH: layout.AlignCenter,
			AlignV: layout.AlignCenter,
			Color:  layout.Color{R: 240, G: 242, B: 246, A: 255},
		})
		if minHeight > 0 {
			panel.SetMinHeight(minHeight)
			panel.SetMaxHeight(minHeight)
		}
		return panel
	}

	safe := makeSlice("Safe Area", toRGBA(38, 71, 116, 255), toRGBA(58, 101, 152, 255), middleSafeHeight)
	toolbar := makeSlice("Toolbar", toRGBA(63, 50, 105, 255), toRGBA(96, 76, 158, 255), middleToolbarHeight)
	game := makeSlice("Game Area", toRGBA(92, 59, 122, 255), toRGBA(140, 90, 178, 255), 0)
	input := makeSlice("Input Area", toRGBA(54, 97, 82, 255), toRGBA(86, 140, 118, 255), middleInputHeight)
	keyboard := makeSlice("Onscreen Keyboard", toRGBA(109, 84, 53, 255), toRGBA(156, 121, 78, 255), 0)

	stack := layout.NewVStack(safe, toolbar, game, input, keyboard)
	stack.Spacing = middleSliceSpacing
	stack.SetFillWidth(true)
	stack.SetFillHeight(true)

	frame := layout.NewPanelComponent(stack)
	frame.SetBackgroundColor(toRGBA(20, 24, 40, 255))
	frame.SetBorder(toRGBA(70, 74, 112, 255), 3)
	frame.SetFillWidth(false)
	frame.SetFillHeight(true)

	col := &middleColumn{
		frame:   frame,
		stack:   stack,
		safe:    safe,
		toolbar: toolbar,
		game:    game,
		input:   input,
		keys:    keyboard,
	}
	col.SetDirty()
	return col
}

func (m *middleColumn) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size {
	availableW := dimensionOrFallback(cs.Max.W, cs.Min.W)
	availableH := dimensionOrFallback(cs.Max.H, cs.Min.H)

	if ctx != nil {
		if availableW <= 0 && ctx.ViewportWidth() > 0 {
			availableW = ctx.ViewportWidth()
		}
		if availableH <= 0 && ctx.ViewportHeight() > 0 {
			availableH = ctx.ViewportHeight()
		}
	}
	if availableH <= 0 {
		availableH = totalHeightForWidth(middleMinimumWidth)
	}

	width := widthFromHeight(availableH)
	totalHeight := math.Max(availableH, totalHeightForWidth(width))

	m.applySizing(width)
	m.frame.Measure(ctx, layout.Constraints{
		Min: layout.Size{W: width, H: totalHeight},
		Max: layout.Size{W: width, H: totalHeight},
	})

	m.lastWidth = width
	m.lastHeight = totalHeight

	size := layout.Size{W: availableW, H: totalHeight}
	return clampConstraints(size, cs)
}

func (m *middleColumn) Layout(ctx *layout.Context, parent layout.Component, bounds layout.Rect) {
	m.SetFrame(parent, bounds)

	height := bounds.H
	if height <= 0 {
		height = m.lastHeight
	}
	if height <= 0 {
		height = totalHeightForWidth(middleMinimumWidth)
	}

	width := widthFromHeight(height)
	totalHeight := math.Max(height, totalHeightForWidth(width))

	m.applySizing(width)

	columnX := bounds.X + (bounds.W-width)/2
	if columnX < bounds.X {
		columnX = bounds.X
	}

	innerBounds := layout.Rect{
		X: columnX,
		Y: bounds.Y,
		W: width,
		H: totalHeight,
	}
	m.frame.Layout(ctx, m, innerBounds)
	if m.frame.Dirty() {
		m.SetDirty()
	}
}

func (m *middleColumn) DrawTo(ctx *layout.Context, dst layout.Surface) {
	m.Base.DrawCachedWithOwner(ctx, dst, m, func(target layout.Surface) {
		m.Render(ctx, target)
	})
}

func (m *middleColumn) Render(ctx *layout.Context, dst layout.Surface) {
	m.frame.DrawTo(ctx, dst)
}

func (m *middleColumn) applySizing(width float64) {
	totalHeight := totalHeightForWidth(width)

	m.game.SetMinHeight(width)
	m.game.SetMaxHeight(width)

	keyboardHeight := width * middleKeyboardRatio
	m.keys.SetMinHeight(keyboardHeight)
	m.keys.SetMaxHeight(keyboardHeight)

	m.frame.SetMinWidth(width)
	m.frame.SetMaxWidth(width)
	m.frame.SetMinHeight(totalHeight)
	m.frame.SetMaxHeight(totalHeight)

	m.stack.SetMinWidth(width)
	m.stack.SetMaxWidth(width)
	m.stack.SetMinHeight(totalHeight)
	m.stack.SetMaxHeight(totalHeight)
}

func widthFromHeight(height float64) float64 {
	return math.Max((height-middleFixedHeight)/middleWidthFactor, 0)
}

func totalHeightForWidth(width float64) float64 {
	return middleFixedHeight + middleWidthFactor*width
}

func dimensionOrFallback(max float64, min float64) float64 {
	switch {
	case max > 0:
		return max
	case min > 0:
		return min
	default:
		return 0
	}
}

func clampConstraints(sz layout.Size, cs layout.Constraints) layout.Size {
	w := sz.W
	h := sz.H
	if cs.Max.W > 0 && w > cs.Max.W {
		w = cs.Max.W
	}
	if cs.Max.H > 0 && h > cs.Max.H {
		h = cs.Max.H
	}
	if cs.Min.W > 0 && w < cs.Min.W {
		w = cs.Min.W
	}
	if cs.Min.H > 0 && h < cs.Min.H {
		h = cs.Min.H
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return layout.Size{W: w, H: h}
}
