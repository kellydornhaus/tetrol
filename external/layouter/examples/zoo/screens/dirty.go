package screens

import "github.com/kellydornhaus/layouter/layout"

func buildDirty(ctx *layout.Context) Screen {
	t := layout.NewLabel("Tick", layout.TextStyle{SizeDp: 18, AlignH: layout.AlignCenter, Color: toRGBA(255, 255, 0, 255)})
	root := layout.NewVStack(
		layout.NewPanelContainer(layout.NewLabel("Dirty Demo", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter}), layout.Insets(8)),
		layout.NewSpacer(1),
		layout.NewPanelContainer(t, layout.Insets(8)),
		layout.NewSpacer(1),
		layout.NewPanelContainer(layout.NewLabel("Changes every second", layout.TextStyle{SizeDp: 12, AlignH: layout.AlignCenter, Color: toRGBA(200, 200, 200, 255)}), layout.Insets(8)),
	)
	root.SetFillWidth(true)
	root.SetBackgroundColor(toRGBA(8, 8, 12, 255))
	return &dirtyScreen{name: "Dirty", root: root, text: t}
}
