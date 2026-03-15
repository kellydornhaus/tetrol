package screens

import "github.com/kellydornhaus/layouter/layout"

func buildRadiusShowcase(ctx *layout.Context) Screen {
	title := layout.NewLabel("Rounded Corners", layout.TextStyle{
		SizeDp: 22,
		AlignH: layout.AlignCenter,
		Color:  toRGBA(250, 250, 255, 255),
	})
	titlePanel := layout.NewPanelContainer(title, layout.Insets(10))
	titlePanel.SetBackgroundColor(toRGBA(32, 24, 48, 255))
	titlePanel.SetCornerRadius(12)
	titlePanel.SetFillWidth(true)

	description := layout.NewLabel("Panels and buttons now support corner radii. These samples show uniform, per-corner, and pill styles along with rounded borders.", layout.TextStyle{
		SizeDp: 14,
		Color:  toRGBA(220, 220, 235, 255),
		Wrap:   true,
	})
	descPanel := layout.NewPanelContainer(description, layout.Insets(10))
	descPanel.SetBackgroundColor(toRGBA(28, 18, 42, 255))
	descPanel.SetCornerRadius(8)
	descPanel.SetFillWidth(true)

	makeSample := func(title, subtitle string, baseColor layout.Color, decorate func(*layout.PanelComponent)) layout.Component {
		titleLabel := layout.NewLabel(title, layout.TextStyle{
			SizeDp: 16,
			Color:  toRGBA(250, 250, 255, 255),
		})
		subLabel := layout.NewLabel(subtitle, layout.TextStyle{
			SizeDp: 13,
			Color:  toRGBA(215, 215, 230, 255),
			Wrap:   true,
		})
		textCol := layout.NewVStack(titleLabel, subLabel)
		textCol.Spacing = 4
		textCol.SetFillWidth(true)
		card := layout.NewPanelContainer(textCol, layout.Insets(14))
		card.SetBackgroundColor(baseColor)
		card.SetFillWidth(true)
		card.SetCornerRadius(10)
		if decorate != nil {
			decorate(card)
		}
		return card
	}

	uniform := makeSample("Uniform radius", "SetCornerRadius(16) applies the same rounding to every corner.", toRGBA(53, 34, 68, 255), func(p *layout.PanelComponent) {
		p.SetCornerRadius(16)
	})

	perCorner := makeSample("Per-corner radii", "CornerRadii can emphasize specific edges. Here we exaggerate the top corners.", toRGBA(42, 54, 78, 255), func(p *layout.PanelComponent) {
		p.SetCornerRadii(layout.CornerRadii{
			TopLeft:     24,
			TopRight:    24,
			BottomRight: 6,
			BottomLeft:  6,
		})
	})

	withBorder := makeSample("Rounded border", "Borders follow the same radii when the renderer supports rounded strokes.", toRGBA(58, 40, 52, 255), func(p *layout.PanelComponent) {
		p.SetCornerRadius(12)
		p.SetBorder(toRGBA(248, 160, 202, 255), 2)
	})

	makeButton := func(label string, fill layout.Color, text layout.Color) layout.Component {
		btn := layout.NewPanelContainer(layout.NewLabel(label, layout.TextStyle{
			SizeDp: 14,
			Color:  text,
			AlignH: layout.AlignCenter,
		}), layout.Insets(8))
		btn.SetBackgroundColor(fill)
		btn.SetCornerRadius(999) // pill
		return btn
	}
	buttonRow := layout.NewHStack(
		makeButton("Primary", toRGBA(120, 90, 200, 255), toRGBA(255, 255, 255, 255)),
		layout.NewSpacer(1),
		makeButton("Secondary", toRGBA(60, 44, 102, 255), toRGBA(210, 210, 240, 255)),
		layout.NewSpacer(1),
		makeButton("Ghost", toRGBA(32, 24, 48, 255), toRGBA(220, 210, 255, 255)),
	)
	buttonRow.Spacing = 10
	buttonRow.SetFillWidth(true)

	buttonCard := makeSample("Pill buttons", "Large radii produce pill shapes—great for chips, tabs, or call-to-action buttons.", toRGBA(30, 28, 58, 255), func(p *layout.PanelComponent) {
		p.SetCornerRadius(14)
		p.SetBackgroundColor(toRGBA(36, 34, 70, 255))
		if p.Child != nil {
			content := layout.NewVStack(p.Child, layout.NewPanelContainer(buttonRow, layout.Insets(6)))
			content.Spacing = 10
			content.SetFillWidth(true)
			p.SetChild(content)
		}
	})

	samples := layout.NewVStack(uniform, perCorner, withBorder, buttonCard)
	samples.Spacing = 14
	samples.SetFillWidth(true)

	rootColumn := layout.NewVStack(titlePanel, descPanel, samples)
	rootColumn.Spacing = 16
	rootColumn.SetFillWidth(true)
	rootColumn.SetBackgroundColor(toRGBA(18, 16, 30, 255))

	wrapper := layout.NewPanelContainer(rootColumn, layout.Insets(18))
	wrapper.SetFillWidth(true)
	wrapper.SetBackgroundColor(toRGBA(10, 8, 18, 255))
	return &staticScreen{name: "Rounded Corners", root: wrapper}
}
