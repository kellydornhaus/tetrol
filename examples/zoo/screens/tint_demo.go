package screens

import (
	"fmt"
	"log"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	"github.com/kellydornhaus/layouter/examples/zoo/assets"
	"github.com/kellydornhaus/layouter/layout"
)

func buildTintDemo(ctx *layout.Context) Screen {
	var backdrop layout.Image
	if img, err := assets.LoadImage("assets/images/illustration.png"); err == nil {
		backdrop = adp.WrapImage(img)
	} else {
		log.Printf("tint demo: unable to load illustration asset: %v", err)
	}

	var puzzle layout.Image
	if img, err := assets.LoadImage("assets/images/puzzle_white.png"); err == nil {
		puzzle = adp.WrapImage(img)
	} else {
		log.Printf("tint demo: unable to load puzzle icon: %v", err)
	}

	header := layout.NewPanelContainer(layout.NewLabel("Tinted Surfaces", layout.TextStyle{
		SizeDp: 22,
		AlignH: layout.AlignCenter,
		Color:  toRGBA(250, 250, 255, 255),
	}), layout.Insets(10))
	header.SetBackgroundColor(toRGBA(36, 30, 54, 255))
	header.SetCornerRadius(10)
	header.SetFillWidth(true)

	description := layout.NewLabel(
		"Panels now accept tint-color—perfect for reusing imagery with different moods or quickly darkening backgrounds behind text.",
		layout.TextStyle{
			SizeDp: 14,
			Wrap:   true,
			Color:  toRGBA(215, 215, 230, 255),
		},
	)
	descPanel := layout.NewPanelContainer(description, layout.Insets(12))
	descPanel.SetBackgroundColor(toRGBA(26, 20, 40, 255))
	descPanel.SetCornerRadius(8)
	descPanel.SetFillWidth(true)

	makeCard := func(title, subtitle string, tint layout.Color) *layout.PanelComponent {
		card := layout.NewPanelComponent(nil)
		card.SetPadding(layout.Insets(16))
		card.SetCornerRadius(14)
		card.SetBackgroundColor(toRGBA(22, 18, 32, 255))
		card.SetFillWidth(false)
		card.SetWidth(220)
		card.SetTextStyle(layout.TextStyle{
			SizeDp: 15,
			Color:  toRGBA(252, 252, 255, 255),
			Wrap:   true,
			AlignH: layout.AlignStart,
		})
		card.SetText(fmt.Sprintf("%s\n%s", title, subtitle))
		card.SetTextMaxWidth(180)
		if backdrop != nil {
			card.SetBackgroundImage(layout.ImagePaint{
				Image:  backdrop,
				Fit:    layout.ImageFitStretch,
				AlignH: layout.AlignCenter,
				AlignV: layout.AlignCenter,
			})
		}
		if tint.A > 0 {
			card.SetTintColor(tint)
		} else {
			card.ClearTintColor()
		}
		return card
	}

	noTint := makeCard("No tint", "Original artwork stays untouched.", layout.Color{})
	warmTint := makeCard("Warm wash", "tint-color: rgba(246, 170, 70, 0.45)", toRGBA(246, 170, 70, 115))
	coolTint := makeCard("Electric blue", "tint-color: rgba(120, 150, 255, 0.40)", toRGBA(120, 150, 255, 102))
	deepTint := makeCard("Midnight mask", "tint-color: rgba(20, 16, 36, 0.65)", toRGBA(20, 16, 36, 166))

	cardFlow := layout.NewFlowStack(noTint, warmTint, coolTint, deepTint)
	cardFlow.Spacing = 16
	cardFlow.LineSpacing = 16
	cardFlow.AlignItems = layout.AlignStart

	var iconSection layout.Component
	if puzzle != nil {
		iconTitle := layout.NewLabel("Transparent icons respect tint masks", layout.TextStyle{
			SizeDp: 16,
			Color:  toRGBA(235, 235, 250, 255),
		})

		makePuzzleStack := func(tint layout.Color, caption string) *layout.VStack {
			img := layout.NewImage(puzzle)
			img.SetExplicitSize(layout.Size{W: 96, H: 96})
			img.SetPadding(layout.Insets(12))
			img.SetCornerRadius(18)
			img.SetBorder(toRGBA(90, 80, 140, 160), 1)
			if tint.A > 0 {
				img.SetTintColor(tint)
			} else {
				img.ClearTintColor()
			}

			label := layout.NewLabel(caption, layout.TextStyle{
				SizeDp: 13,
				Wrap:   true,
				AlignH: layout.AlignCenter,
				Color:  toRGBA(200, 200, 225, 255),
			})

			stack := layout.NewVStack(img, label)
			stack.Spacing = 10
			stack.AlignH = layout.AlignCenter
			return stack
		}

		plainStack := makePuzzleStack(layout.Color{}, "Tint off – pure white icon")
		tintedStack := makePuzzleStack(toRGBA(110, 200, 255, 220), "Tinted with rgba(110,200,255,0.86)")

		iconRow := layout.NewHStack(plainStack, tintedStack)
		iconRow.Spacing = 18
		iconRow.AlignV = layout.AlignStart

		iconNote := layout.NewLabel(
			"Tints only affect opaque pixels, so the transparent canvas stays untouched while the puzzle shape inherits the color.",
			layout.TextStyle{
				SizeDp: 13,
				Wrap:   true,
				Color:  toRGBA(185, 185, 215, 255),
			},
		)

		column := layout.NewVStack(iconTitle, iconRow, iconNote)
		column.Spacing = 12
		column.SetFillWidth(true)

		section := layout.NewPanelContainer(column, layout.Insets(14))
		section.SetBackgroundColor(toRGBA(18, 16, 30, 255))
		section.SetCornerRadius(10)
		section.SetFillWidth(true)
		iconSection = section
	}

	codeLabel := layout.NewLabel(
		"XML / CSS:\nPanel.hero { tint-color: rgba(20, 16, 36, 0.65); }\n<Text style=\"tint-color: rgba(246,170,70,0.45)\"/>",
		layout.TextStyle{
			SizeDp: 13,
			Wrap:   true,
			Color:  toRGBA(210, 210, 240, 255),
		},
	)
	codePanel := layout.NewPanelContainer(codeLabel, layout.Insets(12))
	codePanel.SetBackgroundColor(toRGBA(18, 16, 30, 255))
	codePanel.SetBorder(toRGBA(90, 80, 140, 255), 1)
	codePanel.SetCornerRadius(8)
	codePanel.SetFillWidth(true)

	sections := []layout.Component{header, descPanel, cardFlow}
	if iconSection != nil {
		sections = append(sections, iconSection)
	}
	sections = append(sections, codePanel)

	content := layout.NewVStack(sections...)
	content.Spacing = 18
	content.SetFillWidth(true)
	content.SetBackgroundColor(toRGBA(14, 12, 26, 255))

	wrapper := layout.NewPanelContainer(content, layout.Insets(18))
	wrapper.SetFillWidth(true)
	wrapper.SetBackgroundColor(toRGBA(10, 8, 20, 255))
	return &staticScreen{name: "Tinted Surfaces", root: wrapper}
}
