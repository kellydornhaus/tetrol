package layout

// TextEngineFuncs is a helper to implement TextEngine by plugging functions.
// Useful for wiring to etxt without creating a full type.
type TextEngineFuncs struct {
	MeasureFunc func(text string, style TextStyle, maxWidthPx int) (int, int)
	DrawFunc    func(dst Surface, text string, rectPx PxRect, style TextStyle)
}

func (t TextEngineFuncs) Measure(text string, style TextStyle, maxWidthPx int) (int, int) {
	if t.MeasureFunc == nil {
		return 0, 0
	}
	return t.MeasureFunc(text, style, maxWidthPx)
}

func (t TextEngineFuncs) Draw(dst Surface, text string, rectPx PxRect, style TextStyle) {
	if t.DrawFunc == nil {
		return
	}
	t.DrawFunc(dst, text, rectPx, style)
}
