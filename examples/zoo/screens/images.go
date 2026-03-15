package screens

import (
	"log"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	assets "github.com/kellydornhaus/layouter/examples/zoo/assets"
	cmp "github.com/kellydornhaus/layouter/examples/zoo/internal/components"
	"github.com/kellydornhaus/layouter/layout"
)

func buildImages(ctx *layout.Context) Screen {
	load := func(path string) layout.Image {
		img, err := assets.LoadImage(path)
		if err != nil {
			log.Printf("images: failed to load %s: %v", path, err)
			return nil
		}
		return adp.WrapImage(img)
	}

	illu := load("assets/images/illustration.png")
	buttonSquare := load("assets/images/Vector/Blue/button_square_flat.svg")
	buttonSquareGradient := load("assets/images/Vector/Blue/button_square_gradient.svg")
	flower := load("assets/images/flower.svg")
	slice := layout.NineSlice{Left: 18, Right: 18, Top: 18, Bottom: 18}

	wrapNineSlice := func(img layout.Image, padding layout.EdgeInsets, child layout.Component) *layout.PanelComponent {
		panel := layout.NewPanelComponent(child)
		panel.SetPadding(padding)
		panel.SetFillWidth(true)
		if img != nil {
			panel.SetBackgroundImage(layout.ImagePaint{Image: img, Slice: slice})
		} else {
			panel.SetBackgroundColor(toRGBA(24, 26, 34, 255))
		}
		return panel
	}

	var hero layout.Component
	if illu != nil {
		img := layout.NewImage(illu)
		img.SetFit(layout.ImageFitContain)
		img.SetAlignment(layout.AlignCenter, layout.AlignCenter)
		hero = img
	}
	heroCard := wrapNineSlice(buttonSquare, layout.Insets(12), hero)
	heroCaption := layout.NewLabel("fit=contain keeps aspect ratio while resizing with the window.", layout.TextStyle{SizeDp: 12, Wrap: true, AlignH: layout.AlignCenter, Color: toRGBA(210, 210, 210, 255)})
	heroCaption.SetTextMaxWidth(240)
	heroSection := layout.NewVStack(heroCard, layout.NewPanelContainer(heroCaption, layout.Insets(0)))
	heroSection.Spacing = 6
	heroSection.AlignH = layout.AlignCenter

	cardText := layout.NewLabel("Nine-slice backgrounds preserve beveled corners without stretching the artwork.", layout.TextStyle{SizeDp: 13, Wrap: true, Color: toRGBA(235, 235, 235, 255)})
	cardText.SetTextMaxWidth(180)
	card := wrapNineSlice(buttonSquareGradient, layout.Insets(12), cardText)

	tallText := layout.NewLabel("Resize the window to see this panel stretch. Corners stay crisp because only the center region scales.", layout.TextStyle{SizeDp: 13, Wrap: true, Color: toRGBA(225, 225, 225, 255)})
	tallText.SetTextMaxWidth(280)
	tallCard := wrapNineSlice(buttonSquareGradient, layout.Insets(16), tallText)

	var vectorSection layout.Component
	if flower != nil {
		flowerImg := layout.NewImage(flower)
		flowerImg.SetFit(layout.ImageFitContain)
		flowerImg.SetAlignment(layout.AlignCenter, layout.AlignCenter)
		vectorCard := wrapNineSlice(buttonSquare, layout.Insets(16), flowerImg)
		vectorCaption := layout.NewLabel("SVG art stays sharp because it is rasterized at load.", layout.TextStyle{SizeDp: 12, Wrap: true, AlignH: layout.AlignCenter, Color: toRGBA(210, 210, 210, 255)})
		vectorCaption.SetTextMaxWidth(220)
		stack := layout.NewVStack(vectorCard, layout.NewPanelContainer(vectorCaption, layout.Insets(0)))
		stack.Spacing = 6
		stack.AlignH = layout.AlignCenter
		vectorSection = stack
	} else {
		vectorSection = layout.NewPanelContainer(layout.NewLabel("SVG asset unavailable", layout.TextStyle{SizeDp: 12, AlignH: layout.AlignCenter}), layout.Insets(12))
	}

	header := layout.NewPanelContainer(layout.NewLabel("Images & Backgrounds", layout.TextStyle{SizeDp: 20, AlignH: layout.AlignCenter, Color: toRGBA(250, 250, 250, 255)}), layout.Insets(8))

	topRow := layout.NewHStack(heroSection, vectorSection)
	topRow.Spacing = 16
	topRow.AlignV = layout.AlignStart
	topRow.Justify = layout.JustifyCenter

	cards := layout.NewHStack(card, tallCard)
	cards.Spacing = 16
	cards.AlignV = layout.AlignStart
	cards.Justify = layout.JustifyCenter

	content := layout.NewVStack(header, topRow, cards)
	content.Spacing = 16
	content.AlignH = layout.AlignCenter

	scroll := cmp.NewScroll(content)
	scroll.TailPadDp = 32
	preview := cmp.NewScrollPreview()
	body := layout.NewPanelContainer(cmp.NewOverlay(scroll, preview.Painter(scroll)), layout.Insets(16))
	body.SetBackgroundColor(toRGBA(12, 12, 18, 255))
	body.SetFillWidth(true)
	body.SetWidthLength(layout.LengthVW(1))
	return &scrollScreen{staticScreen: staticScreen{name: "Images", root: body}, scroll: scroll, preview: preview}
}
