package screens

import (
	"fmt"

	"github.com/kellydornhaus/layouter/layout"
)

func buildRowAlignGallery(ctx *layout.Context) Screen {
	makeCell := func(idx int) layout.Component {
		label := layout.NewLabel(fmt.Sprintf("Item %d", idx+1), layout.TextStyle{SizeDp: 13, AlignH: layout.AlignCenter})
		panel := layout.NewPanelContainer(label, layout.Insets(6))
		panel.SetBackgroundColor(toRGBA(40+uint8(idx*25%120), 60+uint8(idx*17%120), 90+uint8(idx*11%120), 255))
		panel.SetFillWidth(true)
		return panel
	}

	makeSection := func(title string, align layout.TextAlign) layout.Component {
		grid := layout.NewGrid(3, 10)
		grid.RowAlign = align
		for i := 0; i < 5; i++ {
			grid.Add(makeCell(i))
		}
		grid.SetFillWidth(true)

		header := layout.NewPanelContainer(layout.NewLabel(title, layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter, Color: toRGBA(240, 240, 240, 255)}), layout.Insets(6))
		header.SetFillWidth(true)

		column := layout.NewVStack(header, grid)
		column.Spacing = 8
		column.SetFillWidth(true)
		wrap := layout.NewPanelContainer(column, layout.Insets(8))
		wrap.SetFillWidth(true)
		return wrap
	}

	sections := layout.NewGrid(3, 0)
	sections.SetFillWidth(true)
	sections.SetCellMinWidthPercent(1.0 / 3.0)
	sections.SetCellMaxWidthPercent(1.0 / 3.0)
	sections.Add(makeSection("RowAlign Start", layout.AlignStart))
	sections.Add(makeSection("RowAlign Center", layout.AlignCenter))
	sections.Add(makeSection("RowAlign End", layout.AlignEnd))

	root := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Row Align Gallery", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		sections,
	)
	root.Spacing = 16
	root.SetFillWidth(true)
	root.SetBackgroundColor(toRGBA(12, 12, 20, 255))

	return &staticScreen{name: "Row Align", root: root}
}
