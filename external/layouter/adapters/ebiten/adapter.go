package ebitenadapter

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kellydornhaus/layouter/layout"
)

// Surface wraps an ebiten.Image to satisfy layout.Surface and layout.Canvas.
type Surface struct{ Img *ebiten.Image }

func (s *Surface) SizePx() (int, int) { return s.Img.Size() }
func (s *Surface) Clear()             { s.Img.Clear() }

// AsEbiten allows other adapters (like etxt) to retrieve the underlying ebiten image.
func (s *Surface) AsEbiten() *ebiten.Image { return s.Img }

// Image wraps an ebiten.Image to satisfy layout.Image.
type Image struct{ Img *ebiten.Image }

func (i *Image) SizePx() (int, int) { return i.Img.Size() }

func (i *Image) SubImage(rect layout.PxRect) layout.Image {
	if i == nil || i.Img == nil {
		return nil
	}
	if rect.W <= 0 || rect.H <= 0 {
		return nil
	}
	r := image.Rect(rect.X, rect.Y, rect.X+rect.W, rect.Y+rect.H)
	r = r.Intersect(i.Img.Bounds())
	if r.Empty() {
		return nil
	}
	sub := i.Img.SubImage(r).(*ebiten.Image)
	return &Image{Img: sub}
}

// Renderer implements layout.Renderer using Ebiten.
type Renderer struct {
	solid         *ebiten.Image
	roundedShader *ebiten.Shader
}

func NewRenderer() *Renderer { return &Renderer{} }

func (r *Renderer) ensureSolid() *ebiten.Image {
	if r.solid == nil {
		img := ebiten.NewImage(1, 1)
		img.Fill(color.White)
		r.solid = img
	}
	return r.solid
}

func (r *Renderer) NewSurface(pxW, pxH int) layout.Surface {
	s := &Surface{Img: ebiten.NewImage(pxW, pxH)}
	return s
}

func (r *Renderer) DrawSurface(dst layout.Surface, src layout.Surface, x, y int) {
	d := dst.(*Surface).Img
	s := src.(*Surface).Img
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	d.DrawImage(s, op)
}

func (r *Renderer) DrawSurfaceScaled(dst layout.Surface, src layout.Surface, bounds layout.PxRect, scaleX, scaleY float64) {
	if bounds.W <= 0 || bounds.H <= 0 || scaleX <= 0 || scaleY <= 0 {
		return
	}
	dSurface, okDst := dst.(*Surface)
	sSurface, okSrc := src.(*Surface)
	if !okDst || !okSrc || dSurface.Img == nil || sSurface.Img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	cx := float64(bounds.X) + float64(bounds.W)/2
	cy := float64(bounds.Y) + float64(bounds.H)/2
	op.GeoM.Translate(-float64(bounds.W)/2, -float64(bounds.H)/2)
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(cx, cy)
	dSurface.Img.DrawImage(sSurface.Img, op)
}

func (r *Renderer) DrawSurfaceRect(dst layout.Surface, src layout.Surface, rect layout.PxRect) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	d := dst.(*Surface).Img
	s := src.(*Surface).Img
	sw, sh := s.Size()
	if sw <= 0 || sh <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.W)/float64(sw), float64(rect.H)/float64(sh))
	op.GeoM.Translate(float64(rect.X), float64(rect.Y))
	d.DrawImage(s, op)
}

func (r *Renderer) FillRect(dst layout.Surface, rect layout.PxRect, clr layout.Color) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if clr.A == 0 {
		return
	}
	d := dst.(*Surface).Img
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.W), float64(rect.H))
	op.GeoM.Translate(float64(rect.X), float64(rect.Y))
	rMul := float64(clr.R) / 255.0
	gMul := float64(clr.G) / 255.0
	bMul := float64(clr.B) / 255.0
	aMul := float64(clr.A) / 255.0
	op.ColorM.Scale(rMul, gMul, bMul, aMul)
	d.DrawImage(r.ensureSolid(), op)
}

func (r *Renderer) TintRect(dst layout.Surface, rect layout.PxRect, clr layout.Color) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if clr.A == 0 {
		return
	}
	d := dst.(*Surface).Img
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.W), float64(rect.H))
	op.GeoM.Translate(float64(rect.X), float64(rect.Y))
	rMul := float64(clr.R) / 255.0
	gMul := float64(clr.G) / 255.0
	bMul := float64(clr.B) / 255.0
	aMul := float64(clr.A) / 255.0
	op.ColorM.Scale(rMul, gMul, bMul, aMul)
	op.CompositeMode = ebiten.CompositeModeSourceAtop
	d.DrawImage(r.ensureSolid(), op)
}

func (r *Renderer) FillRoundedRect(dst layout.Surface, rect layout.PxRect, radii layout.PxCornerRadii, clr layout.Color) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if clr.A == 0 {
		return
	}
	if r.drawRoundedRectShader(dst, rect, radii, 0, clr, ebiten.CompositeModeSourceOver) {
		return
	}
	path, ok := buildRoundedPath(rect, radii, 0)
	if !ok {
		return
	}
	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	r.drawTriangles(dst.(*Surface).Img, vertices, indices, clr, ebiten.CompositeModeSourceOver)
}

