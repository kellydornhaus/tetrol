package layout

import (
	"log"
	"math"
	"strings"
)

// EdgeInsets represents padding in dp units.
type EdgeInsets struct {
	Top, Right, Bottom, Left float64
}

// CornerRadii represents per-corner radii in dp units.
type CornerRadii struct {
	TopLeft     float64
	TopRight    float64
	BottomRight float64
	BottomLeft  float64
}

// CornerRadius creates uniform corner radii (dp units).
func CornerRadius(all float64) CornerRadii {
	if all <= 0 {
		return CornerRadii{}
	}
	return CornerRadii{
		TopLeft:     all,
		TopRight:    all,
		BottomRight: all,
		BottomLeft:  all,
	}
}

// sanitized clamps negative values to zero.
func (r CornerRadii) sanitized() CornerRadii {
	r.TopLeft = maxFloat(0, r.TopLeft)
	r.TopRight = maxFloat(0, r.TopRight)
	r.BottomRight = maxFloat(0, r.BottomRight)
	r.BottomLeft = maxFloat(0, r.BottomLeft)
	return r
}

// HasRadius reports whether any radius is positive.
func (r CornerRadii) HasRadius() bool {
	return r.TopLeft > 0 || r.TopRight > 0 || r.BottomRight > 0 || r.BottomLeft > 0
}

func radiiNearlyEqual(a, b CornerRadii) bool {
	return nearlyEqual(a.TopLeft, b.TopLeft) &&
		nearlyEqual(a.TopRight, b.TopRight) &&
		nearlyEqual(a.BottomRight, b.BottomRight) &&
		nearlyEqual(a.BottomLeft, b.BottomLeft)
}

// ToPx converts the radii into pixel units (scale <=0 treated as 1).
func (r CornerRadii) ToPx(scale float64) PxCornerRadii {
	if scale <= 0 {
		scale = 1
	}
	s := r.sanitized()
	return PxCornerRadii{
		TopLeft:     s.TopLeft * scale,
		TopRight:    s.TopRight * scale,
		BottomRight: s.BottomRight * scale,
		BottomLeft:  s.BottomLeft * scale,
	}
}

// Insets creates uniform padding.
func Insets(all float64) EdgeInsets {
	return EdgeInsets{Top: all, Right: all, Bottom: all, Left: all}
}

// ImageFit defines how a bitmap is mapped into a rectangle.
type ImageFit int

const (
	ImageFitStretch ImageFit = iota
	ImageFitContain
	ImageFitCenter
)

// ImagePaint describes an optional background image for a panel.
type ImagePaint struct {
	Image  Image
	Fit    ImageFit
	AlignH TextAlign
	AlignV TextAlign
	Slice  NineSlice
}

// NineSlice describes corner insets (pixels) for nine-slice scaling.
type NineSlice struct {
	Left   int
	Right  int
	Top    int
	Bottom int
}

// Empty reports whether the slice has no padding.
func (n NineSlice) Empty() bool {
	return n.Left == 0 && n.Right == 0 && n.Top == 0 && n.Bottom == 0
}

const (
	borderEdgeTop = iota
	borderEdgeRight
	borderEdgeBottom
	borderEdgeLeft
)

// Panel augments Base with decoration, text, and measurement helpers.
type Panel struct {
	Base
	Padding               EdgeInsets
	PaddingTopOverride    bool
	PaddingRightOverride  bool
	PaddingBottomOverride bool
	PaddingLeftOverride   bool

	hasBackground bool
	bgColor       Color
	bgImage       *ImagePaint

	hasTint   bool
	tintColor Color

	hasBorder      bool
	borderColor    Color
	borderWidth    float64
	borderTop      float64
	borderRight    float64
	borderBottom   float64
	borderLeft     float64
	borderColors   [4]Color
	borderColorSet [4]bool

	textTemplate string
	text         string
	textStyle    TextStyle
	textMaxWidth float64

	hasDialogVars bool

	textAutoFit bool
	textAutoMin float64
	textAutoMax float64

	autoTextSize float64

	minWidth   Length
	minHeight  Length
	maxWidth   Length
	maxHeight  Length
	fillWidth  bool
	fillHeight bool

	measurementOnly bool

	cachedContent Rect

	aspectRatio float64

	cornerRadii CornerRadii
}

const debugAutoFit = false
const debugAutoFitTarget = ""

// NewPanel constructs a new panel with caching enabled.
func NewPanel() *Panel {
	return &Panel{
		Base: Base{dirty: true},
	}
}

func (p *Panel) dialogBase() *Base {
	if p == nil {
		return nil
	}
	return &p.Base
}

// PanelRef exposes the panel pointer for helper interfaces.
func (p *Panel) PanelRef() *Panel { return p }

// SetMeasurementOnly marks the panel as sizing-only so it won't render or report dirtiness.
func (p *Panel) SetMeasurementOnly(enabled bool) {
	if p == nil || p.measurementOnly == enabled {
		return
	}
	p.measurementOnly = enabled
	p.Base.releaseCache()
	if enabled {
		p.dirty = false
		return
	}
	p.SetDirty()
}

// MeasurementOnly reports whether the panel is sizing-only.
func (p *Panel) MeasurementOnly() bool {
	return p != nil && p.measurementOnly
}

// ShouldRender reports whether the panel should draw this frame.
func (p *Panel) ShouldRender() bool {
	if p == nil {
		return false
	}
	if p.measurementOnly {
		return false
	}
	return p.Base.ShouldRender()
}

// Dirty reports whether the panel needs re-render (measurement-only panels always report clean).
func (p *Panel) Dirty() bool {
	if p == nil || p.measurementOnly {
		return false
	}
	return p.Base.Dirty()
}

func normalizedLength(length Length) Length {
	if !length.Defined() {
		return Length{}
	}
	return length
}

// TextStyle returns the current text style.
func (p *Panel) TextStyle() TextStyle { return p.textStyle }

func (p *Panel) ensureDialogTextCurrent() {
	if p == nil || !p.hasDialogVars {
		return
	}
	resolved := resolveDialogTemplate(p, p.textTemplate)
	if resolved == p.text {
		return
	}
	p.text = resolved
	p.setAutoTextSize(0, false)
}

// Text returns the current text content.
func (p *Panel) Text() string {
	p.ensureDialogTextCurrent()
	return p.text
}

// SetTextAutoFit enables or disables automatic font sizing to fit available space.
func (p *Panel) SetTextAutoFit(enable bool) {
	if p.textAutoFit == enable {
		return
	}
	p.textAutoFit = enable
	p.setAutoTextSize(0, true)
}

// TextAutoFit reports whether automatic font sizing is enabled.
func (p *Panel) TextAutoFit() bool { return p.textAutoFit }

// SetTextAutoFitMin sets the minimum font size (dp) when auto sizing is enabled.
func (p *Panel) SetTextAutoFitMin(min float64) {
	if nearlyEqual(p.textAutoMin, min) {
		return
	}
	if min < 0 {
		min = 0
	}
	p.textAutoMin = min
	if p.textAutoFit {
		p.setAutoTextSize(0, true)
	}
}

