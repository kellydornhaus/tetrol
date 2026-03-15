package layout

import "math"

// Grid arranges children in a responsive grid.
type Grid struct {
	*Panel
	Columns  int
	Spacing  float64
	AlignH   TextAlign
	AlignV   TextAlign
	RowAlign TextAlign
	Children []Component

	cellMinWidth Length
	cellMaxWidth Length
}

// NewGrid creates a new grid container. columns=0 means auto.
func NewGrid(columns int, spacing float64) *Grid {
	if spacing < 0 {
		spacing = 0
	}
	g := &Grid{
		Panel:    NewPanel(),
		Columns:  columns,
		Spacing:  spacing,
		AlignH:   AlignCenter,
		AlignV:   AlignCenter,
		RowAlign: AlignStart,
		Children: make([]Component, 0),
	}
	g.SetFillWidth(true)
	return g
}

func (g *Grid) dialogBase() *Base {
	if g == nil || g.Panel == nil {
		return nil
	}
	return &g.Panel.Base
}

// Add appends a child component to the grid.
func (g *Grid) Add(child Component) {
	if child == nil {
		return
	}
	g.Children = append(g.Children, child)
	attachDialogParent(child, g)
	g.SetDirty()
}

// Clear removes all children.
func (g *Grid) Clear() {
	for _, ch := range g.Children {
		if ch != nil {
			attachDialogParent(ch, nil)
		}
	}
	g.Children = g.Children[:0]
	g.SetDirty()
}

// SetColumns updates column count (0 = auto).
func (g *Grid) SetColumns(columns int) {
	if g.Columns != columns {
		g.Columns = columns
		g.SetDirty()
	}
}

// SetSpacing updates spacing between cells.
func (g *Grid) SetSpacing(spacing float64) {
	if spacing < 0 {
		spacing = 0
	}
	if g.Spacing != spacing {
		g.Spacing = spacing
		g.SetDirty()
	}
}

// SetRowAlign controls alignment of partially filled rows relative to the full row width.
func (g *Grid) SetRowAlign(align TextAlign) {
	if g.RowAlign != align {
		g.RowAlign = align
		g.SetDirty()
	}
}

// SetCellMinWidth enforces a minimum width per cell in dp units.
func (g *Grid) SetCellMinWidth(width float64) {
	g.SetCellMinWidthLength(LengthDP(width))
}

// SetCellMinWidthPercent enforces a minimum cell width relative to available width.
func (g *Grid) SetCellMinWidthPercent(percent float64) {
	g.SetCellMinWidthLength(LengthPercent(percent))
}

// SetCellMinWidthLength assigns a minimum cell width using Length units.
func (g *Grid) SetCellMinWidthLength(length Length) {
	length = normalizedLength(length)
	if g.cellMinWidth == length {
		return
	}
	g.cellMinWidth = length
	g.SetDirty()
}

// SetCellMinWidthViewportWidth enforces minimum cell width using viewport width units.
func (g *Grid) SetCellMinWidthViewportWidth(fraction float64) {
	g.SetCellMinWidthLength(LengthVW(fraction))
}

// SetCellMaxWidth limits the width per cell in dp units.
func (g *Grid) SetCellMaxWidth(width float64) {
	g.SetCellMaxWidthLength(LengthDP(width))
}

// SetCellMaxWidthPercent limits cell width relative to available width.
func (g *Grid) SetCellMaxWidthPercent(percent float64) {
	g.SetCellMaxWidthLength(LengthPercent(percent))
}

// SetCellMaxWidthLength assigns a maximum cell width using Length units.
func (g *Grid) SetCellMaxWidthLength(length Length) {
	length = normalizedLength(length)
	if g.cellMaxWidth == length {
		return
	}
	g.cellMaxWidth = length
	g.SetDirty()
}

// SetCellMaxWidthViewportWidth limits cell width using viewport width units.
func (g *Grid) SetCellMaxWidthViewportWidth(fraction float64) {
	g.SetCellMaxWidthLength(LengthVW(fraction))
}

func (g *Grid) resolveCellWidthLimits(ctx *Context, available float64) (min float64, max float64) {
	if available < 0 {
		available = 0
	}
	min = g.cellMinWidth.ResolveWidth(ctx, available)
	max = g.cellMaxWidth.ResolveWidth(ctx, available)
	if max > 0 && min > max {
		min = max
	}
	return min, max
}

func clampToBounds(value, min, max float64) float64 {
	if min > 0 && value < min {
		value = min
	}
	if max > 0 && value > max {
		value = max
	}
	return value
}