func (r *Renderer) StrokeRoundedRect(dst layout.Surface, rect layout.PxRect, radii layout.PxCornerRadii, strokeWidth float64, clr layout.Color) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if strokeWidth <= 0 {
		return
	}
	if clr.A == 0 {
		return
	}
	width := float64(rect.W)
	height := float64(rect.H)
	if strokeWidth*2 >= width || strokeWidth*2 >= height {
		r.FillRoundedRect(dst, rect, radii, clr)
		return
	}
	if r.drawRoundedRectShader(dst, rect, radii, strokeWidth, clr, ebiten.CompositeModeSourceOver) {
		return
	}
	path, ok := buildRoundedPath(rect, radii, strokeWidth/2)
	if !ok {
		return
	}
	opts := &vector.StrokeOptions{
		Width:    float32(strokeWidth),
		LineJoin: vector.LineJoinRound,
	}
	vertices, indices := path.AppendVerticesAndIndicesForStroke(nil, nil, opts)
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	r.drawTriangles(dst.(*Surface).Img, vertices, indices, clr, ebiten.CompositeModeSourceOver)
}

func (r *Renderer) TintRoundedRect(dst layout.Surface, rect layout.PxRect, radii layout.PxCornerRadii, clr layout.Color) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	if clr.A == 0 {
		return
	}
	if r.drawRoundedRectShader(dst, rect, radii, 0, clr, ebiten.CompositeModeSourceAtop) {
		return
	}
	path, ok := buildRoundedPath(rect, radii, 0)
	if !ok {
		return
	}
	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	r.drawTriangles(dst.(*Surface).Img, vertices, indices, clr, ebiten.CompositeModeSourceAtop)
}

func (r *Renderer) DrawImage(dst layout.Surface, img layout.Image, rect layout.PxRect) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	d := dst.(*Surface).Img
	src := img.(*Image).Img
	sw, sh := src.Size()
	if sw <= 0 || sh <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.W)/float64(sw), float64(rect.H)/float64(sh))
	op.GeoM.Translate(float64(rect.X), float64(rect.Y))
	d.DrawImage(src, op)
}

// WrapCanvas wraps the given ebiten screen image as a layout.Canvas.
func WrapCanvas(screen *ebiten.Image) layout.Canvas { return &Surface{Img: screen} }

// WrapImage exposes an ebiten.Image as a layout.Image.
func WrapImage(img *ebiten.Image) layout.Image { return &Image{Img: img} }

// ScaleProvider implements layout.PixelScaleProvider using ebiten.Monitor().
type ScaleProvider struct{}

func (ScaleProvider) DeviceScaleFactor() float64 {
	m := ebiten.Monitor()
	if m == nil {
		return 1
	}
	return m.DeviceScaleFactor()
}

func (r *Renderer) drawTriangles(dst *ebiten.Image, vertices []ebiten.Vertex, indices []uint16, clr layout.Color, mode ebiten.CompositeMode) {
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	cr, cg, cb, ca := colorComponents(clr)
	if ca <= 0 {
		return
	}
	for i := range vertices {
		vertices[i].SrcX = 0
		vertices[i].SrcY = 0
		vertices[i].ColorR = cr
		vertices[i].ColorG = cg
		vertices[i].ColorB = cb
		vertices[i].ColorA = ca
	}
	op := &ebiten.DrawTrianglesOptions{
		ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha,
	}
	op.AntiAlias = true
	op.CompositeMode = mode
	dst.DrawTriangles(vertices, indices, r.ensureSolid(), op)
}

func (r *Renderer) drawRoundedRectShader(dst layout.Surface, rect layout.PxRect, radii layout.PxCornerRadii, strokeWidth float64, clr layout.Color, mode ebiten.CompositeMode) bool {
	if rect.W <= 0 || rect.H <= 0 {
		return false
	}
	shader := r.ensureRoundedShader()
	if shader == nil {
		return false
	}
	cr, cg, cb, ca := colorComponents(clr)
	if ca <= 0 {
		return false
	}
	op := &ebiten.DrawRectShaderOptions{
		CompositeMode: mode,
		Uniforms: map[string]any{
			"Size":   []float32{float32(rect.W), float32(rect.H)},
			"Radii":  []float32{float32(math.Max(0, radii.TopLeft)), float32(math.Max(0, radii.TopRight)), float32(math.Max(0, radii.BottomRight)), float32(math.Max(0, radii.BottomLeft))},
			"Stroke": float32(strokeWidth),
			"Color":  []float32{cr, cg, cb, ca},
		},
	}
	op.GeoM.Translate(float64(rect.X), float64(rect.Y))
	dst.(*Surface).Img.DrawRectShader(rect.W, rect.H, shader, op)
	return true
}

