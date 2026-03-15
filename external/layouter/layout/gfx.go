package layout

import "math"

// The layout engine is rendering-backend agnostic.
// Callers integrate by implementing these interfaces for their chosen backend (e.g. Ebiten).

// PixelScaleProvider should return the device scale factor for the target monitor.
// With Ebiten, provide ebiten.Monitor().DeviceScaleFactor().
type PixelScaleProvider interface {
	DeviceScaleFactor() float64
}

// Surface is a 2D pixel render target/image. Implemented by adapters (e.g., wrapping *ebiten.Image).
type Surface interface {
	// SizePx returns surface dimensions in pixels.
	SizePx() (w int, h int)
	// Clear the surface with transparent pixels.
	Clear()
}

// Canvas is the destination surface for a frame. Often the screen backbuffer.
// It is also a Surface itself.
type Canvas interface {
	Surface
}

// Image represents an immutable bitmap provided by the rendering backend.
type Image interface {
	SizePx() (w int, h int)
}

// PxCornerRadii stores per-corner radii in pixel units.
type PxCornerRadii struct {
	TopLeft     float64
	TopRight    float64
	BottomRight float64
	BottomLeft  float64
}

// HasRadius reports whether any corner radius is positive.
func (r PxCornerRadii) HasRadius() bool {
	return r.TopLeft > 0 || r.TopRight > 0 || r.BottomRight > 0 || r.BottomLeft > 0
}

// Normalized scales the radii to fit inside width/height (CSS style).
func (r PxCornerRadii) Normalized(width, height float64) PxCornerRadii {
	if width <= 0 || height <= 0 {
		return PxCornerRadii{}
	}
	scale := 1.0
	sumTop := r.TopLeft + r.TopRight
	if sumTop > width && sumTop > 0 {
		scale = math.Min(scale, width/sumTop)
	}
	sumBottom := r.BottomLeft + r.BottomRight
	if sumBottom > width && sumBottom > 0 {
		scale = math.Min(scale, width/sumBottom)
	}
	sumLeft := r.TopLeft + r.BottomLeft
	if sumLeft > height && sumLeft > 0 {
		scale = math.Min(scale, height/sumLeft)
	}
	sumRight := r.TopRight + r.BottomRight
	if sumRight > height && sumRight > 0 {
		scale = math.Min(scale, height/sumRight)
	}
	if scale >= 1 {
		return r
	}
	return PxCornerRadii{
		TopLeft:     r.TopLeft * scale,
		TopRight:    r.TopRight * scale,
		BottomRight: r.BottomRight * scale,
		BottomLeft:  r.BottomLeft * scale,
	}
}

// Inset reduces each radius by delta (px), clamping at zero.
func (r PxCornerRadii) Inset(delta float64) PxCornerRadii {
	if delta <= 0 {
		return r
	}
	return PxCornerRadii{
		TopLeft:     maxFloat(0, r.TopLeft-delta),
		TopRight:    maxFloat(0, r.TopRight-delta),
		BottomRight: maxFloat(0, r.BottomRight-delta),
		BottomLeft:  maxFloat(0, r.BottomLeft-delta),
	}
}

// Renderer creates surfaces and composites them. Implement using your backend.
type Renderer interface {
	// NewSurface creates a new offscreen surface with the given pixel size.
	NewSurface(pxW, pxH int) Surface
	// DrawSurface draws src at (x,y) onto dst in pixel units.
	DrawSurface(dst Surface, src Surface, x, y int)
	// FillRect paints a solid color into the destination rectangle.
	FillRect(dst Surface, rect PxRect, color Color)
	// TintRect overlays color onto dst within rect using destination alpha as a mask.
	TintRect(dst Surface, rect PxRect, color Color)
	// DrawImage draws img into rect (scaled as needed).
	DrawImage(dst Surface, img Image, rect PxRect)
}

// SurfaceScaler optionally augments Renderer with smooth scaling support.
type SurfaceScaler interface {
	// DrawSurfaceScaled draws src within bounds, scaling around its center.
	DrawSurfaceScaled(dst Surface, src Surface, bounds PxRect, scaleX, scaleY float64)
}

// SurfaceRectRenderer optionally augments Renderer with scaled surface compositing.
type SurfaceRectRenderer interface {
	// DrawSurfaceRect draws the src surface scaled and positioned to fit rect on dst.
	DrawSurfaceRect(dst Surface, src Surface, rect PxRect)
}

// RoundedRenderer optionally augments Renderer with rounded-rectangle drawing.
type RoundedRenderer interface {
	// FillRoundedRect draws a filled rounded rectangle using the supplied color.
	FillRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, color Color)
	// TintRoundedRect overlays color inside the rounded rectangle, masking with destination alpha.
	TintRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, color Color)
	// StrokeRoundedRect draws a rounded rectangle border with the given stroke width (pixels).
	StrokeRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, strokeWidth float64, color Color)
}

// Color represents RGBA in premultiplied or straight form as defined by the adapter.
// The core engine doesn't manipulate it; it passes through to text adapters.
type Color struct {
	R, G, B, A uint8
}
