package screens

import (
	"log"

	"github.com/kellydornhaus/layouter/layout"
)

func buildViewportColumnXML(ctx *layout.Context) Screen {
	xmlPath := "screens/layouts/viewport_column.xml"
	res, err := buildLayout(ctx, nil, xmlPath, layoutFS())
	if err != nil {
		log.Printf("viewport column xml load failed: %v", err)
		fallback := layout.NewPanelContainer(layout.NewLabel("Viewport column XML failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter}), layout.Insets(12))
		fallback.SetBackgroundColor(toRGBA(32, 24, 32, 255))
		fallback.SetFillWidth(true)
		fallback.SetFillHeight(true)
		return &staticScreen{name: "Viewport Column (XML)", root: fallback}
	}
	return &staticScreen{name: "Viewport Column (XML)", root: res.Root}
}
