package screens

import "github.com/kellydornhaus/layouter/layout"

func buildNesting(ctx *layout.Context) Screen {
	level3 := layout.NewPanelContainer(layout.NewLabel("Nested", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter}), layout.Insets(8))
	level3.SetBackgroundColor(toRGBA(30, 30, 50, 255))
	level3.SetFillWidth(true)
	level2 := layout.NewPanelContainer(level3, layout.Insets(8))
	level2.SetBackgroundColor(toRGBA(30, 50, 30, 255))
	level2.SetFillWidth(true)
	nested := layout.NewPanelContainer(level2, layout.Insets(8))
	nested.SetBackgroundColor(toRGBA(50, 30, 30, 255))
	nested.SetFillWidth(true)
	root := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Nesting", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		nested,
	)
	root.SetFillWidth(true)
	root.SetBackgroundColor(toRGBA(12, 12, 16, 255))
	return &staticScreen{name: "Nesting", root: root}
}
