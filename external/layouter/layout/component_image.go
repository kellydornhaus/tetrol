package layout

// ImageComponent draws a bitmap with optional panel decorations.
type ImageComponent struct {
	*Panel
	Source             Image
	Fit                ImageFit
	ExplicitSize       Size
	maxWidth           Length
	maxHeight          Length
	AlignH             TextAlign
	AlignV             TextAlign
	explicitWidthAuto  bool
	explicitHeightAuto bool
}

// NewImage creates a new image component with the provided source.
func NewImage(src Image) *ImageComponent {
	return &ImageComponent{
		Panel:        NewPanel(),
		Source:       src,
		Fit:          ImageFitContain,
		ExplicitSize: Size{},
		AlignH:       AlignCenter,
		AlignV:       AlignCenter,
	}
}

func (img *ImageComponent) dialogBase() *Base {
	if img == nil || img.Panel == nil {
		return nil
	}
	return &img.Panel.Base
}

// SetSource updates the image source.
func (img *ImageComponent) SetSource(src Image) {
	if img.Source == src {
		return
	}
	img.Source = src
	img.SetDirty()
}

// SetFit changes how the image is mapped into the available rect.
func (img *ImageComponent) SetFit(fit ImageFit) {
	if img.Fit == fit {
		return
	}
	img.Fit = fit
	img.SetDirty()
}

// SetExplicitSize overrides the intrinsic image size (dp units). Zero fields defer to intrinsic size.
func (img *ImageComponent) SetExplicitSize(sz Size) {
	if img.ExplicitSize == sz && !img.explicitWidthAuto && !img.explicitHeightAuto {
		return
	}
	img.ExplicitSize = sz
	img.explicitWidthAuto = false
	img.explicitHeightAuto = false
	img.SetDirty()
}

// SetExplicitSizeAuto assigns explicit dimensions and allows opting into auto sizing for either axis.
func (img *ImageComponent) SetExplicitSizeAuto(sz Size, widthAuto, heightAuto bool) {
	if img.ExplicitSize == sz && img.explicitWidthAuto == widthAuto && img.explicitHeightAuto == heightAuto {
		return
	}
	img.ExplicitSize = sz
	img.explicitWidthAuto = widthAuto
	img.explicitHeightAuto = heightAuto
	img.SetDirty()
}

// SetMaxWidth constrains the rendered width (dp).
func (img *ImageComponent) SetMaxWidth(w float64) {
	img.SetMaxWidthLength(LengthDP(w))
}

// SetMaxHeight constrains the rendered height (dp).
func (img *ImageComponent) SetMaxHeight(h float64) {
	img.SetMaxHeightLength(LengthDP(h))
}

// SetMaxWidthLength constrains rendered width using a Length.
func (img *ImageComponent) SetMaxWidthLength(length Length) {
	length = normalizedLength(length)
	if img.maxWidth == length {
		return
	}
	img.maxWidth = length
	img.SetDirty()
}

// SetMaxHeightLength constrains rendered height using a Length.
func (img *ImageComponent) SetMaxHeightLength(length Length) {
	length = normalizedLength(length)
	if img.maxHeight == length {
		return
	}
	img.maxHeight = length
	img.SetDirty()
}

// SetMaxWidthPercent constrains rendered width as fraction of available space.
func (img *ImageComponent) SetMaxWidthPercent(percent float64) {
	img.SetMaxWidthLength(LengthPercent(percent))
}

// SetMaxHeightPercent constrains rendered height as fraction of available space.
func (img *ImageComponent) SetMaxHeightPercent(percent float64) {
	img.SetMaxHeightLength(LengthPercent(percent))
}

// SetMaxWidthViewportWidth constrains width using viewport width units.
func (img *ImageComponent) SetMaxWidthViewportWidth(fraction float64) {
	img.SetMaxWidthLength(LengthVW(fraction))
}

// SetMaxHeightViewportHeight constrains height using viewport height units.
func (img *ImageComponent) SetMaxHeightViewportHeight(fraction float64) {
	img.SetMaxHeightLength(LengthVH(fraction))
}

// SetAlignment controls where the image sits inside available space when fit does not stretch.
func (img *ImageComponent) SetAlignment(h, v TextAlign) {
	if img.AlignH == h && img.AlignV == v {
		return
	}
	img.AlignH = h
	img.AlignV = v
	img.SetDirty()
}

