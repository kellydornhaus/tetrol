package screens

import (
	"log"

	cmp "github.com/kellydornhaus/layouter/examples/zoo/internal/components"
	"github.com/kellydornhaus/layouter/layout"
)

func buildPositioningDemo(ctx *layout.Context) Screen {
	xmlPath := "screens/layouts/positioning_demo.xml"
	res, err := buildLayout(ctx, nil, xmlPath, layoutFS())
	if err != nil {
		log.Printf("positioning xml build failed: %v", err)
		fallback := layout.NewPanelContainer(
			layout.NewLabel("Positioning XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter}),
			layout.Insets(12),
		)
		fallback.SetBackgroundColor(toRGBA(28, 20, 40, 255))
		fallback.SetFillWidth(true)
		fallback.SetFillHeight(true)
		return &staticScreen{name: "Positioning", root: fallback}
	}

	body := res.ByID["body"]
	fixed := res.ByID["fixed-helper"]
	if body == nil || fixed == nil {
		log.Printf("positioning xml missing required elements: body=%v fixed=%v", body != nil, fixed != nil)
		return &staticScreen{name: "Positioning", root: res.Root}
	}

	scroll := cmp.NewScroll(body)
	scroll.TailPadDp = 48

	preview := cmp.NewScrollPreview()
	scrolledContent := cmp.NewOverlay(scroll, preview.Painter(scroll))

	root := layout.NewZStack(scrolledContent, fixed)

	return &scrollScreen{staticScreen: staticScreen{name: "Positioning", root: root}, scroll: scroll, preview: preview}
}
