package etxtadapter

import (
	"testing"

	"github.com/kellydornhaus/layouter/layout"
)

func TestMeasureIgnoresMaxWidthWhenWrapFalse(t *testing.T) {
	engine := New(nil)
	style := layout.TextStyle{SizeDp: 16, Wrap: false}
	text := "This is a fairly long line of text to measure."
	w1, h1 := engine.Measure(text, style, 0)
	w2, h2 := engine.Measure(text, style, 1)
	if w1 != w2 || h1 != h2 {
		t.Fatalf("expected equal measurements, got %dx%d vs %dx%d", w1, h1, w2, h2)
	}
}
