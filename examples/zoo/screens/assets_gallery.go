package screens

import (
	"log"
	"path/filepath"
	"strings"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	assets "github.com/kellydornhaus/layouter/examples/zoo/assets"
	cmp "github.com/kellydornhaus/layouter/examples/zoo/internal/components"
	"github.com/kellydornhaus/layouter/layout"
)

func buildAssetGallery(ctx *layout.Context) Screen {
	load := func(path string) layout.Image {
		img, err := assets.LoadImage(path)
		if err != nil {
			log.Printf("assets: failed to load %s: %v", path, err)
			return nil
		}
		return adp.WrapImage(img)
	}
	buttonSquare := load("assets/images/Vector/Blue/button_square_flat.svg")
	buttonSquareGradient := load("assets/images/Vector/Blue/button_square_gradient.svg")
	slice := layout.NineSlice{Left: 18, Right: 18, Top: 18, Bottom: 18}

	vectorPaths, err := assets.ListImagePaths("assets/images/Vector", []string{".svg"}, 8)
	if err != nil {
		log.Printf("assets: listing vector images failed: %v", err)
	}
	pngPaths, err := assets.ListImagePaths("assets/images/PNG", []string{".png"}, 8)
	if err != nil {
		log.Printf("assets: listing png images failed: %v", err)
	}

	wrapWithSlice := func(img layout.Image) func(layout.Component) layout.Component {
		return func(child layout.Component) layout.Component {
			panel := layout.NewPanelComponent(child)
			panel.SetFillWidth(true)
			if img != nil {
				panel.SetBackgroundImage(layout.ImagePaint{Image: img, Slice: slice})
			} else {
				panel.SetBackgroundColor(toRGBA(20, 22, 30, 255))
			}
			panel.SetFillWidth(true)
			return panel
		}
	}

	vectorSection := makeGallerySection("Vector (SVG)", vectorPaths, wrapWithSlice(buttonSquare), layout.ImageFitContain)
	pngSection := makeGallerySection("PNG Sprites", pngPaths, wrapWithSlice(buttonSquareGradient), layout.ImageFitContain)

	sections := layout.NewHStack(vectorSection, pngSection)
	sections.Spacing = 16
	sections.AlignV = layout.AlignStart
	sections.Justify = layout.JustifyCenter

	head := layout.NewPanelContainer(layout.NewLabel("Asset Gallery", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter, Color: toRGBA(250, 250, 250, 255)}), layout.Insets(8))
	content := layout.NewVStack(head, sections)
	content.Spacing = 18
	content.AlignH = layout.AlignCenter

	scroll := cmp.NewScroll(content)
	scroll.TailPadDp = 32
	preview := cmp.NewScrollPreview()
	body := layout.NewPanelContainer(cmp.NewOverlay(scroll, preview.Painter(scroll)), layout.Insets(16))
	body.SetBackgroundColor(toRGBA(10, 10, 16, 255))
	body.SetFillWidth(true)
	return &scrollScreen{staticScreen: staticScreen{name: "Assets", root: body}, scroll: scroll, preview: preview}
}

func makeGallerySection(title string, paths []string, wrap func(layout.Component) layout.Component, fit layout.ImageFit) layout.Component {
	header := layout.NewPanelContainer(layout.NewLabel(title, layout.TextStyle{SizeDp: 16, AlignH: layout.AlignStart, Color: toRGBA(230, 230, 230, 255)}), layout.Insets(4))
	if len(paths) == 0 {
		fallback := layout.NewLabel("No assets found", layout.TextStyle{SizeDp: 12, AlignH: layout.AlignCenter, Color: toRGBA(200, 200, 200, 255)})
		row := layout.NewPanelContainer(fallback, layout.Insets(6))
		section := layout.NewVStack(header, row)
		section.Spacing = 6
		section.AlignH = layout.AlignCenter
		return section
	}

	cards := make([]layout.Component, 0, len(paths))
	for _, p := range paths {
		img, err := assets.LoadImage(p)
		if err != nil {
			log.Printf("assets: failed to load %s: %v", p, err)
			continue
		}
		imageCmp := layout.NewImage(adp.WrapImage(img))
		imageCmp.SetFit(fit)
		imageCmp.SetAlignment(layout.AlignCenter, layout.AlignCenter)

		label := layout.NewLabel(displayName(p), layout.TextStyle{SizeDp: 10, AlignH: layout.AlignCenter, Color: toRGBA(210, 210, 210, 255)})

		stack := layout.NewVStack(imageCmp, label)
		stack.Spacing = 4
		stack.AlignH = layout.AlignCenter

		inner := layout.NewPanelContainer(stack, layout.Insets(8))
		card := layout.Component(inner)
		if wrap != nil {
			card = wrap(inner)
		}
		cards = append(cards, card)
	}

	if len(cards) == 0 {
		return makeGallerySection(title, nil, wrap, fit)
	}

	grid := layout.NewGrid(4, 12)
	for _, card := range cards {
		grid.Add(card)
	}
	grid.Spacing = 8

	section := layout.NewVStack(header, grid)
	section.Spacing = 8
	section.AlignH = layout.AlignCenter
	return section
}

func displayName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	base = strings.ReplaceAll(base, "_", " ")
	return base
}
