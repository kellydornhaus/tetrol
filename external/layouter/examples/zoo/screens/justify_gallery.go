package screens

import "github.com/kellydornhaus/layouter/layout"

func buildJustifyGallery(ctx *layout.Context) Screen {
	mkItem := func(label string, bg layout.Color) layout.Component {
		panel := layout.NewPanelContainer(layout.NewLabel(label, layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter}), layout.Insets(6))
		panel.SetBackgroundColor(bg)
		panel.SetFillWidth(true)
		return panel
	}
	mkRow := func(j layout.Justify, title string) layout.Component {
		a := mkItem("A", toRGBA(90, 50, 50, 255))
		b := mkItem("B", toRGBA(50, 90, 50, 255))
		c := mkItem("C", toRGBA(50, 50, 90, 255))
		row := layout.NewHStack(a, b, c)
		row.Spacing = 8
		row.AlignV = layout.AlignCenter
		row.Justify = j
		row.SetFillWidth(true)
		stack := layout.NewVStack(layout.NewPanelContainer(layout.NewLabel(title, layout.TextStyle{SizeDp: 12, Color: toRGBA(200, 200, 200, 255)}), layout.Insets(4)), row)
		stack.SetFillWidth(true)
		return stack
	}
	column := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Justify Gallery", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		mkRow(layout.JustifyStart, "Start"),
		mkRow(layout.JustifyCenter, "Center"),
		mkRow(layout.JustifyEnd, "End"),
		mkRow(layout.JustifySpaceBetween, "SpaceBetween"),
	)
	column.SetFillWidth(true)
	column.SetBackgroundColor(toRGBA(10, 10, 14, 255))
	return &staticScreen{name: "Justify", root: column}
}