// Measure returns the desired size for the grid.
func (g *Grid) Measure(ctx *Context, cs Constraints) Size {
	if g.Visibility() == VisibilityCollapse {
		return Size{}
	}
	active := make([]Component, 0, len(g.Children))
	for _, child := range g.Children {
		if child == nil || !participatesInLayout(child) {
			continue
		}
		active = append(active, child)
	}
	if len(active) == 0 {
		return g.resolveSize(ctx, cs, Size{})
	}

	inner := g.contentConstraints(ctx, cs)
	availableForCells := inner.Max.W
	if availableForCells <= 0 {
		availableForCells = cs.Max.W
	}
	minCellW, maxCellW := g.resolveCellWidthLimits(ctx, availableForCells)
	maxW := 0.0
	maxH := 0.0
	for _, child := range active {
		childCs := inner
		if maxCellW > 0 && (childCs.Max.W <= 0 || maxCellW < childCs.Max.W) {
			childCs.Max.W = maxCellW
		}
		if minCellW > 0 && minCellW > childCs.Min.W {
			childCs.Min.W = minCellW
		}
		if childCs.Max.W > 0 && childCs.Min.W > childCs.Max.W {
			childCs.Min.W = childCs.Max.W
		}
		sz := measureComponent(ctx, g, child, childCs)
		width := clampToBounds(sz.W, minCellW, maxCellW)
		if width > maxW {
			maxW = width
		}
		if sz.H > maxH {
			maxH = sz.H
		}
	}

	cols := g.Columns
	if cols <= 0 {
		cols = int(math.Sqrt(float64(len(active))))
		if cols < 1 {
			cols = 1
		}
		if availableForCells > 0 && maxW > 0 {
			spacing := g.Spacing
			maxCols := int((availableForCells + spacing) / (maxW + spacing))
			if maxCols < 1 {
				maxCols = 1
			}
			if cols > maxCols {
				cols = maxCols
			}
		}
	}
	if cols > len(active) {
		cols = len(active)
	}
	rows := (len(active) + cols - 1) / cols

	totalW := float64(cols)*maxW + float64(maxInt(cols-1, 0))*g.Spacing
	totalH := float64(rows)*maxH + float64(maxInt(rows-1, 0))*g.Spacing

	result := g.resolveSize(ctx, cs, Size{W: totalW, H: totalH})
	logSelfMeasure(ctx, g, cs, result)
	return result
}