// TextAutoFitMin returns the configured minimum font size for auto sizing (dp).
func (p *Panel) TextAutoFitMin() float64 { return p.textAutoMin }

// SetTextAutoFitMax sets the maximum font size (dp) when auto sizing is enabled (0 = no cap).
func (p *Panel) SetTextAutoFitMax(max float64) {
	if nearlyEqual(p.textAutoMax, max) {
		return
	}
	if max < 0 {
		max = 0
	}
	p.textAutoMax = max
	if p.textAutoFit {
		p.setAutoTextSize(0, true)
	}
}

// TextAutoFitMax returns the maximum font size cap for auto sizing (dp). Zero means unbounded.
func (p *Panel) TextAutoFitMax() float64 { return p.textAutoMax }

// AutoTextSize returns the resolved auto-sized font size (dp). Zero if auto sizing is disabled or unresolved.
func (p *Panel) AutoTextSize() float64 {
	if !p.textAutoFit {
		return 0
	}
	return p.autoTextSize
}

// SetFillWidth hints that the panel prefers to occupy available width.
func (p *Panel) SetFillWidth(fill bool) {
	if p.fillWidth == fill {
		return
	}
	p.fillWidth = fill
	p.SetDirty()
}

// FillWidth reports whether the panel wants to fill available width.
func (p *Panel) FillWidth() bool { return p.fillWidth }

// SetFillHeight hints that the panel prefers to occupy available height.
func (p *Panel) SetFillHeight(fill bool) {
	if p.fillHeight == fill {
		return
	}
	p.fillHeight = fill
	p.SetDirty()
}

// FillHeight reports whether the panel wants to fill available height.
func (p *Panel) FillHeight() bool { return p.fillHeight }

// SetAspectRatio enforces a width:height ratio when resolving size. Zero disables it.
func (p *Panel) SetAspectRatio(ratio float64) {
	if ratio <= 0 {
		ratio = 0
	}
	if nearlyEqual(p.aspectRatio, ratio) {
		return
	}
	p.aspectRatio = ratio
	p.SetDirty()
}

// AspectRatio returns the configured width:height ratio (0 when disabled).
func (p *Panel) AspectRatio() float64 { return p.aspectRatio }

// SetFlexWeight sets the flex weight used by parent stacks.
func (p *Panel) SetFlexWeight(weight float64) { p.Base.SetFlexWeight(weight) }

// FlexWeight returns the stored flex weight.
func (p *Panel) FlexWeight() float64 { return p.Base.FlexWeight() }

// SetMinWidth enforces a minimum width (including padding).
func (p *Panel) SetMinWidth(width float64) {
	p.SetMinWidthLength(LengthDP(width))
}

// SetMinHeight enforces a minimum height (including padding).
func (p *Panel) SetMinHeight(height float64) {
	p.SetMinHeightLength(LengthDP(height))
}

// SetMinWidthPercent enforces a minimum width as a fraction of available max width.
func (p *Panel) SetMinWidthPercent(percent float64) {
	p.SetMinWidthLength(LengthPercent(percent))
}

// SetMinHeightPercent enforces a minimum height as a fraction of available max height.
func (p *Panel) SetMinHeightPercent(percent float64) {
	p.SetMinHeightLength(LengthPercent(percent))
}

// SetMaxWidth limits the panel width (including padding).
func (p *Panel) SetMaxWidth(width float64) {
	p.SetMaxWidthLength(LengthDP(width))
}

// SetMaxHeight limits the panel height (including padding).
func (p *Panel) SetMaxHeight(height float64) {
	p.SetMaxHeightLength(LengthDP(height))
}

// SetMaxWidthPercent limits width as a fraction of available max width.
func (p *Panel) SetMaxWidthPercent(percent float64) {
	p.SetMaxWidthLength(LengthPercent(percent))
}

// SetMaxHeightPercent limits height as a fraction of available max height.
func (p *Panel) SetMaxHeightPercent(percent float64) {
	p.SetMaxHeightLength(LengthPercent(percent))
}

// SetWidth fixes the panel width (including padding). width <= 0 clears the constraint.
func (p *Panel) SetWidth(width float64) {
	if width <= 0 {
		p.SetWidthLength(Length{})
		return
	}
	p.SetWidthLength(LengthDP(width))
}

// SetWidthPercent fixes the panel width as a fraction of available width.
func (p *Panel) SetWidthPercent(percent float64) {
	if percent <= 0 {
		p.SetWidthLength(Length{})
		return
	}
	p.SetWidthLength(LengthPercent(percent))
}

// SetWidthLength assigns a fixed width using a Length (dp/percent/viewport). Undefined clears.
func (p *Panel) SetWidthLength(length Length) {
	length = normalizedLength(length)
	if !length.Defined() {
		cleared := false
		if p.minWidth.Defined() {
			p.minWidth = Length{}
			cleared = true
		}
		if p.maxWidth.Defined() {
			p.maxWidth = Length{}
			cleared = true
		}
		if cleared {
			p.SetDirty()
		}
		return
	}
	changed := false
	if p.minWidth != length {
		p.minWidth = length
		changed = true
	}
	if p.maxWidth != length {
		p.maxWidth = length
		changed = true
	}
	if changed {
		p.SetDirty()
	}
}

// SetHeight fixes the panel height (including padding). height <= 0 clears the constraint.
func (p *Panel) SetHeight(height float64) {
	if height <= 0 {
		p.SetHeightLength(Length{})
		return
	}
	p.SetHeightLength(LengthDP(height))
}

// SetHeightPercent fixes height as a fraction of available height.
func (p *Panel) SetHeightPercent(percent float64) {
	if percent <= 0 {
		p.SetHeightLength(Length{})
		return
	}
	p.SetHeightLength(LengthPercent(percent))
}

// SetHeightLength assigns a fixed height using a Length. Undefined clears.
func (p *Panel) SetHeightLength(length Length) {
	length = normalizedLength(length)
	if !length.Defined() {
		cleared := false
		if p.minHeight.Defined() {
			p.minHeight = Length{}
			cleared = true
		}
		if p.maxHeight.Defined() {
			p.maxHeight = Length{}
			cleared = true
		}
		if cleared {
			p.SetDirty()
		}
		return
	}
	changed := false
	if p.minHeight != length {
		p.minHeight = length
		changed = true
	}
	if p.maxHeight != length {
		p.maxHeight = length
		changed = true
	}
	if changed {
		p.SetDirty()
	}
}

// SetPadding updates the panel padding.
func (p *Panel) SetPadding(padding EdgeInsets) {
	if p.Padding == padding {
		return
	}
	p.Padding = padding
	p.PaddingTopOverride = false
	p.PaddingRightOverride = false
	p.PaddingBottomOverride = false
	p.PaddingLeftOverride = false
	p.SetDirty()
}

