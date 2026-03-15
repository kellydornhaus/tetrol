package screens

import "github.com/kellydornhaus/layouter/layout"

func buildAlignment(ctx *layout.Context) Screen {
	mkCell := func(label string, align layout.TextAlign, bg layout.Color) layout.Component {
		t := layout.NewLabel(label, layout.TextStyle{SizeDp: 14, AlignH: align})
		inner := layout.NewPanelContainer(t, layout.Insets(12))
		inner.SetFillWidth(true)
		inner.SetBackgroundColor(bg)
		card := layout.NewPanelContainer(inner, layout.Insets(6))
		card.SetFillWidth(true)
		return card
	}
	grid := layout.NewGrid(3, 0)
	grid.SetFillWidth(true)
	grid.SetCellMinWidthPercent(1.0 / 3.0)
	grid.SetCellMaxWidthPercent(1.0 / 3.0)
	grid.Add(mkCell("Left", layout.AlignStart, toRGBA(60, 30, 30, 255)))
	grid.Add(mkCell("Center", layout.AlignCenter, toRGBA(30, 60, 30, 255)))
	grid.Add(mkCell("Right", layout.AlignEnd, toRGBA(30, 30, 60, 255)))
	column := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Alignment", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		grid,
	)
	column.SetFillWidth(true)
	column.SetBackgroundColor(toRGBA(14, 14, 20, 255))
	return &staticScreen{name: "Alignment", root: column}
}
