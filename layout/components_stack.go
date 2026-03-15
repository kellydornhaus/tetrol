package layout

// Justify controls distribution on the main axis for stacks.
type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

// HStack arranges children horizontally with optional spacing and justification.
type HStack struct {
	*Panel
	Children []Component
	Spacing  float64
	AlignV   TextAlign
	Justify  Justify
}

func NewHStack(children ...Component) *HStack {
	h := &HStack{
		Panel:    NewPanel(),
		Children: children,
		Spacing:  0,
		AlignV:   AlignStart,
		Justify:  JustifyStart,
	}
	for _, ch := range children {
		if ch != nil {
			attachDialogParent(ch, h)
		}
	}
	return h
}

func (s *HStack) dialogBase() *Base {
	if s == nil || s.Panel == nil {
		return nil
	}
	return &s.Panel.Base
}

func (s *HStack) Measure(ctx *Context, cs Constraints) Size {
	if s.Visibility() == VisibilityCollapse {
		return Size{}
	}
	inner := s.contentConstraints(ctx, cs)
	var totalW float64
	var maxH float64
	visibleChildren := 0
	for _, ch := range s.Children {
		if ch == nil || !participatesInLayout(ch) {
			continue
		}
		visibleChildren++
		if isFlex(ch) {
			continue
		}
		childCs := Constraints{Max: Size{W: inner.Max.W, H: inner.Max.H}}
		sz := measureComponent(ctx, s, ch, childCs)
		totalW += sz.W
		if sz.H > maxH {
			maxH = sz.H
		}
	}
	if visibleChildren > 1 {
		totalW += s.Spacing * float64(visibleChildren-1)
	}
	result := s.resolveSize(ctx, cs, Size{W: totalW, H: maxH})
	logSelfMeasure(ctx, s, cs, result)
	return result
}

func (s *HStack) Layout(ctx *Context, parent Component, bounds Rect) {
	if s.Visibility() == VisibilityCollapse {
		s.SetFrame(parent, Rect{})
		return
	}
	s.SetFrame(parent, bounds)
	logLayoutBounds(ctx, s, parent, bounds)
	content := s.ContentBounds()
	s.refreshAutoTextSize(ctx, content)
	defer layoutOutOfFlowChildren(ctx, s, s.Children, content)
	if content.W <= 0 {
		return
	}

	active := make([]Component, 0, len(s.Children))
	for _, ch := range s.Children {
		if ch == nil || !participatesInLayout(ch) {
			continue
		}
		active = append(active, ch)
	}
	childCount := len(active)
	if childCount == 0 {
		return
	}

	widths := make([]float64, childCount)
	flexes := make([]float64, childCount)
	flexWeight := 0.0
	totalNonFlexW := 0.0
	for i, ch := range active {
		if fw := flexWeightOf(ch); fw > 0 {
			flexWeight += fw
			flexes[i] = fw
			continue
		}
		childCs := Constraints{Max: Size{W: content.W, H: content.H}}
		size := measureComponent(ctx, s, ch, childCs)
		widths[i] = size.W
		totalNonFlexW += size.W
	}

	gaps := maxInt(childCount-1, 0)
	spacing := s.Spacing
	totalGaps := spacing * float64(gaps)
	available := maxFloat(0, content.W-totalNonFlexW-totalGaps)

	flexSpace := 0.0
	if flexWeight > 0 {
		flexSpace = available
		available = 0
	}

	offsetAvailable := available

	x := content.X
	switch s.Justify {
	case JustifyCenter:
		x += offsetAvailable / 2
	case JustifyEnd:
		x += offsetAvailable
	case JustifySpaceBetween:
		if gaps > 0 {
			spacing = s.Spacing + available/float64(gaps)
		}
		available = 0
	}

	for i, ch := range active {
		cw := widths[i]
		if cw == 0 && flexes[i] > 0 && flexWeight > 0 {
			cw = flexSpace * (flexes[i] / flexWeight)
		}
		childCs := Constraints{Max: Size{W: maxFloat(cw, 0), H: content.H}}
		sz := measureComponent(ctx, s, ch, childCs)
		chHeight := sz.H
		fillHeight := false
		if fh, ok := ch.(interface{ FillHeight() bool }); ok && fh.FillHeight() {
			fillHeight = true
		}
		if fillHeight && content.H > 0 {
			chHeight = content.H
		}
		childY := content.Y
		if fillHeight {
			childY = content.Y
		} else {
			align := s.AlignV
			if aligned, ok := AlignSelfOf(ch); ok {
				align = aligned
			}
			switch align {
			case AlignCenter:
				childY = content.Y + (content.H-chHeight)/2
			case AlignEnd:
				childY = content.Y + (content.H - chHeight)
			default:
				childY = content.Y
			}
		}

		flowRect := Rect{X: x, Y: childY, W: cw, H: chHeight}
		layoutFlowChild(ctx, s, ch, flowRect)
		if ch.Dirty() {
			s.SetDirty()
		}

		x += cw
		if i < childCount-1 {
			x += spacing
		}
	}

}