// SetPaddingTop sets only the top padding (dp units).
func (p *Panel) SetPaddingTop(value float64) {
	if nearlyEqual(p.Padding.Top, value) && p.PaddingTopOverride {
		return
	}
	p.Padding.Top = value
	p.PaddingTopOverride = true
	p.SetDirty()
}

// SetPaddingRight sets only the right padding.
func (p *Panel) SetPaddingRight(value float64) {
	if nearlyEqual(p.Padding.Right, value) && p.PaddingRightOverride {
		return
	}
	p.Padding.Right = value
	p.PaddingRightOverride = true
	p.SetDirty()
}

// SetPaddingBottom sets only the bottom padding.
func (p *Panel) SetPaddingBottom(value float64) {
	if nearlyEqual(p.Padding.Bottom, value) && p.PaddingBottomOverride {
		return
	}
	p.Padding.Bottom = value
	p.PaddingBottomOverride = true
	p.SetDirty()
}

// SetPaddingLeft sets only the left padding.
func (p *Panel) SetPaddingLeft(value float64) {
	if nearlyEqual(p.Padding.Left, value) && p.PaddingLeftOverride {
		return
	}
	p.Padding.Left = value
	p.PaddingLeftOverride = true
	p.SetDirty()
}

// SetBackgroundColor applies a solid background color.
func (p *Panel) SetBackgroundColor(color Color) {
	if p.hasBackground && p.bgColor == color {
		return
	}
	p.hasBackground = true
	p.bgColor = color
	p.SetDirty()
}

// ClearBackgroundColor removes the background fill.
func (p *Panel) ClearBackgroundColor() {
	if !p.hasBackground {
		return
	}
	p.hasBackground = false
	p.SetDirty()
}

// SetTintColor overlays the panel contents with color. Alpha <=0 clears the tint.
func (p *Panel) SetTintColor(color Color) {
	if color.A == 0 {
		p.ClearTintColor()
		return
	}
	if p.hasTint && p.tintColor == color {
		return
	}
	p.hasTint = true
	p.tintColor = color
	p.SetDirty()
}

// ClearTintColor removes any tint overlay.
func (p *Panel) ClearTintColor() {
	if !p.hasTint {
		return
	}
	p.hasTint = false
	p.tintColor = Color{}
	p.SetDirty()
}

// HasTint reports whether a tint overlay is active (alpha > 0).
func (p *Panel) HasTint() bool {
	return p.hasTint && p.tintColor.A > 0
}

// TintColor returns the configured tint overlay color.
func (p *Panel) TintColor() Color {
	return p.tintColor
}

// SetBackgroundImage configures background imagery (nil clears it).
func (p *Panel) SetBackgroundImage(paint ImagePaint) {
	if paint.Image == nil {
		p.ClearBackgroundImage()
		return
	}
	if paint.AlignH != AlignStart && paint.AlignH != AlignCenter && paint.AlignH != AlignEnd {
		paint.AlignH = AlignCenter
	}
	if paint.AlignV != AlignStart && paint.AlignV != AlignCenter && paint.AlignV != AlignEnd {
		paint.AlignV = AlignCenter
	}
	if p.bgImage != nil && p.bgImage.Image == paint.Image && p.bgImage.Fit == paint.Fit && p.bgImage.AlignH == paint.AlignH && p.bgImage.AlignV == paint.AlignV && p.bgImage.Slice == paint.Slice {
		return
	}
	cpy := paint
	p.bgImage = &cpy
	p.SetDirty()
}

// ClearBackgroundImage removes any background image.
func (p *Panel) ClearBackgroundImage() {
	if p.bgImage == nil {
		return
	}
	p.bgImage = nil
	p.SetDirty()
}

// SetCornerRadius applies a uniform corner radius (dp units). Non-positive clears radii.
func (p *Panel) SetCornerRadius(radius float64) {
	p.SetCornerRadii(CornerRadius(radius))
}

// SetCornerRadii configures per-corner radii (dp units). Negative values clamp to zero.
func (p *Panel) SetCornerRadii(radii CornerRadii) {
	r := radii.sanitized()
	if radiiNearlyEqual(p.cornerRadii, r) {
		return
	}
	p.cornerRadii = r
	p.SetDirty()
}

// ClearCornerRadius removes all corner radii.
func (p *Panel) ClearCornerRadius() {
	if !p.cornerRadii.HasRadius() {
		return
	}
	p.cornerRadii = CornerRadii{}
	p.SetDirty()
}

// CornerRadii returns the currently configured corner radii (dp units).
func (p *Panel) CornerRadii() CornerRadii {
	return p.cornerRadii
}

// SetBorder configures a rectangular border drawn inside the panel bounds.
// width is specified in dp units; values <=0 disable the border.
func (p *Panel) SetBorder(color Color, width float64) {
	if width <= 0 || color.A == 0 {
		if p.hasBorder {
			p.clearBorderInternal()
		}
		return
	}
	if p.hasBorder && p.borderColor == color && nearlyEqual(p.borderWidth, width) && nearlyEqual(p.borderTop, width) && nearlyEqual(p.borderRight, width) && nearlyEqual(p.borderBottom, width) && nearlyEqual(p.borderLeft, width) {
		return
	}
	p.hasBorder = true
	p.borderColor = color
	p.borderWidth = width
	p.borderTop = width
	p.borderRight = width
	p.borderBottom = width
	p.borderLeft = width
	for i := range p.borderColorSet {
		p.borderColorSet[i] = false
		p.borderColors[i] = Color{}
	}
	p.updateBorderState()
	p.SetDirty()
}

// ClearBorder removes any configured border.
func (p *Panel) ClearBorder() {
	if !p.hasBorder {
		return
	}
	p.clearBorderInternal()
}

// SetBorderTop applies a border to the top edge only.
func (p *Panel) SetBorderTop(width float64, color Color) {
	p.setEdgeWidth(borderEdgeTop, width)
	p.setEdgeColor(borderEdgeTop, color, true)
	p.updateBorderState()
	p.SetDirty()
}

// SetBorderTopWidth sets the top border width, leaving color unchanged.
func (p *Panel) SetBorderTopWidth(width float64) {
	p.setEdgeWidth(borderEdgeTop, width)
	p.updateBorderState()
	p.SetDirty()
}

// SetBorderTopColor sets the top border color override.
func (p *Panel) SetBorderTopColor(color Color) {
	p.setEdgeColor(borderEdgeTop, color, true)
	p.updateBorderState()
	p.SetDirty()
}

// ClearBorderTop removes the top border.
func (p *Panel) ClearBorderTop() {
	p.setEdgeWidth(borderEdgeTop, 0)
	p.setEdgeColor(borderEdgeTop, Color{}, false)
	p.updateBorderState()
	p.SetDirty()
}

