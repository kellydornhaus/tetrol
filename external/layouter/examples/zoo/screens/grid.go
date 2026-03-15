package screens

import (
	"fmt"

	"github.com/kellydornhaus/layouter/layout"
)

func buildGrid(ctx *layout.Context) Screen {
	cells := []layout.Component{}
	for i := 0; i < 12; i++ {
		t := layout.NewLabel(fmt.Sprintf("%d", i+1), layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter, Color: toRGBA(230, 230, 230, 255)})
		card := layout.NewPanelContainer(t, layout.Insets(12))
		card.SetBackgroundColor(toRGBA(30+uint8(i*8%80), 30+uint8(i*5%80), 60, 255))
		card.SetFillWidth(true)
		wrap := layout.NewPanelContainer(card, layout.Insets(6))
		wrap.SetFillWidth(true)
		cells = append(cells, wrap)
	}
	grid := layout.NewGrid(3, 0)
	grid.SetFillWidth(true)
	grid.SetCellMinWidthPercent(1.0 / 3.0)
	grid.SetCellMaxWidthPercent(1.0 / 3.0)
	for _, cell := range cells {
		grid.Add(cell)
	}
	column := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Grid (3 columns)", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		grid,
	)
	column.SetFillWidth(true)
	column.SetBackgroundColor(toRGBA(10, 10, 14, 255))
	return &staticScreen{name: "Grid", root: column}
}