func (s *HStack) DrawTo(ctx *Context, dst Surface) {
	if !s.ShouldRender() {
		s.Base.releaseCache()
		return
	}
	for _, ch := range s.Children {
		if ch != nil && ch.Dirty() {
			s.SetDirty()
			break
		}
	}
	s.DrawPanelChildrenWithOwner(ctx, dst, s, func(target Surface) { s.Render(ctx, target) })
}

func (s *HStack) Render(ctx *Context, dst Surface) {
	ordered := orderByZIndex(s.Children)
	for _, ch := range ordered {
		if ch != nil && rendersToSurface(ch) {
			ch.DrawTo(ctx, dst)
		}
	}
}

// VStack arranges children vertically.
type VStack struct {
	*Panel
	Children []Component
	Spacing  float64
	AlignH   TextAlign
	Justify  Justify
}

func NewVStack(children ...Component) *VStack {
	v := &VStack{
		Panel:    NewPanel(),
		Children: children,
		Spacing:  0,
		AlignH:   AlignStart,
		Justify:  JustifyStart,
	}
	for _, ch := range children {
		if ch != nil {
			attachDialogParent(ch, v)
		}
	}
	return v
}

func (s *VStack) dialogBase() *Base {
	if s == nil || s.Panel == nil {
		return nil
	}
	return &s.Panel.Base
}

func (s *VStack) Measure(ctx *Context, cs Constraints) Size {
	if s.Visibility() == VisibilityCollapse {
		return Size{}
	}
	inner := s.contentConstraints(ctx, cs)
	var totalH float64
	var maxW float64
	visibleChildren := 0
	for _, ch := range s.Children {
		if ch == nil || !participatesInLayout(ch) {
			continue
		}
		visibleChildren++
		if isFlex(ch) {
			continue
		}
		childCs := Constraints{Max: Size{W: inner.Max.W, H: inner.Max.H}}
		sz := measureComponent(ctx, s, ch, childCs)
		totalH += sz.H
		if sz.W > maxW {
			maxW = sz.W
		}
	}
	if visibleChildren > 1 {
		totalH += s.Spacing * float64(visibleChildren-1)
	}
	result := s.resolveSize(ctx, cs, Size{W: maxW, H: totalH})
	logSelfMeasure(ctx, s, cs, result)
	return result
}

func (s *VStack) Layout(ctx *Context, parent Component, bounds Rect) {
	if s.Visibility() == VisibilityCollapse {
		s.SetFrame(parent, Rect{})
		return
	}
	s.SetFrame(parent, bounds)
	logLayoutBounds(ctx, s, parent, bounds)
	content := s.ContentBounds()
	s.refreshAutoTextSize(ctx, content)
	defer layoutOutOfFlowChildren(ctx, s, s.Children, content)
	if content.H <= 0 {
		return
	}

	active := make([]Component, 0, len(s.Children))
	for _, ch := range s.Children {
		if ch == nil || !participatesInLayout(ch) {
			continue
		}
		active = append(active, ch)
	}
	childCount := len(active)
	if childCount == 0 {
		return
	}

	heights := make([]float64, childCount)
	flexes := make([]float64, childCount)
	flexWeight := 0.0
	totalNonFlexH := 0.0
	for i, ch := range active {
		if fw := flexWeightOf(ch); fw > 0 {
			flexWeight += fw
			flexes[i] = fw
			continue
		}
		childCs := Constraints{Max: Size{W: content.W, H: content.H}}
		sz := measureComponent(ctx, s, ch, childCs)
		heights[i] = sz.H
		totalNonFlexH += sz.H
	}

	gaps := maxInt(childCount-1, 0)
	spacing := s.Spacing
	totalGaps := spacing * float64(gaps)
	available := maxFloat(0, content.H-totalNonFlexH-totalGaps)

	flexSpace := 0.0
	if flexWeight > 0 {
		flexSpace = available
		available = 0
	}

	offsetAvailable := available

	y := content.Y
	switch s.Justify {
	case JustifyCenter:
		y += offsetAvailable / 2
	case JustifyEnd:
		y += offsetAvailable
	case JustifySpaceBetween:
		if gaps > 0 {
			spacing = s.Spacing + available/float64(gaps)
		}
		available = 0
	}

	for i, ch := range active {
		chH := heights[i]
		if chH == 0 && flexes[i] > 0 && flexWeight > 0 {
			chH = flexSpace * (flexes[i] / flexWeight)
		}

		childCs := Constraints{Max: Size{W: content.W, H: maxFloat(chH, 0)}}
		sz := measureComponent(ctx, s, ch, childCs)

		fillWidth := false
		if fw, ok := ch.(interface{ FillWidth() bool }); ok && fw.FillWidth() {
			fillWidth = true
		}

		childW := sz.W
		if fillWidth || childW > content.W {
			childW = content.W
		}
		childX := content.X
		if !fillWidth && childW < content.W {
			align := s.AlignH
			if aligned, ok := AlignSelfOf(ch); ok {
				align = aligned
			}
			switch align {
			case AlignCenter:
				childX = content.X + (content.W-childW)/2
			case AlignEnd:
				childX = content.X + (content.W - childW)
			default:
				childX = content.X
			}
		}

		flowRect := Rect{X: childX, Y: y, W: childW, H: chH}
		layoutFlowChild(ctx, s, ch, flowRect)
		if ch.Dirty() {
			s.SetDirty()
		}

		y += chH
		if i < childCount-1 {
			y += spacing
		}
	}

}