// SetBorderRight applies a border to the right edge only.
func (p *Panel) SetBorderRight(width float64, color Color) {
	p.setEdgeWidth(borderEdgeRight, width)
	p.setEdgeColor(borderEdgeRight, color, true)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) SetBorderRightWidth(width float64) {
	p.setEdgeWidth(borderEdgeRight, width)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) SetBorderRightColor(color Color) {
	p.setEdgeColor(borderEdgeRight, color, true)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) ClearBorderRight() {
	p.setEdgeWidth(borderEdgeRight, 0)
	p.setEdgeColor(borderEdgeRight, Color{}, false)
	p.updateBorderState()
	p.SetDirty()
}

// SetBorderBottom applies a border to the bottom edge only.
func (p *Panel) SetBorderBottom(width float64, color Color) {
	p.setEdgeWidth(borderEdgeBottom, width)
	p.setEdgeColor(borderEdgeBottom, color, true)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) SetBorderBottomWidth(width float64) {
	p.setEdgeWidth(borderEdgeBottom, width)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) SetBorderBottomColor(color Color) {
	p.setEdgeColor(borderEdgeBottom, color, true)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) ClearBorderBottom() {
	p.setEdgeWidth(borderEdgeBottom, 0)
	p.setEdgeColor(borderEdgeBottom, Color{}, false)
	p.updateBorderState()
	p.SetDirty()
}

// SetBorderLeft applies a border to the left edge only.
func (p *Panel) SetBorderLeft(width float64, color Color) {
	p.setEdgeWidth(borderEdgeLeft, width)
	p.setEdgeColor(borderEdgeLeft, color, true)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) SetBorderLeftWidth(width float64) {
	p.setEdgeWidth(borderEdgeLeft, width)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) SetBorderLeftColor(color Color) {
	p.setEdgeColor(borderEdgeLeft, color, true)
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) ClearBorderLeft() {
	p.setEdgeWidth(borderEdgeLeft, 0)
	p.setEdgeColor(borderEdgeLeft, Color{}, false)
	p.updateBorderState()
	p.SetDirty()
}

// BorderTopWidth returns the current top border width in dp.
func (p *Panel) BorderTopWidth() float64 { return p.borderTop }

// BorderRightWidth returns the current right border width in dp.
func (p *Panel) BorderRightWidth() float64 { return p.borderRight }

// BorderBottomWidth returns the current bottom border width in dp.
func (p *Panel) BorderBottomWidth() float64 { return p.borderBottom }

// BorderLeftWidth returns the current left border width in dp.
func (p *Panel) BorderLeftWidth() float64 { return p.borderLeft }

// BorderTopColor reports the effective top border color.
func (p *Panel) BorderTopColor() Color { return p.edgeColor(borderEdgeTop) }

func (p *Panel) BorderRightColor() Color { return p.edgeColor(borderEdgeRight) }

func (p *Panel) BorderBottomColor() Color { return p.edgeColor(borderEdgeBottom) }

func (p *Panel) BorderLeftColor() Color { return p.edgeColor(borderEdgeLeft) }

func (p *Panel) clearBorderInternal() {
	p.hasBorder = false
	p.borderWidth = 0
	p.borderTop, p.borderRight, p.borderBottom, p.borderLeft = 0, 0, 0, 0
	for i := range p.borderColorSet {
		p.borderColorSet[i] = false
		p.borderColors[i] = Color{}
	}
	p.updateBorderState()
	p.SetDirty()
}

func (p *Panel) setEdgeWidth(edge int, width float64) {
	if width < 0 {
		width = 0
	}
	switch edge {
	case borderEdgeTop:
		p.borderTop = width
	case borderEdgeRight:
		p.borderRight = width
	case borderEdgeBottom:
		p.borderBottom = width
	case borderEdgeLeft:
		p.borderLeft = width
	}
}

func (p *Panel) edgeColor(edge int) Color {
	if edge >= 0 && edge < len(p.borderColorSet) && p.borderColorSet[edge] {
		return p.borderColors[edge]
	}
	return p.borderColor
}

func (p *Panel) setEdgeColor(edge int, color Color, defined bool) {
	if edge < 0 || edge >= len(p.borderColorSet) {
		return
	}
	if defined {
		p.borderColorSet[edge] = true
		p.borderColors[edge] = color
	} else {
		p.borderColorSet[edge] = false
		p.borderColors[edge] = Color{}
	}
}

func (p *Panel) updateBorderState() {
	widths := []float64{p.borderTop, p.borderRight, p.borderBottom, p.borderLeft}
	maxWidth := 0.0
	any := false
	for i, w := range widths {
		if w <= 0 {
			continue
		}
		if w > maxWidth {
			maxWidth = w
		}
		col := p.edgeColor(i)
		if col.A > 0 {
			any = true
		}
	}
	p.borderWidth = maxWidth
	p.hasBorder = any
}

// SetText updates the panel text content.
func (p *Panel) SetText(text string) {
	if p == nil {
		return
	}
	hasVars := hasDialogPlaceholder(text)
	if !hasVars && p.textTemplate == text && p.text == text {
		return
	}
	p.textTemplate = text
	p.hasDialogVars = hasVars
	if hasVars {
		p.text = resolveDialogTemplate(p, text)
	} else {
		p.text = text
	}
	p.setAutoTextSize(0, false)
	p.SetDirty()
}

// SetDialogVariable stores a dialog variable on this panel.
func (p *Panel) SetDialogVariable(name, value string) {
	if p == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if setDialogVariableInBase(p.dialogBase(), name, value) {
		p.Base.SetDirty()
	}
}

// ClearDialogVariable removes a dialog variable override from this panel.
func (p *Panel) ClearDialogVariable(name string) {
	if p == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if clearDialogVariableInBase(p.dialogBase(), name) {
		p.Base.SetDirty()
	}
}

