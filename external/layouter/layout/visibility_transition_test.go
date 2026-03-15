package layout

import (
	"math"
	"testing"
	"time"
)

func TestParseVisibilityTransition(t *testing.T) {
	trans, ok := ParseVisibilityTransition("size 1s 50%")
	if !ok {
		t.Fatalf("expected transition parse success")
	}
	if trans.Mode != VisibilityTransitionSize {
		t.Fatalf("expected size mode, got %v", trans.Mode)
	}
	if trans.Duration != time.Second {
		t.Fatalf("expected 1s duration, got %v", trans.Duration)
	}
	if math.Abs(trans.Scale-0.5) > 1e-6 {
		t.Fatalf("expected scale 0.5, got %.4f", trans.Scale)
	}

	inTrans, ok := ParseVisibilityTransition("size-in 750ms 30%")
	if !ok {
		t.Fatalf("expected size-in parse success")
	}
	if inTrans.Mode != VisibilityTransitionSizeIn {
		t.Fatalf("expected size-in mode, got %v", inTrans.Mode)
	}
	if inTrans.Duration != 750*time.Millisecond {
		t.Fatalf("expected 750ms duration, got %v", inTrans.Duration)
	}
	if math.Abs(inTrans.Scale-0.3) > 1e-6 {
		t.Fatalf("expected scale 0.3, got %.4f", inTrans.Scale)
	}

	outTrans, ok := ParseVisibilityTransition("size-out 2s 70%")
	if !ok {
		t.Fatalf("expected size-out parse success")
	}
	if outTrans.Mode != VisibilityTransitionSizeOut {
		t.Fatalf("expected size-out mode, got %v", outTrans.Mode)
	}
	if outTrans.Duration != 2*time.Second {
		t.Fatalf("expected 2s duration, got %v", outTrans.Duration)
	}
	if math.Abs(outTrans.Scale-0.7) > 1e-6 {
		t.Fatalf("expected scale 0.7, got %.4f", outTrans.Scale)
	}

	none, ok := ParseVisibilityTransition("none")
	if !ok {
		t.Fatalf("expected none to parse")
	}
	if none.Enabled() {
		t.Fatalf("none transition should be disabled")
	}
}

func TestVisibilityTransitionHideShow(t *testing.T) {
	var base Base
	cfg := VisibilityTransition{
		Mode:     VisibilityTransitionSize,
		Duration: time.Second,
		Scale:    0.5,
	}
	base.SetVisibilityTransition(cfg)

	start := time.Now()
	base.SetVisibility(VisibilityHidden)
	base.visAnim.start = start

	if !base.ShouldRender() {
		t.Fatalf("expected component to render during hide transition")
	}
	scale := base.visibilityScale(start)
	if math.Abs(scale-1) > 1e-6 {
		t.Fatalf("expected initial hide scale 1, got %.3f", scale)
	}

	mid := start.Add(500 * time.Millisecond)
	scale = base.visibilityScale(mid)
	if math.Abs(scale-0.75) > 1e-3 {
		t.Fatalf("expected mid hide scale 0.75, got %.3f", scale)
	}
	if !base.ShouldRender() {
		t.Fatalf("expected to continue rendering mid hide")
	}

	end := start.Add(1200 * time.Millisecond)
	scale = base.visibilityScale(end)
	if math.Abs(scale-0.5) > 1e-6 {
		t.Fatalf("expected final hide scale 0.5, got %.3f", scale)
	}
	if base.ShouldRender() {
		t.Fatalf("expected rendering to stop after hide completes")
	}

	// Show transition
	showStart := time.Now()
	base.SetVisibility(VisibilityVisible)
	base.visAnim.start = showStart

	if !base.ShouldRender() {
		t.Fatalf("expected rendering during show transition")
	}
	scale = base.visibilityScale(showStart)
	if math.Abs(scale-0.5) > 1e-6 {
		t.Fatalf("expected show to start at scale 0.5, got %.3f", scale)
	}

	showMid := showStart.Add(500 * time.Millisecond)
	scale = base.visibilityScale(showMid)
	if math.Abs(scale-0.75) > 1e-3 {
		t.Fatalf("expected mid show scale 0.75, got %.3f", scale)
	}

	showEnd := showStart.Add(1200 * time.Millisecond)
	scale = base.visibilityScale(showEnd)
	if math.Abs(scale-1) > 1e-6 {
		t.Fatalf("expected final show scale 1, got %.3f", scale)
	}
	if !base.ShouldRender() {
		t.Fatalf("expected rendering to continue after show completes")
	}
}

func TestVisibilityTransitionDirections(t *testing.T) {
	var hideOnly Base
	hideOnly.SetVisibilityTransition(VisibilityTransition{
		Mode:     VisibilityTransitionSizeOut,
		Duration: time.Second,
		Scale:    0.5,
	})

	hideOnly.SetVisibility(VisibilityHidden)
	if !hideOnly.visAnim.active {
		t.Fatalf("expected hide transition to activate for size-out")
	}
	if hideOnly.visAnim.direction != -1 {
		t.Fatalf("expected hide direction -1, got %d", hideOnly.visAnim.direction)
	}
	hideOnly.visAnim.reset()

	hideOnly.SetVisibility(VisibilityVisible)
	if hideOnly.visAnim.active {
		t.Fatalf("expected show to skip animation for size-out")
	}

	var showOnly Base
	showOnly.SetVisibilityTransition(VisibilityTransition{
		Mode:     VisibilityTransitionSizeIn,
		Duration: time.Second,
		Scale:    0.6,
	})

	showOnly.SetVisibility(VisibilityHidden)
	if showOnly.visAnim.active {
		t.Fatalf("expected hide to skip animation for size-in")
	}

	showOnly.SetVisibility(VisibilityVisible)
	if !showOnly.visAnim.active {
		t.Fatalf("expected show transition to activate for size-in")
	}
	if showOnly.visAnim.direction != 1 {
		t.Fatalf("expected show direction 1, got %d", showOnly.visAnim.direction)
	}
}
