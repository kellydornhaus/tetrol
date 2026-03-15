package layout

import "math"

// FlowStack arranges children left-to-right and wraps onto new rows when they
// no longer fit within the available width. Each row honours spacing and
// justification similar to HStack, while LineSpacing governs gaps between rows.
type FlowStack struct {
	*Panel
	Children     []Component
	Spacing      float64
	LineSpacing  float64
	AlignItems   TextAlign
	AlignContent TextAlign
	Justify      Justify
}

// NewFlowStack constructs a flow layout with default spacing/alignment.
func NewFlowStack(children ...Component) *FlowStack {
	f := &FlowStack{
		Panel:        NewPanel(),
		Children:     children,
		Spacing:      0,
		LineSpacing:  0,
		AlignItems:   AlignStart,
		AlignContent: AlignStart,
		Justify:      JustifyStart,
	}
	f.SetFillWidth(true)
	for _, ch := range children {
		if ch != nil {
			attachDialogParent(ch, f)
		}
	}
	return f
}

func (f *FlowStack) dialogBase() *Base {
	if f == nil || f.Panel == nil {
		return nil
	}
	return &f.Panel.Base
}

// Add appends a child to the flow and marks it dirty.
func (f *FlowStack) Add(child Component) {
	if f == nil || child == nil {
		return
	}
	f.Children = append(f.Children, child)
	attachDialogParent(child, f)
	f.SetDirty()
}

// Clear removes all children from the flow stack.
func (f *FlowStack) Clear() {
	if f == nil {
		return
	}
	if len(f.Children) == 0 {
		return
	}
	for _, ch := range f.Children {
		if ch != nil {
			attachDialogParent(ch, nil)
		}
	}
	f.Children = f.Children[:0]
	f.SetDirty()
}

func (f *FlowStack) Measure(ctx *Context, cs Constraints) Size {
	if f.Visibility() == VisibilityCollapse {
		return Size{}
	}
	inner := f.contentConstraints(ctx, cs)
	active := f.activeChildren()
	if len(active) == 0 {
		return f.resolveSize(ctx, cs, Size{})
	}

	rows := f.measureRows(ctx, inner, active)
	var maxW float64
	var totalH float64
	for _, row := range rows {
		if row.width > maxW {
			maxW = row.width
		}
		totalH += row.height
	}
	if len(rows) > 1 {
		totalH += f.LineSpacing * float64(len(rows)-1)
	}
	result := f.resolveSize(ctx, cs, Size{W: maxW, H: totalH})
	logSelfMeasure(ctx, f, cs, result)
	return result
}

func (f *FlowStack) Layout(ctx *Context, parent Component, bounds Rect) {
	if f.Visibility() == VisibilityCollapse {
		f.SetFrame(parent, Rect{})
		return
	}
	f.SetFrame(parent, bounds)
	logLayoutBounds(ctx, f, parent, bounds)
	content := f.ContentBounds()
	f.refreshAutoTextSize(ctx, content)
	defer layoutOutOfFlowChildren(ctx, f, f.Children, content)
	active := f.activeChildren()
	if len(active) == 0 {
		return
	}

	rows := f.layoutRows(ctx, content, active)
	var totalHeight float64
	for _, row := range rows {
		totalHeight += row.height
	}
	if len(rows) > 1 {
		totalHeight += f.LineSpacing * float64(len(rows)-1)
	}

	startY := content.Y
	if extra := content.H - totalHeight; extra > 0 {
		switch f.AlignContent {
		case AlignCenter:
			startY += extra / 2
		case AlignEnd:
			startY += extra
		}
	}

	y := startY
	for rowIdx, row := range rows {
		rowWidth := row.width
		available := content.W
		if available <= 0 {
			available = rowWidth
		}
		x := content.X
		gap := f.Spacing
		if available > rowWidth {
			space := available - rowWidth
			switch f.Justify {
			case JustifyCenter:
				x += space / 2
			case JustifyEnd:
				x += space
			case JustifySpaceBetween:
				if len(row.items) > 1 {
					gap = f.Spacing + space/float64(len(row.items)-1)
				}
			}
		}

		for idx, item := range row.items {
			ch := item.component
			targetH := item.size.H
			if fh, ok := ch.(interface{ FillHeight() bool }); ok && fh.FillHeight() && row.height > 0 {
				targetH = row.height
			}
			if targetH > row.height && row.height > 0 {
				targetH = row.height
			}

			align := f.AlignItems
			if aligned, ok := AlignSelfOf(ch); ok {
				align = aligned
			}
			childY := y
			switch align {
			case AlignCenter:
				childY = y + (row.height-targetH)/2
			case AlignEnd:
				childY = y + (row.height - targetH)
			default:
				childY = y
			}
			if childY < y {
				childY = y
			}
			if targetH <= 0 {
				targetH = item.size.H
			}
			if row.height <= 0 && targetH < 0 {
				targetH = 0
			}

			width := maxFloat(item.size.W, 0)
			rect := Rect{X: x, Y: childY, W: width, H: targetH}
			layoutFlowChild(ctx, f, ch, rect)
			if ch.Dirty() {
				f.SetDirty()
			}

			x += width
			if idx < len(row.items)-1 {
				x += gap
			}
		}
		y += row.height
		if rowIdx < len(rows)-1 {
			y += f.LineSpacing
		}
	}

}

