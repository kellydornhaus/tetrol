package screens

import (
	"image/color"

	"github.com/kellydornhaus/layouter/layout"
)

func buildOutlines(ctx *layout.Context) Screen {
	mk := func(label string, bg, border color.RGBA) layout.Component {
		panel := layout.NewPanelContainer(layout.NewLabel(label, layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter, Color: toRGBA(240, 240, 240, 255)}), layout.Insets(8))
		panel.SetBackgroundColor(toRGBA(bg.R, bg.G, bg.B, bg.A))
		panel.SetBorder(toRGBA(border.R, border.G, border.B, border.A), 2)
		panel.SetFillWidth(true)
		return panel
	}
	row := layout.NewHStack(
		mk("Outline A", color.RGBA{30, 30, 50, 255}, color.RGBA{180, 80, 80, 255}),
		mk("Outline B", color.RGBA{30, 50, 30, 255}, color.RGBA{80, 180, 80, 255}),
		mk("Outline C", color.RGBA{50, 30, 30, 255}, color.RGBA{80, 80, 180, 255}),
	)
	row.Spacing = 8
	root := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Outlines", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		row,
	)
	root.SetFillWidth(true)
	root.SetBackgroundColor(toRGBA(10, 10, 14, 255))
	return &staticScreen{name: "Outlines", root: root}
}
