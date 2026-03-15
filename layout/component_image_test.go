package layout

import "testing"

func TestImageWidthAutoDerivesFromHeight(t *testing.T) {
	img := NewImage(&fakeImage{w: 400, h: 200})
	img.SetExplicitSizeAuto(Size{H: 150}, true, false)
	ctx := &Context{Scale: 1}

	size := img.Measure(ctx, Constraints{})
	if !almostEqual(size.H, 150) {
		t.Fatalf("expected height 150dp, got %.2f", size.H)
	}
	if !almostEqual(size.W, 300) {
		t.Fatalf("expected width 300dp when width=auto, got %.2f", size.W)
	}
}

func TestImageHeightAutoDerivesFromWidth(t *testing.T) {
	img := NewImage(&fakeImage{w: 400, h: 200})
	img.SetExplicitSizeAuto(Size{W: 320}, false, true)
	ctx := &Context{Scale: 1}

	size := img.Measure(ctx, Constraints{})
	if !almostEqual(size.W, 320) {
		t.Fatalf("expected width 320dp, got %.2f", size.W)
	}
	if !almostEqual(size.H, 160) {
		t.Fatalf("expected height 160dp when height=auto, got %.2f", size.H)
	}
}
