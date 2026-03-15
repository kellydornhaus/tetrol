package screens

import "github.com/kellydornhaus/layouter/layout"

func buildWrapGallery(ctx *layout.Context) Screen {
	paragraph := "Wrapping within a constrained width shows how MeasureWithWrap and DrawWithWrap behave with different max widths."
	mkWrapped := func(w float64, bg layout.Color) layout.Component {
		t := layout.NewLabel(paragraph, layout.TextStyle{SizeDp: 14, Wrap: true, Color: toRGBA(230, 230, 230, 255)})
		if w > 0 {
			t.SetTextMaxWidth(w)
		}
		panel := layout.NewPanelContainer(t, layout.Insets(8))
		panel.SetBackgroundColor(bg)
		if w <= 0 {
			panel.SetFillWidth(true)
			panel.SetFillHeight(true)
			panel.SetHeightLength(layout.LengthVH(1))
			panel.SetMinHeightLength(layout.LengthVH(1))
		}
		return panel
	}
	row := layout.NewHStack(
		mkWrapped(120, toRGBA(34, 34, 60, 255)),
		mkWrapped(170, toRGBA(28, 60, 34, 255)),
		mkWrapped(220, toRGBA(60, 34, 28, 255)),
		mkWrapped(0, toRGBA(24, 70, 90, 255)),
	)
	row.Spacing = 8
	row.SetFillWidth(true)
	root := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Wrap Gallery", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		row,
	)
	root.SetFillWidth(true)
	root.SetBackgroundColor(toRGBA(10, 10, 14, 255))
	return &staticScreen{name: "Wrap", root: root}
}
