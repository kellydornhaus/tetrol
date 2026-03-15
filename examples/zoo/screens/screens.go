package screens

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	cmp "github.com/kellydornhaus/layouter/examples/zoo/internal/components"
	"github.com/kellydornhaus/layouter/layout"
)

type Screen interface {
	Name() string
	Root() layout.Component
	UpdateFrame()
}

type staticScreen struct {
	name string
	root layout.Component
}

type scrollScreen struct {
	staticScreen
	scroll  *cmp.Scroll
	preview *cmp.ScrollPreview
}

func (s *scrollScreen) UpdateFrame() {
	if s.scroll == nil {
		return
	}
	_, dy := ebiten.Wheel()
	if dy != 0 {
		s.scroll.ScrollBy(-dy * 32)
		if s.preview != nil {
			s.preview.Trigger(s.scroll.Offset())
		}
	} else if s.preview != nil {
		s.preview.Tick()
	}
}

func (s *staticScreen) Name() string           { return s.name }
func (s *staticScreen) Root() layout.Component { return s.root }
func (s *staticScreen) UpdateFrame()           {}

type dirtyScreen struct {
	name string
	root layout.Component
	text *layout.PanelComponent
	tick int
}

func (s *dirtyScreen) Name() string           { return s.name }
func (s *dirtyScreen) Root() layout.Component { return s.root }
func (s *dirtyScreen) UpdateFrame() {
	s.tick++
	if s.tick%60 == 0 {
		if s.text.Text() == "Tick" {
			s.text.SetText("Tock")
		} else {
			s.text.SetText("Tick")
		}
	}
}

func NewScreens(ctx *layout.Context) []Screen {
	return []Screen{
		buildTextBasics(ctx),
		buildStacks(ctx),
		buildAlignment(ctx),
		buildNesting(ctx),
		buildDirty(ctx),
		buildVisibilityDemo(ctx),
		buildImages(ctx),
		buildTintDemo(ctx),
		buildAssetGallery(ctx),
		buildJustifyGallery(ctx),
		buildSpacingAlignGallery(ctx),
		buildWrapGallery(ctx),
		buildGrid(ctx),
		buildGridAlignment(ctx),
		buildRowAlignGallery(ctx),
		buildRadiusShowcase(ctx),
		buildPositioningDemo(ctx),
		buildViewportColumn(ctx),
		buildViewportColumnXML(ctx),
		buildOutlines(ctx),
		func() Screen {
			s, err := buildXMLDemo(ctx)
			if err != nil {
				log.Printf("xml demo init failed: %v", err)
				placeholder := layout.NewLabel("XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(30, 18, 24, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "XML Demo", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLAssetsDemo(ctx)
			if err != nil {
				log.Printf("assets xml demo init failed: %v", err)
				placeholder := layout.NewLabel("Asset gallery failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(22, 20, 34, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "Assets XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLAutoFontDemo(ctx)
			if err != nil {
				log.Printf("auto font xml init failed: %v", err)
				placeholder := layout.NewLabel("Auto font demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(34, 20, 20, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "Auto Font XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLRadiusDemo(ctx)
			if err != nil {
				log.Printf("radius xml init failed: %v", err)
				placeholder := layout.NewLabel("Rounded XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(36, 18, 34, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "Rounded XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLThemeDemo(ctx)
			if err != nil {
				log.Printf("theme xml init failed: %v", err)
				placeholder := layout.NewLabel("Theme toggle demo failed", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(28, 20, 36, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "Theme XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLFlowStackDemo(ctx)
			if err != nil {
				log.Printf("flowstack xml init failed: %v", err)
				placeholder := layout.NewLabel("FlowStack XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(24, 20, 32, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "FlowStack XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLTemplateDemo(ctx)
			if err != nil {
				log.Printf("templates xml init failed: %v", err)
				placeholder := layout.NewLabel("Templates XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(18, 24, 36, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "Templates XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLScoreboardDemo(ctx)
			if err != nil {
				log.Printf("scoreboard xml init failed: %v", err)
				placeholder := layout.NewLabel("Scoreboard XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(10, 16, 28, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "Scoreboard XML", root: panel}
			}
			return s
		}(),
		func() Screen {
			s, err := buildXMLAlignSelfDemo(ctx)
			if err != nil {
				log.Printf("align-self xml init failed: %v", err)
				placeholder := layout.NewLabel("align-self XML demo failed to load", layout.TextStyle{SizeDp: 16, AlignH: layout.AlignCenter})
				panel := layout.NewPanelContainer(placeholder, layout.Insets(16))
				panel.SetBackgroundColor(toRGBA(20, 24, 32, 255))
				panel.SetFillWidth(true)
				return &staticScreen{name: "align-self XML", root: panel}
			}
			return s
		}(),
	}
}

func toRGBA(r, g, b, a uint8) layout.Color {
	return layout.Color{R: r, G: g, B: b, A: a}
}