func (s *VStack) DrawTo(ctx *Context, dst Surface) {
	if !s.ShouldRender() {
		s.Base.releaseCache()
		return
	}
	for _, ch := range s.Children {
		if ch != nil && ch.Dirty() {
			s.SetDirty()
			break
		}
	}
	s.DrawPanelChildrenWithOwner(ctx, dst, s, func(target Surface) { s.Render(ctx, target) })
}

func (s *VStack) Render(ctx *Context, dst Surface) {
	ordered := orderByZIndex(s.Children)
	for _, ch := range ordered {
		if ch != nil && rendersToSurface(ch) {
			ch.DrawTo(ctx, dst)
		}
	}
}

// ZStack overlays children, giving each the full bounds (z-ordered by slice order).
type ZStack struct {
	*Panel
	Children []Component
}

func NewZStack(children ...Component) *ZStack {
	z := &ZStack{
		Panel:    NewPanel(),
		Children: children,
	}
	z.SetFillWidth(true)
	for _, ch := range children {
		if ch != nil {
			attachDialogParent(ch, z)
		}
	}
	return z
}

func (s *ZStack) dialogBase() *Base {
	if s == nil || s.Panel == nil {
		return nil
	}
	return &s.Panel.Base
}

func (s *ZStack) Measure(ctx *Context, cs Constraints) Size {
	if s.Visibility() == VisibilityCollapse {
		return Size{}
	}
	inner := s.contentConstraints(ctx, cs)
	var maxW, maxH float64
	for _, ch := range s.Children {
		if ch == nil || !participatesInLayout(ch) {
			continue
		}
		sz := measureComponent(ctx, s, ch, inner)
		if sz.W > maxW {
			maxW = sz.W
		}
		if sz.H > maxH {
			maxH = sz.H
		}
	}
	result := s.resolveSize(ctx, cs, Size{W: maxW, H: maxH})
	logSelfMeasure(ctx, s, cs, result)
	return result
}