func (img *ImageComponent) Measure(ctx *Context, cs Constraints) Size {
	if img.Visibility() == VisibilityCollapse {
		return Size{}
	}
	inner := img.contentConstraints(ctx, cs)
	var content Size
	if img.Source != nil && ctx != nil && ctx.Scale > 0 {
		w, h := img.Source.SizePx()
		content = PxSize{W: w, H: h}.ToDp(ctx.Scale)
	}
	if img.ExplicitSize.W > 0 {
		content.W = img.ExplicitSize.W
	}
	if img.ExplicitSize.H > 0 {
		content.H = img.ExplicitSize.H
	}

	maxW := inner.Max.W
	if resolved := img.maxWidth.ResolveWidth(ctx, inner.Max.W); resolved > 0 && (maxW <= 0 || resolved < maxW) {
		maxW = resolved
	}
	maxH := inner.Max.H
	if resolved := img.maxHeight.ResolveHeight(ctx, inner.Max.H); resolved > 0 && (maxH <= 0 || resolved < maxH) {
		maxH = resolved
	}

	intrinsic := content
	if intrinsic.W <= 0 && img.Source != nil && ctx != nil && ctx.Scale > 0 {
		w, h := img.Source.SizePx()
		intrinsic = PxSize{W: w, H: h}.ToDp(ctx.Scale)
	}

	switch img.Fit {
	case ImageFitContain:
		width := intrinsic.W
		height := intrinsic.H
		if maxW > 0 {
			width = minFloat(width, maxW)
		}
		if maxH > 0 {
			height = minFloat(height, maxH)
		}
		scale := 1.0
		if intrinsic.W > 0 && width > 0 {
			scale = minFloat(scale, width/intrinsic.W)
		}
		if intrinsic.H > 0 && height > 0 {
			scale = minFloat(scale, height/intrinsic.H)
		}
		if img.ExplicitSize.W > 0 {
			width = img.ExplicitSize.W
		} else {
			width = intrinsic.W * scale
		}
		if img.ExplicitSize.H > 0 {
			height = img.ExplicitSize.H
		} else {
			height = intrinsic.H * scale
		}
		content = Size{W: width, H: height}
	case ImageFitCenter:
		width := intrinsic.W
		height := intrinsic.H
		if img.ExplicitSize.W > 0 {
			width = img.ExplicitSize.W
		}
		if img.ExplicitSize.H > 0 {
			height = img.ExplicitSize.H
		}
		if maxW > 0 && width > maxW {
			width = maxW
		}
		if maxH > 0 && height > maxH {
			height = maxH
		}
		content = Size{W: width, H: height}
	default: // stretch
		width := intrinsic.W
		height := intrinsic.H
		if img.ExplicitSize.W > 0 {
			width = img.ExplicitSize.W
		} else if maxW > 0 {
			width = maxW
		}
		if img.ExplicitSize.H > 0 {
			height = img.ExplicitSize.H
		} else if maxH > 0 {
			height = maxH
		}
		if width <= 0 {
			width = intrinsic.W
		}
		if height <= 0 {
			height = intrinsic.H
		}
		content = Size{W: width, H: height}
	}

	ratio := 0.0
	if img.Source != nil && ctx != nil && ctx.Scale > 0 {
		if w, h := img.Source.SizePx(); w > 0 && h > 0 {
			ratio = float64(w) / float64(h)
		}
	}
	if ratio <= 0 && img.Panel != nil {
		if r := img.Panel.AspectRatio(); r > 0 {
			ratio = r
		}
	}
	if ratio <= 0 && img.ExplicitSize.W > 0 && img.ExplicitSize.H > 0 {
		ratio = img.ExplicitSize.W / img.ExplicitSize.H
	}
	if ratio > 0 {
		if img.explicitWidthAuto && !img.explicitHeightAuto && content.H > 0 {
			content.W = content.H * ratio
		}
		if img.explicitHeightAuto && !img.explicitWidthAuto && content.W > 0 {
			content.H = content.W / ratio
		}
	}

	content = clampSizeToConstraints(content, inner)
	result := img.resolveSize(ctx, cs, content)
	logSelfMeasure(ctx, img, cs, result)
	return result
}

func (img *ImageComponent) Layout(ctx *Context, parent Component, bounds Rect) {
	if img.Visibility() == VisibilityCollapse {
		img.SetFrame(parent, Rect{})
		return
	}
	img.SetFrame(parent, bounds)
	logLayoutBounds(ctx, img, parent, bounds)
	img.ContentBounds()
}

func (img *ImageComponent) DrawTo(ctx *Context, dst Surface) {
	if !img.ShouldRender() {
		img.Base.releaseCache()
		return
	}
	img.DrawPanelChildrenWithOwner(ctx, dst, img, func(target Surface) { img.Render(ctx, target) })
}

func (img *ImageComponent) Render(ctx *Context, dst Surface) {
	if ctx == nil || ctx.Renderer == nil || img.Source == nil {
		return
	}
	content := img.ContentBounds()
	if content.W <= 0 || content.H <= 0 {
		return
	}
	imgW, imgH := img.Source.SizePx()
	if imgW <= 0 || imgH <= 0 {
		return
	}
	alignH := img.AlignH
	if alignH == 0 {
		alignH = AlignCenter
	}
	alignV := img.AlignV
	if alignV == 0 {
		alignV = AlignCenter
	}
	rectPx := Rect{X: content.X, Y: content.Y, W: content.W, H: content.H}.ToPx(ctx.Scale)
	drawRect := computeImageRect(rectPx, imgW, imgH, img.Fit, alignH, alignV)
	if drawRect.W <= 0 || drawRect.H <= 0 {
		return
	}
	if ctx != nil {
		if dx, dy := ctx.drawOffset(); dx != 0 || dy != 0 {
			drawRect.X += dx
			drawRect.Y += dy
		}
	}
	ctx.Renderer.DrawImage(dst, img.Source, drawRect)
}
