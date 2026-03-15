package screens

import "github.com/kellydornhaus/layouter/layout"

func buildStacks(ctx *layout.Context) Screen {
	mkItem := func(label string, bg layout.Color) layout.Component {
		panel := layout.NewPanelContainer(layout.NewLabel(label, layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter, Color: toRGBA(240, 240, 240, 255)}), layout.Insets(6))
		panel.SetBackgroundColor(bg)
		panel.SetFillWidth(true)
		return panel
	}
	mkRow := func(w1, w2 float64) layout.Component {
		a := mkItem("A", toRGBA(80, 40, 40, 255))
		b := mkItem("B", toRGBA(40, 80, 40, 255))
		c := mkItem("C", toRGBA(40, 40, 80, 255))
		row := layout.NewHStack(a, layout.NewSpacer(w1), b, layout.NewSpacer(w2), c)
		row.Spacing = 8
		row.Justify = layout.JustifyStart
		row.AlignV = layout.AlignCenter
		row.SetFillWidth(true)
		return row
	}

	col := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Stacks & Spacers", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		mkRow(1, 2),
		mkRow(1, 1),
		mkRow(2, 1),
	)
	col.Spacing = 10
	col.SetFillWidth(true)
	col.SetBackgroundColor(toRGBA(10, 10, 15, 255))
	return &staticScreen{name: "Stacks", root: col}
}
