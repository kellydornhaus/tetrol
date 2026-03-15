package xmlui

import (
	"fmt"

	"github.com/kellydornhaus/layouter/layout"
)

// Template represents a reusable subtree defined by a <template> element.
type Template struct {
	id     string
	node   *Node
	loader *Loader
}

// ID returns the template identifier.
func (t *Template) ID() string {
	if t == nil {
		return ""
	}
	return t.id
}

// Instantiate builds a fresh set of components from the template definition.
func (t *Template) Instantiate() (TemplateInstance, error) {
	if t == nil || t.loader == nil || t.node == nil {
		return TemplateInstance{}, fmt.Errorf("xmlui: template not initialised")
	}
	source := cloneNode(t.node)
	if source == nil {
		return TemplateInstance{}, nil
	}
	source.resetRecursive()
	if len(t.loader.Options.Classes) > 0 {
		applyExtraClasses(source, t.loader.Options.Classes)
	}
	if t.loader.Options.Styles != nil {
		applyStylesRecursive(source, t.loader.Options.Styles)
	}

	instLoader := t.loader.spawnForTemplate(source)
	comps, err := instLoader.BuildChildren(source)
	if err != nil {
		return TemplateInstance{}, err
	}
	instance := TemplateInstance{
		Components: comps,
		ByID:       instLoader.byID,
		styler:     &styler{loader: instLoader},
	}
	return instance, nil
}

// TemplateInstance is the result of instantiating a Template.
type TemplateInstance struct {
	Components []layout.Component
	ByID       map[string]layout.Component
	styler     *styler
}

// AddClass attaches a class to a component in this instance, reapplies styles, and reports whether it changed.
func (ti TemplateInstance) AddClass(comp layout.Component, class string) bool {
	if ti.styler == nil {
		return false
	}
	return ti.styler.addClass(comp, class)
}

// RemoveClass detaches a class from a component, reapplies styles, and reports whether it changed.
func (ti TemplateInstance) RemoveClass(comp layout.Component, class string) bool {
	if ti.styler == nil {
		return false
	}
	return ti.styler.removeClass(comp, class)
}

// HasClass reports whether the component currently owns the given class.
func (ti TemplateInstance) HasClass(comp layout.Component, class string) bool {
	if ti.styler == nil {
		return false
	}
	return ti.styler.hasClass(comp, class)
}

// Classes lists the classes associated with the component.
func (ti TemplateInstance) Classes(comp layout.Component) []string {
	if ti.styler == nil {
		return nil
	}
	return ti.styler.classes(comp)
}

func (l *Loader) spawnForTemplate(root *Node) *Loader {
	child := &Loader{
		Ctx:              l.Ctx,
		Reg:              l.Reg,
		Options:          l.Options,
		byID:             make(map[string]layout.Component),
		rootNode:         root,
		componentToNode:  make(map[layout.Component]*Node),
		nodeBindings:     make(map[*Node]*binding),
		imageCache:       l.imageCache,
		imageFailures:    l.imageFailures,
		resolveImagePath: l.resolveImagePath,
	}
	return child
}
