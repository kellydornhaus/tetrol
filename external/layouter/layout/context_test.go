package layout

import "testing"

func TestDrawOffsetPxTracksPushAndRestore(t *testing.T) {
	ctx := &Context{}

	if x, y := ctx.DrawOffsetPx(); x != 0 || y != 0 {
		t.Fatalf("expected initial offset 0,0 got %d,%d", x, y)
	}

	restore := ctx.PushOffsetPx(3, 4)
	if x, y := ctx.DrawOffsetPx(); x != 3 || y != 4 {
		t.Fatalf("expected offset 3,4 got %d,%d", x, y)
	}

	nested := ctx.PushOffsetPx(-1, 5)
	if x, y := ctx.DrawOffsetPx(); x != 2 || y != 9 {
		t.Fatalf("expected nested offset 2,9 got %d,%d", x, y)
	}
	nested()
	if x, y := ctx.DrawOffsetPx(); x != 3 || y != 4 {
		t.Fatalf("expected restored offset 3,4 got %d,%d", x, y)
	}

	restore()
	if x, y := ctx.DrawOffsetPx(); x != 0 || y != 0 {
		t.Fatalf("expected restored offset 0,0 got %d,%d", x, y)
	}

	var nilCtx *Context
	if x, y := nilCtx.DrawOffsetPx(); x != 0 || y != 0 {
		t.Fatalf("expected nil context offset 0,0 got %d,%d", x, y)
	}
}

func TestDrawOffsetDpUsesScale(t *testing.T) {
	ctx := &Context{Scale: 2.0}
	restore := ctx.PushOffsetPx(4, 6)
	if x, y := ctx.DrawOffsetDp(); x != 2 || y != 3 {
		t.Fatalf("expected dp offset 2,3 got %.2f,%.2f", x, y)
	}
	restore()

	ctx.Scale = 0 // fallback to scale=1
	restore2 := ctx.PushOffsetPx(2, 4)
	if x, y := ctx.DrawOffsetDp(); x != 2 || y != 4 {
		t.Fatalf("expected dp offset 2,4 when scale<=0 got %.2f,%.2f", x, y)
	}
	restore2()
}
