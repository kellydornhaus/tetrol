package screens

import (
	"fmt"

	cmp "github.com/kellydornhaus/layouter/examples/zoo/internal/components"
	"github.com/kellydornhaus/layouter/layout"
)

func buildGridAlignment(ctx *layout.Context) Screen {
	makeCells := func() []layout.Component {
		cells := make([]layout.Component, 6)
		for i := 0; i < len(cells); i++ {
			t := layout.NewLabel(fmt.Sprintf("Cell %d", i+1), layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter, Color: toRGBA(235, 235, 235, 255)})
			panel := layout.NewPanelContainer(t, layout.Insets(6))
			panel.SetBackgroundColor(toRGBA(30+uint8(i*12%120), 40+uint8(i*9%120), 70, 255))
			panel.SetFillWidth(true)
			cells[i] = panel
		}
		return cells
	}

	makeSection := func(label string, align layout.TextAlign, bg layout.Color) layout.Component {
		grid := layout.NewGrid(3, 12)
		grid.AlignH = layout.AlignCenter
		grid.AlignV = align
		for _, cell := range makeCells() {
			grid.Add(cell)
		}
		title := layout.NewLabel(label, layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter, Color: toRGBA(245, 245, 245, 255)})
		constrained := layout.NewPanelComponent(grid)
		constrained.SetMinHeight(160)
		stack := layout.NewVStack(
			layout.NewPanelContainer(title, layout.Insets(6)),
			cmp.NewRowGuide(constrained),
		)
		stack.Spacing = 8
		stack.SetFillWidth(true)
		wrap := layout.NewPanelContainer(stack, layout.Insets(10))
		wrap.SetFillWidth(true)
		wrap.SetBackgroundColor(bg)
		return wrap
	}

	header := layout.NewPanelContainer(layout.NewLabel("Grid Alignment", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8))
	header.SetFillWidth(true)

	sections := layout.NewGrid(3, 0)
	sections.SetFillWidth(true)
	sections.SetCellMinWidthPercent(1.0 / 3.0)
	sections.SetCellMaxWidthPercent(1.0 / 3.0)
	sections.Add(makeSection("align-v: start", layout.AlignStart, toRGBA(24, 28, 52, 255)))
	sections.Add(makeSection("align-v: center", layout.AlignCenter, toRGBA(28, 44, 36, 255)))
	sections.Add(makeSection("align-v: end", layout.AlignEnd, toRGBA(48, 28, 36, 255)))

	rootCol := layout.NewVStack(header, sections)
	rootCol.Spacing = 12
	rootCol.SetFillWidth(true)
	rootCol.SetBackgroundColor(toRGBA(10, 10, 16, 255))

	return &staticScreen{name: "Grid Align", root: rootCol}
}
