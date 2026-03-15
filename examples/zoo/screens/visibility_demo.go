package screens

import (
	"log"

	"github.com/kellydornhaus/layouter/layout"
)

type visibilityDemoItem struct {
	comp          layout.Component
	visibleFrames int
	hiddenFrames  int
	timer         int
}

type visibilityScreen struct {
	name  string
	root  layout.Component
	items []visibilityDemoItem
}

func (s *visibilityScreen) Name() string           { return s.name }
func (s *visibilityScreen) Root() layout.Component { return s.root }

func (s *visibilityScreen) UpdateFrame() {
	for i := range s.items {
		item := &s.items[i]
		if item.comp == nil {
			continue
		}
		if item.timer > 0 {
			item.timer--
			continue
		}
		if layout.VisibilityOf(item.comp) == layout.VisibilityVisible {
			layout.SetVisibility(item.comp, layout.VisibilityHidden)
			item.timer = item.hiddenFrames
		} else {
			layout.SetVisibility(item.comp, layout.VisibilityVisible)
			item.timer = item.visibleFrames
		}
	}
}

func buildVisibilityDemo(ctx *layout.Context) Screen {
	xmlPath := "screens/layouts/visibility_demo.xml"
	res, err := buildLayout(ctx, nil, xmlPath, layoutFS())
	if err != nil {
		log.Printf("visibility demo load failed: %v", err)
		return visibilityFallbackScreen("Visibility FX demo failed to load")
	}
	if res.Root == nil {
		log.Printf("visibility demo root missing")
		return visibilityFallbackScreen("Visibility FX root missing")
	}

	addItem := func(id string, visibleFrames int, hiddenFrames int, list *[]visibilityDemoItem) {
		comp := res.ByID[id]
		if comp == nil {
			log.Printf("visibility demo missing component id=%s", id)
			return
		}
		timer := 0
		if layout.VisibilityOf(comp) == layout.VisibilityVisible {
			timer = visibleFrames
		} else {
			timer = hiddenFrames
		}
		*list = append(*list, visibilityDemoItem{
			comp:          comp,
			visibleFrames: visibleFrames,
			hiddenFrames:  hiddenFrames,
			timer:         timer,
		})
	}

	items := make([]visibilityDemoItem, 0, 4)
	addItem("slow-card", 360, 360, &items)   // ~6s per state
	addItem("snappy-card", 240, 240, &items) // ~4s per state
	addItem("stagger-a", 300, 300, &items)
	addItem("stagger-b", 300, 300, &items)

	// Offset stagger-b so that it starts hidden for half a cycle.
	for i := range items {
		if items[i].comp == res.ByID["stagger-b"] {
			items[i].timer /= 2
			break
		}
	}

	return &visibilityScreen{
		name:  "Visibility FX",
		root:  res.Root,
		items: items,
	}
}

func visibilityFallbackScreen(message string) Screen {
	fallback := layout.NewPanelContainer(
		layout.NewLabel(message, layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter}),
		layout.Insets(12),
	)
	fallback.SetBackgroundColor(toRGBA(34, 24, 42, 255))
	fallback.SetFillWidth(true)
	fallback.SetFillHeight(true)
	return &staticScreen{name: "Visibility FX", root: fallback}
}
