package layout

import (
	"testing"
)

func TestPublicAPIAccessors(t *testing.T) {
	renderer := &fakeRenderer{}
	text := TextEngineFuncs{
		MeasureFunc: func(text string, style TextStyle, maxWidthPx int) (int, int) {
			if style.SizeDp <= 0 {
				style.SizeDp = 12
			}
			return len(text) * int(style.SizeDp), int(style.SizeDp)
		},
		DrawFunc: func(dst Surface, text string, rect PxRect, style TextStyle) {},
	}
	ctx := NewContext(nil, renderer, text)
	ctx.Scale = 2
	if got := ctx.ViewportSize(); got != (Size{}) {
		t.Fatalf("expected zero viewport before layout, got %+v", got)
	}

	label := NewLabel("Hello API", TextStyle{SizeDp: 14})
	panel := NewPanelContainer(label, Insets(4))
	SetFlexWeight(panel, 2)
	if FlexWeight(panel) != 2 {
		t.Fatalf("expected flex weight 2, got %.2f", FlexWeight(panel))
	}

	root := NewVStack(panel)
	root.Spacing = 6
	root.SetFillWidth(true)

	canvas := &fakeSurface{w: 200, h: 120}
	LayoutAndDraw(ctx, root, canvas)

	inf := Infinite()
	if inf.Max != (Size{}) {
		t.Fatalf("expected Infinite to have zero max, got %+v", inf.Max)
	}
	tight := Tight(Size{W: 80, H: 60})
	if tight.Min.W != 80 || tight.Max.H != 60 {
		t.Fatalf("unexpected Tight constraints: %+v", tight)
	}
	tightW := TightWidth(32, 100)
	if tightW.Min.W != 32 || tightW.Max.H != 100 {
		t.Fatalf("unexpected TightWidth constraints: %+v", tightW)
	}
	tightH := TightHeight(24, 50)
	if tightH.Min.H != 24 || tightH.Max.W != 50 {
		t.Fatalf("unexpected TightHeight constraints: %+v", tightH)
	}

	if !participatesInLayout(panel) {
		t.Fatalf("panel should participate in layout")
	}
	panel.SetVisibility(VisibilityHidden)
	if !participatesInLayout(panel) {
		t.Fatalf("hidden panel should participate in layout")
	}
	panel.SetVisibility(VisibilityCollapse)
	if participatesInLayout(panel) {
		t.Fatalf("collapsed panel should not participate in layout")
	}
	panel.SetVisibility(VisibilityVisible)
	if !rendersToSurface(panel) {
		t.Fatalf("visible panel should render")
	}
}

func TestAdaptersAndHelpersReachable(t *testing.T) {
	callsMeasure := false
	callsDraw := false
	engine := TextEngineFuncs{
		MeasureFunc: func(text string, style TextStyle, maxWidthPx int) (int, int) {
			callsMeasure = true
			return 10, 10
		},
		DrawFunc: func(dst Surface, text string, rect PxRect, style TextStyle) {
			callsDraw = true
		},
	}
	engine.Measure("hi", TextStyle{}, 100)
	engine.Draw(&fakeSurface{}, "hi", PxRect{}, TextStyle{})
	if !callsMeasure || !callsDraw {
		t.Fatalf("expected TextEngineFuncs callbacks to run")
	}
}
