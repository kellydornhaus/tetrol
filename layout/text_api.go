package layout

// TextAlign represents horizontal and vertical alignment for text.
type TextAlign int

const (
	AlignStart TextAlign = iota
	AlignCenter
	AlignEnd
)

// TextStyle carries minimal styling for measurement/draw; adapters can add more via FontKey.
type TextStyle struct {
	FontKey string  // arbitrary key resolved by adapter (e.g., font family/face)
	SizeDp  float64 // logical size in dp; adapter converts to px
	Color   Color
	AlignH  TextAlign
	AlignV  TextAlign
	// Wrap: if true, wrap within bounds. If false, single line (clip/ellipsis per adapter choice).
	Wrap bool
	// BaselineOffset moves the text baseline by the given dp amount before drawing (positive = up).
	BaselineOffset float64
}

// TextEngine abstracts text measurement and draw, typically implemented using etxt.
// All measure/draw operate in pixel space for precision; the layout engine converts dp<->px.
type TextEngine interface {
	// Measure returns text extent in pixels for the given options constrained by maxWidthPx (0 = unbounded).
	Measure(text string, style TextStyle, maxWidthPx int) (pxW, pxH int)
	// Draw renders the text within rectPx using the style.
	Draw(dst Surface, text string, rectPx PxRect, style TextStyle)
}