func (s *ZStack) Layout(ctx *Context, parent Component, bounds Rect) {
	if s.Visibility() == VisibilityCollapse {
		s.SetFrame(parent, Rect{})
		return
	}
	s.SetFrame(parent, bounds)
	logLayoutBounds(ctx, s, parent, bounds)
	content := s.ContentBounds()
	s.refreshAutoTextSize(ctx, content)
	childConstraints := Constraints{}
	if content.W > 0 {
		childConstraints.Max.W = content.W
	}
	if content.H > 0 {
		childConstraints.Max.H = content.H
	}
	for _, ch := range s.Children {
		if ch == nil {
			continue
		}
		mode := PositionModeOf(ch)
		if mode == PositionAbsolute || mode == PositionFixed {
			layoutOutOfFlowChild(ctx, s, ch, content)
		} else if participatesInLayout(ch) {
			flowRect := Rect{X: content.X, Y: content.Y, W: maxFloat(content.W, 0), H: maxFloat(content.H, 0)}
			size := measureComponent(ctx, s, ch, childConstraints)

			fillWidth := false
			if fw, ok := ch.(interface{ FillWidth() bool }); ok && fw.FillWidth() {
				fillWidth = true
			}
			fillHeight := false
			if fh, ok := ch.(interface{ FillHeight() bool }); ok && fh.FillHeight() {
				fillHeight = true
			}

			if !fillWidth {
				if size.W > 0 {
					if content.W > 0 {
						flowRect.W = minFloat(size.W, content.W)
					} else {
						flowRect.W = size.W
					}
				} else if content.W <= 0 {
					flowRect.W = maxFloat(size.W, 0)
				}
			} else if flowRect.W <= 0 && size.W > 0 {
				flowRect.W = size.W
			}

			if !fillHeight {
				if size.H > 0 {
					if content.H > 0 {
						flowRect.H = minFloat(size.H, content.H)
					} else {
						flowRect.H = size.H
					}
				} else if content.H <= 0 {
					flowRect.H = maxFloat(size.H, 0)
				}
			} else if flowRect.H <= 0 && size.H > 0 {
				flowRect.H = size.H
			}

			// Align within the content rect when not filling the axis.
			if !fillWidth && flowRect.W > 0 && content.W > 0 && flowRect.W < content.W {
				align := AlignStart
				if override, ok := AlignSelfOf(ch); ok {
					align = override
				}
				switch align {
				case AlignCenter:
					flowRect.X = content.X + (content.W-flowRect.W)/2
				case AlignEnd:
					flowRect.X = content.X + (content.W - flowRect.W)
				default:
					flowRect.X = content.X
				}
			}
			if !fillHeight && flowRect.H > 0 && content.H > 0 && flowRect.H < content.H {
				align := AlignStart
				if override, ok := AlignSelfOf(ch); ok {
					align = override
				}
				switch align {
				case AlignCenter:
					flowRect.Y = content.Y + (content.H-flowRect.H)/2
				case AlignEnd:
					flowRect.Y = content.Y + (content.H - flowRect.H)
				default:
					flowRect.Y = content.Y
				}
			}
			layoutFlowChild(ctx, s, ch, flowRect)
		}
		if ch.Dirty() {
			s.SetDirty()
		}
	}
}

func (s *ZStack) DrawTo(ctx *Context, dst Surface) {
	if !s.ShouldRender() {
		s.Base.releaseCache()
		return
	}
	for _, ch := range s.Children {
		if ch != nil && ch.Dirty() {
			s.SetDirty()
			break
		}
	}
	s.DrawPanelChildrenWithOwner(ctx, dst, s, func(target Surface) { s.Render(ctx, target) })
}

func (s *ZStack) Render(ctx *Context, dst Surface) {
	ordered := orderByZIndex(s.Children)
	for _, ch := range ordered {
		if ch != nil && rendersToSurface(ch) {
			ch.DrawTo(ctx, dst)
		}
	}
}

// Spacer is a flexible component that expands to fill extra space in stacks.
type Spacer struct {
	Base
	Weight float64
}

func NewSpacer(weight float64) *Spacer {
	if weight <= 0 {
		weight = 1
	}
	return &Spacer{Weight: weight}
}

func (s *Spacer) dialogBase() *Base {
	if s == nil {
		return nil
	}
	return &s.Base
}

func (s *Spacer) Measure(ctx *Context, cs Constraints) Size {
	size := Size{}
	logSelfMeasure(ctx, s, cs, size)
	return size
}

func (s *Spacer) Layout(ctx *Context, parent Component, bounds Rect) {
	if s.Visibility() == VisibilityCollapse {
		s.SetFrame(parent, Rect{})
		return
	}
	s.SetFrame(parent, bounds)
	logLayoutBounds(ctx, s, parent, bounds)
}

func (s *Spacer) DrawTo(ctx *Context, dst Surface) {}

func (s *Spacer) Render(ctx *Context, dst Surface) {}

func (s *Spacer) SetFlexWeight(weight float64) {
	if weight <= 0 {
		s.Weight = 0
	} else {
		s.Weight = weight
	}
}

func (s *Spacer) FlexWeight() float64 {
	if s.Weight <= 0 {
		return 0
	}
	return s.Weight
}

func flexWeightOf(c Component) float64 {
	if c == nil || !participatesInLayout(c) {
		return 0
	}
	if fw, ok := c.(interface{ FlexWeight() float64 }); ok {
		w := fw.FlexWeight()
		if w > 0 {
			return w
		}
	}
	return 0
}

func isFlex(c Component) bool {
	return flexWeightOf(c) > 0
}