func (r *Renderer) ensureRoundedShader() *ebiten.Shader {
	if r.roundedShader != nil {
		return r.roundedShader
	}
	shader, err := ebiten.NewShader([]byte(roundedRectShaderSrc))
	if err != nil {
		return nil
	}
	r.roundedShader = shader
	return shader
}

const roundedRectShaderSrc = `//kage:unit pixels
package main

var Size vec2
var Radii vec4
var Stroke float
var Color vec4

func smoothstep(edge0, edge1, x float) float {
	t := clamp((x-edge0)/(edge1-edge0), 0.0, 1.0)
	return t*t*(3.0-2.0*t)
}

func roundedAlpha(p vec2, size vec2, r vec4) float {
	if p.x < 0.0 || p.y < 0.0 || p.x > size.x || p.y > size.y {
		return 0.0
	}
	aa := 1.0
	if r.x > 0.0 && p.x < r.x && p.y < r.x {
		d := r.x - length(vec2(r.x-p.x, r.x-p.y))
		return smoothstep(0.0, aa, d)
	}
	if r.y > 0.0 && p.x > size.x-r.y && p.y < r.y {
		d := r.y - length(vec2(p.x-(size.x-r.y), r.y-p.y))
		return smoothstep(0.0, aa, d)
	}
	if r.z > 0.0 && p.x > size.x-r.z && p.y > size.y-r.z {
		d := r.z - length(vec2(p.x-(size.x-r.z), p.y-(size.y-r.z)))
		return smoothstep(0.0, aa, d)
	}
	if r.w > 0.0 && p.x < r.w && p.y > size.y-r.w {
		d := r.w - length(vec2(r.w-p.x, p.y-(size.y-r.w)))
		return smoothstep(0.0, aa, d)
	}
	dist := min(min(p.x, size.x-p.x), min(p.y, size.y-p.y))
	return smoothstep(0.0, aa, dist)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := dstPos.xy - imageDstOrigin()
	outer := roundedAlpha(pos, Size, Radii)
	alpha := outer
	if Stroke > 0.0 {
		innerSize := Size - vec2(Stroke*2.0, Stroke*2.0)
		if innerSize.x > 0.0 && innerSize.y > 0.0 {
			innerPos := pos - vec2(Stroke, Stroke)
			innerRadii := max(Radii-vec4(Stroke, Stroke, Stroke, Stroke), vec4(0.0))
			inner := roundedAlpha(innerPos, innerSize, innerRadii)
			alpha = clamp(outer-inner, 0.0, 1.0)
		}
	}
	return Color * alpha
}
`

func colorComponents(clr layout.Color) (float32, float32, float32, float32) {
	c := color.NRGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A}
	r16, g16, b16, a16 := c.RGBA()
	return float32(r16) / 0xffff, float32(g16) / 0xffff, float32(b16) / 0xffff, float32(a16) / 0xffff
}

func buildRoundedPath(rect layout.PxRect, radii layout.PxCornerRadii, inset float64) (*vector.Path, bool) {
	width := float64(rect.W)
	height := float64(rect.H)
	if width <= 0 || height <= 0 {
		return nil, false
	}
	x := float64(rect.X)
	y := float64(rect.Y)
	if inset > 0 {
		width -= inset * 2
		height -= inset * 2
		if width <= 0 || height <= 0 {
			return nil, false
		}
		x += inset
		y += inset
	}
	adjusted := radii
	if inset > 0 {
		adjusted = adjusted.Inset(inset)
	}
	adjusted = adjusted.Normalized(width, height)

	var path vector.Path
	path.MoveTo(float32(x+adjusted.TopLeft), float32(y))
	path.LineTo(float32(x+width-adjusted.TopRight), float32(y))
	if adjusted.TopRight > 0 {
		path.ArcTo(float32(x+width), float32(y), float32(x+width), float32(y+adjusted.TopRight), float32(adjusted.TopRight))
	} else {
		path.LineTo(float32(x+width), float32(y))
	}
	path.LineTo(float32(x+width), float32(y+height-adjusted.BottomRight))
	if adjusted.BottomRight > 0 {
		path.ArcTo(float32(x+width), float32(y+height), float32(x+width-adjusted.BottomRight), float32(y+height), float32(adjusted.BottomRight))
	} else {
		path.LineTo(float32(x+width), float32(y+height))
	}
	path.LineTo(float32(x+adjusted.BottomLeft), float32(y+height))
	if adjusted.BottomLeft > 0 {
		path.ArcTo(float32(x), float32(y+height), float32(x), float32(y+height-adjusted.BottomLeft), float32(adjusted.BottomLeft))
	} else {
		path.LineTo(float32(x), float32(y+height))
	}
	path.LineTo(float32(x), float32(y+adjusted.TopLeft))
	if adjusted.TopLeft > 0 {
		path.ArcTo(float32(x), float32(y), float32(x+adjusted.TopLeft), float32(y), float32(adjusted.TopLeft))
	} else {
		path.LineTo(float32(x), float32(y))
	}
	path.Close()
	return &path, true
}
