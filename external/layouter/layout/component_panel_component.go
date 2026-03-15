package layout

// PanelComponent is a versatile container that applies panel decorations to an optional child.
type PanelComponent struct {
	*Panel
	Child Component
}

// NewPanelComponent creates a panel wrapping the provided child (nil allowed).
func NewPanelComponent(child Component) *PanelComponent {
	p := &PanelComponent{
		Panel: NewPanel(),
		Child: child,
	}
	if child != nil {
		attachDialogParent(child, p)
	}
	return p
}

func (p *PanelComponent) dialogBase() *Base {
	if p == nil || p.Panel == nil {
		return nil
	}
	return &p.Panel.Base
}

// NewPanelContainer creates a panel with padding around the given child.
func NewPanelContainer(child Component, padding EdgeInsets) *PanelComponent {
	p := NewPanelComponent(child)
	p.SetPadding(padding)
	return p
}

// NewLabel constructs a text-only panel using the provided style.
func NewLabel(text string, style TextStyle) *PanelComponent {
	p := NewPanelComponent(nil)
	p.SetText(text)
	p.SetTextStyle(style)
	return p
}

// SetChild replaces the contained child component.
func (p *PanelComponent) SetChild(child Component) {
	if p.Child == child {
		return
	}
	attachDialogParent(p.Child, nil)
	p.Child = child
	if child != nil {
		attachDialogParent(child, p)
	}
	p.SetDirty()
}

func (p *PanelComponent) Measure(ctx *Context, cs Constraints) Size {
	if p.Visibility() == VisibilityCollapse {
		return Size{}
	}
	var content Size
	inner := p.contentConstraints(ctx, cs)
	if p.Child != nil && !isOutOfFlow(p.Child) {
		content = measureComponent(ctx, p, p.Child, inner)
	}
	result := p.resolveSize(ctx, cs, content)
	logSelfMeasure(ctx, p, cs, result)
	return result
}

func (p *PanelComponent) Layout(ctx *Context, parent Component, bounds Rect) {
	if p.Visibility() == VisibilityCollapse {
		p.SetFrame(parent, Rect{})
		return
	}
	p.SetFrame(parent, bounds)
	logLayoutBounds(ctx, p, parent, bounds)
	content := p.ContentBounds()
	p.refreshAutoTextSize(ctx, content)
	if p.Child == nil {
		return
	}
	mode := PositionModeOf(p.Child)
	flowRect := Rect{X: content.X, Y: content.Y, W: content.W, H: content.H}
	if mode == PositionAbsolute || mode == PositionFixed {
		layoutOutOfFlowChild(ctx, p, p.Child, content)
	} else {
		layoutFlowChild(ctx, p, p.Child, flowRect)
	}
	if p.Child.Dirty() {
		p.SetDirty()
	}
}

func (p *PanelComponent) DrawTo(ctx *Context, dst Surface) {
	if !p.ShouldRender() {
		p.Base.releaseCache()
		return
	}
	if p.Child != nil && p.Child.Dirty() {
		p.SetDirty()
	}
	p.DrawPanelChildrenWithOwner(ctx, dst, p, func(target Surface) { p.Render(ctx, target) })
}

func (p *PanelComponent) Render(ctx *Context, dst Surface) {
	if p.Child != nil {
		p.Child.DrawTo(ctx, dst)
	}
}