// Layout positions children inside bounds.
func (g *Grid) Layout(ctx *Context, parent Component, bounds Rect) {
	if g.Visibility() == VisibilityCollapse {
		g.SetFrame(parent, Rect{})
		return
	}
	g.SetFrame(parent, bounds)
	logLayoutBounds(ctx, g, parent, bounds)
	content := g.ContentBounds()
	g.refreshAutoTextSize(ctx, content)
	defer layoutOutOfFlowChildren(ctx, g, g.Children, content)
	if content.W <= 0 || content.H <= 0 {
		return
	}

	active := make([]Component, 0, len(g.Children))
	for _, child := range g.Children {
		if child == nil || !participatesInLayout(child) {
			continue
		}
		active = append(active, child)
	}
	if len(active) == 0 {
		return
	}

	childConstraints := Constraints{Max: Size{W: content.W, H: content.H}}
	minCellW, maxCellW := g.resolveCellWidthLimits(ctx, content.W)
	if maxCellW > 0 && (childConstraints.Max.W <= 0 || maxCellW < childConstraints.Max.W) {
		childConstraints.Max.W = maxCellW
	}
	if minCellW > 0 && minCellW > childConstraints.Min.W {
		childConstraints.Min.W = minCellW
	}
	if childConstraints.Max.W > 0 && childConstraints.Min.W > childConstraints.Max.W {
		childConstraints.Min.W = childConstraints.Max.W
	}
	baseChildConstraints := childConstraints

	sizes := make([]Size, len(active))
	widths := make([]float64, len(active))
	weights := make([]float64, len(active))
	maxW := 0.0
	maxH := 0.0
	for i, child := range active {
		childCs := baseChildConstraints
		sz := measureComponent(ctx, g, child, childCs)
		sizes[i] = sz
		width := clampToBounds(sz.W, minCellW, maxCellW)
		widths[i] = width
		weights[i] = FlexWeight(child)
		if width > maxW {
			maxW = width
		}
		if sz.H > maxH {
			maxH = sz.H
		}
	}

	cols := g.Columns
	if cols <= 0 {
		spacing := g.Spacing
		if maxW > 0 {
			cols = int((content.W + spacing) / (maxW + spacing))
		}
		if cols < 1 {
			cols = 1
		}
	}
	if cols > len(active) {
		cols = len(active)
	}
	if cols < 1 {
		cols = 1
	}

	rows := (len(active) + cols - 1) / cols
	spacing := g.Spacing
	cellW := clampToBounds(maxW, minCellW, maxCellW)
	cellH := maxH
	if content.W > 0 && cols > 0 {
		available := content.W
		if cols > 1 {
			available -= spacing * float64(cols-1)
		}
		if available < 0 {
			available = 0
		}
		widthPerCol := available / float64(cols)
		if widthPerCol > 0 {
			target := widthPerCol
			if cellW > 0 {
				target = minFloat(cellW, widthPerCol)
			}
			cellW = clampToBounds(target, minCellW, maxCellW)
		}
	}

	totalW := float64(cols)*cellW + float64(maxInt(cols-1, 0))*spacing
	totalH := float64(rows)*cellH + float64(maxInt(rows-1, 0))*spacing

	offsetX := content.X
	switch g.AlignH {
	case AlignCenter:
		offsetX = content.X + maxFloat(0, (content.W-totalW)/2)
	case AlignEnd:
		offsetX = content.X + maxFloat(0, content.W-totalW)
	}

	offsetY := content.Y
	switch g.AlignV {
	case AlignCenter:
		offsetY = content.Y + maxFloat(0, (content.H-totalH)/2)
	case AlignEnd:
		offsetY = content.Y + maxFloat(0, content.H-totalH)
	}

	rowsCount := (len(active) + cols - 1) / cols
	for row := 0; row < rowsCount; row++ {
		rowStart := row * cols
		rowCount := cols
		if remaining := len(active) - rowStart; remaining < cols {
			rowCount = remaining
		}
		if rowCount <= 0 {
			continue
		}

		rowWidth := float64(rowCount)*cellW + float64(maxInt(rowCount-1, 0))*spacing
		rowOffset := 0.0
		if rowWidth < totalW {
			switch g.RowAlign {
			case AlignCenter:
				rowOffset = (totalW - rowWidth) / 2
			case AlignEnd:
				rowOffset = totalW - rowWidth
			}
			if rowOffset < 0 {
				rowOffset = 0
			}
		}

		rowWeight := 0.0
		for i := 0; i < rowCount; i++ {
			if w := weights[rowStart+i]; w > 0 {
				rowWeight += w
			}
		}
		rowExtra := 0.0
		if rowWeight > 0 && rowWidth < totalW {
			rowExtra = totalW - rowWidth
			rowOffset = 0
		}

		x := offsetX + rowOffset
		y := offsetY + float64(row)*(cellH+spacing)
		for i := 0; i < rowCount; i++ {
			index := rowStart + i
			child := active[index]
			width := cellW
			if rowExtra > 0 && weights[index] > 0 {
				width += rowExtra * (weights[index] / rowWeight)
			}

			rect := Rect{X: x, Y: y, W: width, H: cellH}

			alignH := g.AlignH
			alignV := g.AlignV
			if override, ok := AlignSelfOf(child); ok {
				alignH = override
				alignV = override
			}

			fillWidth := false
			if fw, ok := child.(interface{ FillWidth() bool }); ok && fw.FillWidth() {
				fillWidth = true
			}
			if rowExtra > 0 && weights[index] > 0 {
				fillWidth = true
			}

			fillHeight := false
			if fh, ok := child.(interface{ FillHeight() bool }); ok && fh.FillHeight() {
				fillHeight = true
			}

			widthHint := widths[index]
			if !fillWidth {
				effWidth := widthHint
				if effWidth <= 0 || effWidth > width {
					effWidth = width
				}
				switch alignH {
				case AlignCenter:
					rect.X = x + (width-effWidth)/2
				case AlignEnd:
					rect.X = x + (width - effWidth)
				default:
					rect.X = x
				}
				rect.W = effWidth
			} else {
				rect.X = x
				rect.W = width
			}

			heightHint := sizes[index].H
			if !fillHeight {
				effHeight := heightHint
				if effHeight <= 0 || effHeight > cellH {
					if cellH > 0 {
						if effHeight <= 0 || effHeight > cellH {
							effHeight = cellH
						}
					} else if effHeight < 0 {
						effHeight = 0
					}
				}
				switch alignV {
				case AlignCenter:
					rect.Y = y + (cellH-effHeight)/2
				case AlignEnd:
					rect.Y = y + (cellH - effHeight)
				default:
					rect.Y = y
				}
				rect.H = effHeight
			} else {
				rect.Y = y
				rect.H = cellH
			}

			layoutFlowChild(ctx, g, child, rect)
			if child.Dirty() {
				g.SetDirty()
			}

			x += width
			if i < rowCount-1 {
				x += spacing
			}
		}
	}

}

// DrawTo renders the grid with caching.
func (g *Grid) DrawTo(ctx *Context, dst Surface) {
	if !g.ShouldRender() {
		g.Base.releaseCache()
		return
	}
	for _, child := range g.Children {
		if child != nil && child.Dirty() {
			g.SetDirty()
			break
		}
	}
	g.DrawPanelChildrenWithOwner(ctx, dst, g, func(target Surface) { g.Render(ctx, target) })
}

// Render draws children in order.
func (g *Grid) Render(ctx *Context, dst Surface) {
	ordered := orderByZIndex(g.Children)
	for _, child := range ordered {
		if child == nil || !rendersToSurface(child) {
			continue
		}
		child.DrawTo(ctx, dst)
	}
}
