package screens

import "github.com/kellydornhaus/layouter/layout"

func buildTextBasics(ctx *layout.Context) Screen {
	title := layout.NewLabel("Text Basics", layout.TextStyle{SizeDp: 22, AlignH: layout.AlignCenter, Color: toRGBA(255, 255, 255, 255)})
	line1 := layout.NewLabel("Hello, world!", layout.TextStyle{SizeDp: 16, Color: toRGBA(240, 240, 240, 255)})
	line2 := layout.NewLabel("Wrapping within a box to show MeasureWithWrap works properly.", layout.TextStyle{SizeDp: 14, Wrap: true, Color: toRGBA(220, 220, 220, 255)})

	titlePanel := layout.NewPanelContainer(title, layout.Insets(8))
	titlePanel.SetBackgroundColor(toRGBA(30, 30, 60, 255))
	titlePanel.SetFillWidth(true)
	line2Panel := layout.NewPanelContainer(line2, layout.Insets(8))
	line2Panel.SetBackgroundColor(toRGBA(40, 20, 20, 255))
	line2Panel.SetFillWidth(true)
	column := layout.NewVStack(
		titlePanel,
		layout.NewPanelContainer(line1, layout.Insets(6)),
		layout.NewSpacer(1),
		line2Panel,
	)
	column.Spacing = 8
	column.SetFillWidth(true)
	column.SetBackgroundColor(toRGBA(15, 15, 20, 255))
	root := column
	return &staticScreen{name: "Text Basics", root: root}
}
