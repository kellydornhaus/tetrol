package layout

import "strings"

// Visibility controls whether components participate in layout and rendering.
type Visibility int

const (
	// VisibilityVisible renders and participates in layout (default).
	VisibilityVisible Visibility = iota
	// VisibilityHidden participates in layout but is skipped during rendering.
	VisibilityHidden
	// VisibilityCollapse removes the component from both layout and rendering.
	VisibilityCollapse
)

// String returns the lower-case identifier for the visibility value.
func (v Visibility) String() string {
	switch v {
	case VisibilityHidden:
		return "hidden"
	case VisibilityCollapse:
		return "collapse"
	default:
		return "visible"
	}
}

// renders reports whether the current visibility should produce draw calls.
func (v Visibility) renders() bool { return v == VisibilityVisible }

// participates reports whether the component affects layout calculations.
func (v Visibility) participates() bool { return v != VisibilityCollapse }

// ParseVisibility converts a textual representation into a Visibility value.
func ParseVisibility(raw string) (Visibility, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return VisibilityVisible, false
	case "hidden":
		return VisibilityHidden, true
	case "collapse", "collapsed":
		return VisibilityCollapse, true
	case "visible":
		return VisibilityVisible, true
	default:
		return VisibilityVisible, false
	}
}

type visibilityGetter interface {
	Visibility() Visibility
}

type visibilitySetter interface {
	SetVisibility(Visibility)
}

type visibilityTransitionSetter interface {
	SetVisibilityTransition(VisibilityTransition)
}

type visibilityTransitionGetter interface {
	VisibilityTransition() (VisibilityTransition, bool)
}

// VisibilityOf reports the current visibility for the given component.
func VisibilityOf(c Component) Visibility {
	if c == nil {
		return VisibilityVisible
	}
	if getter, ok := c.(visibilityGetter); ok {
		return getter.Visibility()
	}
	return VisibilityVisible
}

// SetVisibility assigns the given visibility when supported.
func SetVisibility(c Component, visibility Visibility) {
	if setter, ok := c.(visibilitySetter); ok {
		setter.SetVisibility(visibility)
	}
}

// SetVisibilityTransition stores the provided transition when supported.
func SetVisibilityTransition(c Component, transition VisibilityTransition) {
	if setter, ok := c.(visibilityTransitionSetter); ok {
		setter.SetVisibilityTransition(transition)
	}
}

// VisibilityTransitionOf reports the configured visibility transition if supported.
func VisibilityTransitionOf(c Component) (VisibilityTransition, bool) {
	if getter, ok := c.(visibilityTransitionGetter); ok {
		return getter.VisibilityTransition()
	}
	return VisibilityTransition{}, false
}

// participatesInLayout returns true when the component should be considered during layout.
func participatesInLayout(c Component) bool {
	if !VisibilityOf(c).participates() {
		return false
	}
	return !isOutOfFlow(c)
}

// rendersToSurface returns true when the component should issue draw calls.
func rendersToSurface(c Component) bool {
	if c == nil {
		return false
	}
	if provider, ok := c.(interface{ ShouldRender() bool }); ok {
		return provider.ShouldRender()
	}
	return VisibilityOf(c).renders()
}