func (f *FlowStack) DrawTo(ctx *Context, dst Surface) {
	if !f.ShouldRender() {
		f.Base.releaseCache()
		return
	}
	for _, ch := range f.Children {
		if ch != nil && ch.Dirty() {
			f.SetDirty()
			break
		}
	}
	f.DrawPanelChildrenWithOwner(ctx, dst, f, func(target Surface) { f.Render(ctx, target) })
}

func (f *FlowStack) Render(ctx *Context, dst Surface) {
	ordered := orderByZIndex(f.Children)
	for _, ch := range ordered {
		if ch != nil && rendersToSurface(ch) {
			ch.DrawTo(ctx, dst)
		}
	}
}

type flowItem struct {
	component Component
	size      Size
	weight    float64
}

type flowRow struct {
	items  []flowItem
	width  float64
	height float64
}

func (f *FlowStack) activeChildren() []Component {
	active := make([]Component, 0, len(f.Children))
	for _, ch := range f.Children {
		if ch == nil || !participatesInLayout(ch) {
			continue
		}
		active = append(active, ch)
	}
	return active
}

func (f *FlowStack) measureRows(ctx *Context, inner Constraints, children []Component) []flowRow {
	available := inner.Max.W
	if available <= 0 {
		available = math.MaxFloat64
	}
	childConstraints := Constraints{Max: Size{W: inner.Max.W, H: inner.Max.H}}
	return f.buildRows(ctx, children, childConstraints, available)
}

func (f *FlowStack) layoutRows(ctx *Context, content Rect, children []Component) []flowRow {
	available := content.W
	if available <= 0 {
		available = math.MaxFloat64
	}
	childConstraints := Constraints{Max: Size{W: content.W, H: content.H}}
	return f.buildRows(ctx, children, childConstraints, available)
}

func (f *FlowStack) buildRows(ctx *Context, children []Component, cs Constraints, available float64) []flowRow {
	rows := make([]flowRow, 0)
	current := flowRow{items: make([]flowItem, 0)}
	wrapWidth := available
	for _, ch := range children {
		size := measureComponent(ctx, f, ch, cs)
		item := flowItem{component: ch, size: size, weight: FlexWeight(ch)}
		itemWidth := size.W
		if itemWidth < 0 {
			itemWidth = 0
		}
		nextWidth := itemWidth
		if current.width > 0 {
			nextWidth += current.width + f.Spacing
		}
		if wrapWidth < math.MaxFloat64 && current.width > 0 && nextWidth > wrapWidth {
			if len(current.items) > 0 {
				f.distributeFlex(&current, wrapWidth)
			}
			rows = append(rows, current)
			current = flowRow{items: make([]flowItem, 0)}
			current.items = append(current.items, item)
			current.width = itemWidth
			current.height = size.H
		} else {
			if len(current.items) > 0 && current.width > 0 {
				current.width += f.Spacing
			}
			current.items = append(current.items, item)
			current.width += itemWidth
			if size.H > current.height {
				current.height = size.H
			}
		}
	}
	if len(current.items) > 0 {
		f.distributeFlex(&current, wrapWidth)
		rows = append(rows, current)
	}
	return rows
}

func (f *FlowStack) distributeFlex(row *flowRow, available float64) {
	if row == nil || len(row.items) == 0 {
		return
	}
	if available <= 0 || available >= math.MaxFloat64 || math.IsInf(available, 1) || math.IsInf(available, -1) {
		return
	}
	extra := available - row.width
	if extra <= 1e-6 {
		return
	}
	totalWeight := 0.0
	for _, item := range row.items {
		if item.weight > 0 {
			totalWeight += item.weight
		}
	}
	if totalWeight <= 0 {
		return
	}
	for i := range row.items {
		w := row.items[i].weight
		if w <= 0 {
			continue
		}
		row.items[i].size.W += extra * (w / totalWeight)
	}
	row.width = available
}
