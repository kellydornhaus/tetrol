package screens

import "github.com/kellydornhaus/layouter/layout"

func buildSpacingAlignGallery(ctx *layout.Context) Screen {
	mkTall := func(size float64, label string, bg layout.Color) layout.Component {
		panel := layout.NewPanelContainer(layout.NewLabel(label, layout.TextStyle{SizeDp: size, Color: toRGBA(240, 240, 240, 255), AlignH: layout.AlignCenter}), layout.Insets(6))
		panel.SetBackgroundColor(bg)
		panel.SetFillWidth(true)
		return panel
	}
	mkRow := func(align layout.TextAlign, title string) layout.Component {
		a := mkTall(12, "a", toRGBA(80, 40, 40, 255))
		b := mkTall(36, "b", toRGBA(40, 80, 40, 255))
		c := mkTall(16, "c", toRGBA(40, 40, 80, 255))
		row := layout.NewHStack(a, b, c)
		row.Spacing = 8
		row.AlignV = align
		row.Justify = layout.JustifyStart
		stack := layout.NewVStack(layout.NewPanelContainer(layout.NewLabel(title, layout.TextStyle{SizeDp: 12, Color: toRGBA(200, 200, 200, 255)}), layout.Insets(4)), row)
		stack.SetFillWidth(true)
		return stack
	}
	root := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Spacing & Vertical Align", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		mkRow(layout.AlignStart, "Top"),
		mkRow(layout.AlignCenter, "Center"),
		mkRow(layout.AlignEnd, "Bottom"),
	)
	root.SetFillWidth(true)
	root.SetBackgroundColor(toRGBA(10, 10, 14, 255))
	return &staticScreen{name: "Spacing/Align", root: root}
}