// DialogVariable resolves a dialog variable value using this panel as the lookup root.
func (p *Panel) DialogVariable(name string) (string, bool) {
	if p == nil {
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return resolveDialogVariableForHost(p, name)
}

// SetTextStyle updates the panel text style.
func (p *Panel) SetTextStyle(style TextStyle) {
	if p.textStyle == style {
		return
	}
	p.textStyle = style
	p.setAutoTextSize(0, false)
	p.SetDirty()
}

// SetTextMaxWidth constrains text layout width (dp). Zero disables.
func (p *Panel) SetTextMaxWidth(width float64) {
	if nearlyEqual(p.textMaxWidth, width) {
		return
	}
	p.textMaxWidth = width
	if p.textAutoFit {
		p.setAutoTextSize(0, false)
	}
	p.SetDirty()
}

// contentConstraints adjusts constraints for padding and min size.
func (p *Panel) contentConstraints(ctx *Context, cs Constraints) Constraints {
	inner := cs
	padW := p.Padding.Left + p.Padding.Right
	padH := p.Padding.Top + p.Padding.Bottom

	if inner.Min.W > 0 {
		inner.Min.W = maxFloat(0, inner.Min.W-padW)
	}
	if inner.Min.H > 0 {
		inner.Min.H = maxFloat(0, inner.Min.H-padH)
	}

	if minW := p.minWidth.ResolveWidth(ctx, cs.Max.W); minW > 0 {
		minInner := maxFloat(0, minW-padW)
		if minInner > inner.Min.W {
			inner.Min.W = minInner
		}
	}
	if minH := p.minHeight.ResolveHeight(ctx, cs.Max.H); minH > 0 {
		minInner := maxFloat(0, minH-padH)
		if minInner > inner.Min.H {
			inner.Min.H = minInner
		}
	}

	if inner.Max.W > 0 {
		inner.Max.W = maxFloat(0, inner.Max.W-padW)
	}
	if inner.Max.W <= 0 && cs.Max.W > 0 {
		inner.Max.W = maxFloat(0, cs.Max.W-padW)
	}
	if maxW := p.maxWidth.ResolveWidth(ctx, cs.Max.W); maxW > 0 {
		limit := maxFloat(0, maxW-padW)
		if inner.Max.W <= 0 || limit < inner.Max.W {
			inner.Max.W = limit
		}
	}
	if inner.Max.H > 0 {
		inner.Max.H = maxFloat(0, inner.Max.H-padH)
	}
	if inner.Max.H <= 0 && cs.Max.H > 0 {
		inner.Max.H = maxFloat(0, cs.Max.H-padH)
	}
	if maxH := p.maxHeight.ResolveHeight(ctx, cs.Max.H); maxH > 0 {
		limit := maxFloat(0, maxH-padH)
		if inner.Max.H <= 0 || limit < inner.Max.H {
			inner.Max.H = limit
		}
	}
	return inner
}

// ContentConstraints exposes the inner constraints available to a child/content area after
// accounting for padding and the panel's min/max sizing rules. This is useful for custom
// components that want to size their own content but still honor Panel padding.
func (p *Panel) ContentConstraints(ctx *Context, cs Constraints) Constraints {
	if p == nil {
		return cs
	}
	return p.contentConstraints(ctx, cs)
}

// resolveSize combines child content, padding, and text needs.
func (p *Panel) resolveSize(ctx *Context, cs Constraints, content Size) Size {
	inner := p.contentConstraints(ctx, cs)
	if txt := p.measureText(ctx, inner); txt.W > 0 || txt.H > 0 {
		content.W = maxFloat(content.W, txt.W)
		content.H = maxFloat(content.H, txt.H)
	}
	width := content.W + p.Padding.Left + p.Padding.Right
	height := content.H + p.Padding.Top + p.Padding.Bottom
	if minW := p.minWidth.ResolveWidth(ctx, cs.Max.W); minW > 0 && width < minW {
		width = minW
	}
	if minH := p.minHeight.ResolveHeight(ctx, cs.Max.H); minH > 0 && height < minH {
		height = minH
	}
	if maxW := p.maxWidth.ResolveWidth(ctx, cs.Max.W); maxW > 0 && width > maxW {
		width = maxW
	}
	if maxH := p.maxHeight.ResolveHeight(ctx, cs.Max.H); maxH > 0 && height > maxH {
		height = maxH
	}
	if p.aspectRatio > 0 {
		width, height = p.applyAspectRatio(ctx, cs, width, height)
	}
	return clampSizeToConstraints(Size{W: width, H: height}, cs)
}

// ResolveSize merges the provided content size with padding, min/max limits, aspect ratio,
// and other Panel sizing rules so custom components can reuse the standard measurement path.
func (p *Panel) ResolveSize(ctx *Context, cs Constraints, content Size) Size {
	if p == nil {
		return clampSizeToConstraints(content, cs)
	}
	return p.resolveSize(ctx, cs, content)
}

func (p *Panel) applyAspectRatio(ctx *Context, cs Constraints, width, height float64) (float64, float64) {
	ratio := p.aspectRatio
	if ratio <= 0 {
		return width, height
	}
	resolveMinWidth := func() float64 {
		return p.minWidth.ResolveWidth(ctx, cs.Max.W)
	}
	resolveMinHeight := func() float64 {
		return p.minHeight.ResolveHeight(ctx, cs.Max.H)
	}
	resolveMaxWidth := func() float64 {
		if cs.Max.W > 0 {
			return cs.Max.W
		}
		return p.maxWidth.ResolveWidth(ctx, cs.Max.W)
	}
	resolveMaxHeight := func() float64 {
		if cs.Max.H > 0 {
			return cs.Max.H
		}
		return p.maxHeight.ResolveHeight(ctx, cs.Max.H)
	}

	if width > 0 && height > 0 {
		expected := width / ratio
		if expected > height {
			width = height * ratio
		} else {
			height = expected
		}
		return width, height
	}
	if width > 0 {
		height = width / ratio
		return width, height
	}
	if height > 0 {
		width = height * ratio
		return width, height
	}
	if mw := resolveMinWidth(); mw > 0 {
		width = mw
		height = width / ratio
		return width, height
	}
	if mh := resolveMinHeight(); mh > 0 {
		height = mh
		width = height * ratio
		return width, height
	}
	if maxW := resolveMaxWidth(); maxW > 0 {
		width = maxW
		height = width / ratio
		return width, height
	}
	if maxH := resolveMaxHeight(); maxH > 0 {
		height = maxH
		width = height * ratio
		return width, height
	}
	return width, height
}

// ContentBounds returns the inner rectangle available to children.
func (p *Panel) ContentBounds() Rect {
	b := p.bounds
	innerW := maxFloat(0, b.W-p.Padding.Left-p.Padding.Right)
	innerH := maxFloat(0, b.H-p.Padding.Top-p.Padding.Bottom)
	rect := Rect{
		X: p.Padding.Left,
		Y: p.Padding.Top,
		W: innerW,
		H: innerH,
	}
	p.cachedContent = rect
	return rect
}

func (p *Panel) resolveCornerRadiiPx(scale float64, w, h int) (PxCornerRadii, bool) {
	if scale <= 0 {
		scale = 1
	}
	if !p.cornerRadii.HasRadius() || w <= 0 || h <= 0 {
		return PxCornerRadii{}, false
	}
	px := p.cornerRadii.ToPx(scale)
	px = px.Normalized(float64(w), float64(h))
	if !px.HasRadius() {
		return PxCornerRadii{}, false
	}
	return px, true
}

func (p *Panel) needsRoundedCache(ctx *Context) bool {
	if ctx == nil || ctx.Renderer == nil {
		return false
	}
	if !p.cornerRadii.HasRadius() {
		return false
	}
	if _, ok := ctx.Renderer.(RoundedRenderer); !ok {
		return false
	}
	if p.hasBackground && p.bgColor.A > 0 {
		return true
	}
	if p.hasTint && p.tintColor.A > 0 {
		return true
	}
	if p.hasBorder && p.borderWidth > 0 {
		_, _, ok := p.uniformBorder()
		return ok
	}
	return false
}

func (p *Panel) uniformBorder() (float64, Color, bool) {
	widths := []float64{p.borderTop, p.borderRight, p.borderBottom, p.borderLeft}
	baseWidth := widths[borderEdgeTop]
	if baseWidth <= 0 {
		return 0, Color{}, false
	}
	for _, w := range widths[1:] {
		if !nearlyEqual(baseWidth, w) {
			return 0, Color{}, false
		}
	}
	baseColor := p.edgeColor(borderEdgeTop)
	if baseColor.A == 0 {
		return 0, Color{}, false
	}
	for edge := borderEdgeRight; edge <= borderEdgeLeft; edge++ {
		if col := p.edgeColor(edge); col != baseColor {
			return 0, Color{}, false
		}
	}
	return baseWidth, baseColor, true
}

func (p *Panel) hasTextContent() bool {
	if p == nil {
		return false
	}
	return p.textTemplate != "" || p.text != ""
}

// DrawPanel renders decorations and delegates child drawing.
func (p *Panel) DrawPanel(ctx *Context, dst Surface, render func(target Surface)) {
	p.drawPanel(ctx, dst, nil, render, true)
}

// DrawPanelWithOwner renders panel decorations tagging debug logs with the owning component.
func (p *Panel) DrawPanelWithOwner(ctx *Context, dst Surface, owner Component, render func(target Surface)) {
	p.drawPanel(ctx, dst, owner, render, true)
}

// DrawPanelChildren renders child content and panel decorations, only caching when text
// or visibility transitions require it (cheap visuals draw directly each frame).
func (p *Panel) DrawPanelChildren(ctx *Context, dst Surface, render func(target Surface)) {
	p.drawPanel(ctx, dst, nil, render, false)
}

// DrawPanelChildrenWithOwner renders child content and decorations while tagging logs with the owning component.
func (p *Panel) DrawPanelChildrenWithOwner(ctx *Context, dst Surface, owner Component, render func(target Surface)) {
	p.drawPanel(ctx, dst, owner, render, false)
}

func (p *Panel) drawPanel(ctx *Context, dst Surface, owner Component, render func(target Surface), renderHasOwnContent bool) {
	if !p.ShouldRender() {
		p.Base.releaseCache()
		return
	}
	if ctx == nil || ctx.Renderer == nil || dst == nil {
		return
	}

	px := p.bounds.ToPx(ctx.Scale)
	needsCache := renderHasOwnContent || p.hasTextContent() || p.visTransitionEnabled || p.needsRoundedCache(ctx)
	if needsCache {
		p.Base.DrawCachedWithOwner(ctx, dst, owner, func(target Surface) {
			localRect := PxRect{X: 0, Y: 0, W: px.W, H: px.H}
			p.paintBackground(ctx, target, localRect)
			if render != nil {
				render(target)
			}
			p.paintTint(ctx, target, localRect)
			p.paintText(ctx, target, localRect)
			p.paintBorder(ctx, target, localRect)
		})
		return
	}

	p.Base.releaseCache()
	if px.W <= 0 || px.H <= 0 {
		p.dirty = false
		return
	}

	restore := ctx.pushOffsetPx(px.X, px.Y)
	offsetX, offsetY := ctx.drawOffset()
	panelRect := PxRect{X: offsetX, Y: offsetY, W: px.W, H: px.H}

	p.paintBackground(ctx, dst, panelRect)
	if render != nil {
		render(dst)
	}
	p.paintTint(ctx, dst, panelRect)
	p.paintText(ctx, dst, panelRect)
	p.paintBorder(ctx, dst, panelRect)
	restore()
	p.dirty = false
}

func (p *Panel) paintBackground(ctx *Context, dst Surface, rect PxRect) {
	if ctx == nil || ctx.Renderer == nil {
		return
	}
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	radii, hasRadii := p.resolveCornerRadiiPx(ctx.Scale, rect.W, rect.H)
	var rounded RoundedRenderer
	if hasRadii {
		if rr, ok := ctx.Renderer.(RoundedRenderer); ok {
			rounded = rr
		} else {
			hasRadii = false
		}
	}
	if p.hasBackground && p.bgColor.A > 0 {
		if hasRadii && rounded != nil {
			rounded.FillRoundedRect(dst, rect, radii, p.bgColor)
		} else {
			ctx.Renderer.FillRect(dst, rect, p.bgColor)
		}
	}
	if p.bgImage != nil && p.bgImage.Image != nil {
		if !p.bgImage.Slice.Empty() {
			drawNineSlice(ctx, dst, rect, *p.bgImage)
		} else {
			drawBackgroundImage(ctx, dst, rect, *p.bgImage)
		}
	}
}

func (p *Panel) paintTint(ctx *Context, dst Surface, rect PxRect) {
	if ctx == nil || ctx.Renderer == nil || !p.hasTint || p.tintColor.A == 0 {
		return
	}
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if radii, has := p.resolveCornerRadiiPx(ctx.Scale, rect.W, rect.H); has {
		if rounded, ok := ctx.Renderer.(RoundedRenderer); ok && rounded != nil {
			rounded.TintRoundedRect(dst, rect, radii, p.tintColor)
			return
		}
	}
	ctx.Renderer.TintRect(dst, rect, p.tintColor)
}

func (p *Panel) paintBorder(ctx *Context, dst Surface, rect PxRect) {
	if !p.hasBorder || ctx == nil || ctx.Renderer == nil || p.borderWidth <= 0 {
		return
	}
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if radii, has := p.resolveCornerRadiiPx(ctx.Scale, rect.W, rect.H); has {
		if rounded, ok := ctx.Renderer.(RoundedRenderer); ok && rounded != nil {
			if widthDp, color, ok := p.uniformBorder(); ok {
				scale := ctx.Scale
				if scale <= 0 {
					scale = 1
				}
				strokePx := widthDp * scale
				if strokePx > 0 {
					rounded.StrokeRoundedRect(dst, rect, radii, strokePx, color)
					return
				}
			}
		}
	}
	widths := []float64{p.borderTop, p.borderRight, p.borderBottom, p.borderLeft}
	renderer := ctx.Renderer

	if topPx := dpToPx(widths[borderEdgeTop], ctx.Scale); topPx > 0 {
		if topPx > rect.H {
			topPx = rect.H
		}
		col := p.edgeColor(borderEdgeTop)
		if col.A > 0 {
			renderer.FillRect(dst, PxRect{X: rect.X, Y: rect.Y, W: rect.W, H: topPx}, col)
		}
	}
	if bottomPx := dpToPx(widths[borderEdgeBottom], ctx.Scale); bottomPx > 0 {
		if bottomPx > rect.H {
			bottomPx = rect.H
		}
		col := p.edgeColor(borderEdgeBottom)
		if col.A > 0 {
			y := maxInt(rect.Y+rect.H-bottomPx, rect.Y)
			renderer.FillRect(dst, PxRect{X: rect.X, Y: y, W: rect.W, H: bottomPx}, col)
		}
	}
	if leftPx := dpToPx(widths[borderEdgeLeft], ctx.Scale); leftPx > 0 {
		if leftPx > rect.W {
			leftPx = rect.W
		}
		col := p.edgeColor(borderEdgeLeft)
		if col.A > 0 {
			renderer.FillRect(dst, PxRect{X: rect.X, Y: rect.Y, W: leftPx, H: rect.H}, col)
		}
	}
	if rightPx := dpToPx(widths[borderEdgeRight], ctx.Scale); rightPx > 0 {
		if rightPx > rect.W {
			rightPx = rect.W
		}
		col := p.edgeColor(borderEdgeRight)
		if col.A > 0 {
			x := maxInt(rect.X+rect.W-rightPx, rect.X)
			renderer.FillRect(dst, PxRect{X: x, Y: rect.Y, W: rightPx, H: rect.H}, col)
		}
	}
}

func (p *Panel) paintText(ctx *Context, dst Surface, rect PxRect) {
	p.ensureDialogTextCurrent()
	if ctx == nil || ctx.Text == nil || p.text == "" {
		return
	}
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	rectDp := p.cachedContent
	if rectDp.W <= 0 || rectDp.H <= 0 {
		rectDp = p.ContentBounds()
	}
	if rectDp.W <= 0 || rectDp.H <= 0 {
		return
	}
	available := rectDp
	if p.textMaxWidth > 0 && p.textMaxWidth < available.W {
		switch p.textStyle.AlignH {
		case AlignCenter:
			offset := (available.W - p.textMaxWidth) / 2
			available.X += offset
		case AlignEnd:
			available.X += available.W - p.textMaxWidth
		}
		available.W = p.textMaxWidth
	}
	rectPx := available.ToPx(ctx.Scale)
	rectPx.X += rect.X
	rectPx.Y += rect.Y
	style := p.textStyle
	if p.textAutoFit && p.autoTextSize > 0 {
		style.SizeDp = p.autoTextSize
	}
	if style.SizeDp <= 0 {
		style.SizeDp = defaultAutoTextSizeDp
	}
	ctx.Text.Draw(dst, p.text, rectPx, style)
}

type subImager interface {
	SubImage(rect PxRect) Image
}

func drawBackgroundImage(ctx *Context, dst Surface, rect PxRect, paint ImagePaint) {
	imgW, imgH := paint.Image.SizePx()
	if imgW <= 0 || imgH <= 0 {
		return
	}
	alignH := paint.AlignH
	if alignH != AlignStart && alignH != AlignCenter && alignH != AlignEnd {
		alignH = AlignCenter
	}
	alignV := paint.AlignV
	if alignV != AlignStart && alignV != AlignCenter && alignV != AlignEnd {
		alignV = AlignCenter
	}
	drawRect := computeImageRect(rect, imgW, imgH, paint.Fit, alignH, alignV)
	ctx.Renderer.DrawImage(dst, paint.Image, drawRect)
}

func drawNineSlice(ctx *Context, dst Surface, rect PxRect, paint ImagePaint) {
	si, ok := paint.Image.(subImager)
	if !ok || paint.Slice.Empty() {
		drawBackgroundImage(ctx, dst, rect, paint)
		return
	}
	srcW, srcH := paint.Image.SizePx()
	if srcW <= 0 || srcH <= 0 || rect.W <= 0 || rect.H <= 0 {
		return
	}
	ns := normalizeNineSlice(paint.Slice, srcW, srcH)
	left, right := adjustSliceEdge(ns.Left, ns.Right, rect.W)
	top, bottom := adjustSliceEdge(ns.Top, ns.Bottom, rect.H)

	sx := [4]int{0, ns.Left, srcW - ns.Right, srcW}
	sy := [4]int{0, ns.Top, srcH - ns.Bottom, srcH}
	dx := [4]int{rect.X, rect.X + left, rect.X + rect.W - right, rect.X + rect.W}
	dy := [4]int{rect.Y, rect.Y + top, rect.Y + rect.H - bottom, rect.Y + rect.H}

	for yi := 0; yi < 3; yi++ {
		for xi := 0; xi < 3; xi++ {
			srcRect := PxRect{X: sx[xi], Y: sy[yi], W: sx[xi+1] - sx[xi], H: sy[yi+1] - sy[yi]}
			dstRect := PxRect{X: dx[xi], Y: dy[yi], W: dx[xi+1] - dx[xi], H: dy[yi+1] - dy[yi]}
			if srcRect.W <= 0 || srcRect.H <= 0 || dstRect.W <= 0 || dstRect.H <= 0 {
				continue
			}
			sub := si.SubImage(srcRect)
			if sub == nil {
				continue
			}
			ctx.Renderer.DrawImage(dst, sub, dstRect)
		}
	}
}

func normalizeNineSlice(s NineSlice, srcW, srcH int) NineSlice {
	left := clampInt(s.Left, 0, srcW)
	right := clampInt(s.Right, 0, srcW-left)
	if total := left + right; total > srcW && total > 0 {
		left = int(math.Round(float64(left) * float64(srcW) / float64(total)))
		right = srcW - left
	}
	top := clampInt(s.Top, 0, srcH)
	bottom := clampInt(s.Bottom, 0, srcH-top)
	if total := top + bottom; total > srcH && total > 0 {
		top = int(math.Round(float64(top) * float64(srcH) / float64(total)))
		bottom = srcH - top
	}
	return NineSlice{Left: left, Right: right, Top: top, Bottom: bottom}
}

func adjustSliceEdge(a, b, total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if a+b <= total {
		return a, b
	}
	if a+b == 0 {
		return 0, 0
	}
	scale := float64(total) / float64(a+b)
	left := int(math.Round(float64(a) * scale))
	right := total - left
	return left, right
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (p *Panel) measureText(ctx *Context, cs Constraints) Size {
	p.ensureDialogTextCurrent()
	if ctx == nil || ctx.Text == nil || p.text == "" {
		p.setAutoTextSize(0, false)
		return Size{}
	}
	if p.textAutoFit {
		if size, measured := p.autoFitText(ctx, cs); size > 0 {
			p.setAutoTextSize(size, false)
			if debugAutoFit && (debugAutoFitTarget == "" || strings.Contains(p.text, debugAutoFitTarget)) {
				log.Printf("panel auto-fit text size=%.2f dp for %q (constraints=%+v measured=%+v)", size, p.text, cs, measured)
			}
			return clampSizeToConstraints(measured, cs)
		}
		p.setAutoTextSize(0, false)
	}
	maxW := cs.Max.W
	if p.textMaxWidth > 0 && (maxW <= 0 || p.textMaxWidth < maxW) {
		maxW = p.textMaxWidth
	}
	style := p.textStyle
	if style.SizeDp <= 0 {
		style.SizeDp = defaultAutoTextSizeDp
	}
	maxPx := 0
	if maxW > 0 {
		maxPx = dpToPx(maxW, ctx.Scale)
	}
	pxW, pxH := ctx.Text.Measure(p.text, style, maxPx)
	return clampSizeToConstraints(PxSize{W: pxW, H: pxH}.ToDp(ctx.Scale), cs)
}

const defaultAutoTextSizeDp = 14.0

func (p *Panel) setAutoTextSize(size float64, markDirty bool) {
	if nearlyEqual(p.autoTextSize, size) {
		return
	}
	p.autoTextSize = size
	if markDirty {
		p.SetDirty()
	}
}

func (p *Panel) refreshAutoTextSize(ctx *Context, bounds Rect) {
	p.ensureDialogTextCurrent()
	if !p.textAutoFit || ctx == nil || ctx.Text == nil || p.text == "" {
		return
	}
	if bounds.W <= 0 && bounds.H <= 0 {
		return
	}
	size, _ := p.autoFitText(ctx, Constraints{Max: Size{W: bounds.W, H: bounds.H}})
	if size > 0 {
		p.setAutoTextSize(size, true)
	}
}

func (p *Panel) autoFitText(ctx *Context, cs Constraints) (float64, Size) {
	p.ensureDialogTextCurrent()
	if ctx == nil || ctx.Text == nil || p.text == "" {
		return 0, Size{}
	}
	scale := ctx.Scale
	if scale <= 0 {
		scale = 1
	}

	baseStyle := p.textStyle
	baseSize := baseStyle.SizeDp
	if baseSize <= 0 {
		baseSize = defaultAutoTextSizeDp
	}
	baseStyle.SizeDp = baseSize

	limitW := cs.Max.W
	if p.textMaxWidth > 0 && (limitW <= 0 || p.textMaxWidth < limitW) {
		limitW = p.textMaxWidth
	}
	limitH := cs.Max.H

	if limitW <= 0 && limitH <= 0 {
		pxW, pxH := ctx.Text.Measure(p.text, baseStyle, 0)
		return baseSize, PxSize{W: pxW, H: pxH}.ToDp(scale)
	}

	minSize := p.textAutoMin
	if minSize <= 0 {
		minSize = baseSize * 0.25
		if minSize < 4 {
			minSize = 4
		}
		if debugAutoFit && (debugAutoFitTarget == "" || strings.Contains(p.text, debugAutoFitTarget)) {
			log.Printf("panel auto-fit min defaulted to %.2f dp (base=%.2f)", minSize, baseSize)
		}
	}
	maxSize := p.textAutoMax
	if maxSize > 0 && maxSize < minSize {
		maxSize = minSize
	}
	const maxCap = 512.0

	fits := func(size float64) (Size, bool) {
		if size <= 0 {
			return Size{}, false
		}
		style := baseStyle
		style.SizeDp = size
		maxWidthPx := 0
		if limitW > 0 && style.Wrap {
			maxWidthPx = dpToPx(limitW, scale)
		}
		pxW, pxH := ctx.Text.Measure(p.text, style, maxWidthPx)
		measured := PxSize{W: pxW, H: pxH}.ToDp(scale)
		if limitW > 0 && measured.W > limitW+0.1 {
			return measured, false
		}
		if limitH > 0 && measured.H > limitH+0.1 {
			return measured, false
		}
		return measured, true
	}

	bestSize := minSize
	bestMeasured, ok := fits(bestSize)
	if !ok {
		return bestSize, bestMeasured
	}

	high := maxSize
	var highMeasured Size
	if maxSize > 0 {
		highMeasured, ok = fits(high)
		if ok {
			return high, highMeasured
		}
	} else {
		high = baseSize
		if high < bestSize {
			high = bestSize
		}
	}

	if high <= bestSize {
		high = bestSize * 1.25
	}

	if maxSize <= 0 {
		for i := 0; i < 12; i++ {
			highMeasured, ok = fits(high)
			if !ok {
				break
			}
			bestSize = high
			bestMeasured = highMeasured
			high *= 1.5
			if high > maxCap {
				high = maxCap
				break
			}
		}
		highMeasured, ok = fits(high)
		if ok {
			return high, highMeasured
		}
	} else {
		highMeasured, ok = fits(high)
		if ok {
			return high, highMeasured
		}
	}

	low := bestSize
	lowMeasured := bestMeasured
	if high <= low {
		return low, lowMeasured
	}

	for i := 0; i < 20 && high-low > 0.1; i++ {
		mid := (low + high) / 2
		measured, fit := fits(mid)
		if fit {
			low = mid
			lowMeasured = measured
		} else {
			high = mid
		}
	}
	return low, lowMeasured
}

func computeImageRect(bounds PxRect, imgW, imgH int, fit ImageFit, alignH, alignV TextAlign) PxRect {
	if bounds.W <= 0 || bounds.H <= 0 || imgW <= 0 || imgH <= 0 {
		return PxRect{X: bounds.X, Y: bounds.Y, W: 0, H: 0}
	}
	switch fit {
	case ImageFitContain:
		scale := math.Min(float64(bounds.W)/float64(imgW), float64(bounds.H)/float64(imgH))
		if scale <= 0 {
			return PxRect{X: bounds.X, Y: bounds.Y, W: 0, H: 0}
		}
		w := int(math.Round(float64(imgW) * scale))
		h := int(math.Round(float64(imgH) * scale))
		if w > bounds.W {
			w = bounds.W
		}
		if h > bounds.H {
			h = bounds.H
		}
		x := bounds.X + alignOffsetPx(alignH, bounds.W, w)
		y := bounds.Y + alignOffsetPx(alignV, bounds.H, h)
		return PxRect{X: x, Y: y, W: w, H: h}
	case ImageFitCenter:
		w := minInt(bounds.W, imgW)
		h := minInt(bounds.H, imgH)
		x := bounds.X + alignOffsetPx(alignH, bounds.W, w)
		y := bounds.Y + alignOffsetPx(alignV, bounds.H, h)
		return PxRect{X: x, Y: y, W: w, H: h}
	default:
		return bounds
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func alignOffsetPx(alignment TextAlign, container, content int) int {
	delta := container - content
	if delta <= 0 {
		return 0
	}
	switch alignment {
	case AlignEnd:
		return delta
	case AlignCenter:
		return delta / 2
	default:
		return 0
	}
}

// SetMinWidthLength assigns a minimum width using the unified length type.
func (p *Panel) SetMinWidthLength(length Length) {
	length = normalizedLength(length)
	if p.minWidth == length {
		return
	}
	p.minWidth = length
	p.SetDirty()
}

// SetMinHeightLength assigns a minimum height using the unified length type.
func (p *Panel) SetMinHeightLength(length Length) {
	length = normalizedLength(length)
	if p.minHeight == length {
		return
	}
	p.minHeight = length
	p.SetDirty()
}

// SetMaxWidthLength assigns a maximum width using the unified length type.
func (p *Panel) SetMaxWidthLength(length Length) {
	length = normalizedLength(length)
	if p.maxWidth == length {
		return
	}
	p.maxWidth = length
	p.SetDirty()
}

// SetMaxHeightLength assigns a maximum height using the unified length type.
func (p *Panel) SetMaxHeightLength(length Length) {
	length = normalizedLength(length)
	if p.maxHeight == length {
		return
	}
	p.maxHeight = length
	p.SetDirty()
}
