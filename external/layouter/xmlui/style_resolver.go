package xmlui

import (
	"strings"

	"github.com/kellydornhaus/layouter/layout"
)

// StyleResolver evaluates stylesheet rules without building an XML tree.
type StyleResolver struct {
	sheet *Stylesheet
}

// ResolveOptions customize resolution, mirroring XML precedence (attributes > inline style > CSS).
type ResolveOptions struct {
	// Attributes simulate explicit XML attributes (highest precedence).
	Attributes map[string]string
	// InlineStyle simulates a style="..." attribute (higher precedence than stylesheet rules).
	InlineStyle string
	// Ancestors describe ancestor chain from root to parent, enabling descendant selectors.
	Ancestors []StyleSelector
}

// StyleSelector describes a synthetic ancestor element.
type StyleSelector struct {
	Tag         string
	ID          string
	Classes     []string
	Attributes  map[string]string
	InlineStyle string
}

func (s StyleSelector) toNode() *Node {
	attrs := make(map[string]string)
	if len(s.Attributes) > 0 {
		for k, v := range s.Attributes {
			key := strings.ToLower(strings.TrimSpace(k))
			if key == "" || v == "" {
				continue
			}
			attrs[key] = v
		}
	}
	if s.ID != "" {
		attrs["id"] = s.ID
	}
	if strings.TrimSpace(s.InlineStyle) != "" {
		attrs["style"] = strings.TrimSpace(s.InlineStyle)
	}
	attrs["class"] = mergeClasses(attrs["class"], s.Classes)
	return finalizeSyntheticNode(s.Tag, attrs)
}

// ResolvedStyle exposes computed attributes and typed helpers.
type ResolvedStyle struct {
	values map[string]string
}

// NewStyleResolver creates a resolver; nil sheets yield empty results.
func NewStyleResolver(sheet *Stylesheet) *StyleResolver {
	return &StyleResolver{sheet: sheet}
}

// Resolve computes the merged attribute map for the provided element, accepting a CSS selector
// (e.g., "Panel.tile#active"). For convenience, whitespace-separated selectors ("Panel .tile")
// are treated as descendants, with the last simple selector representing the node being resolved
// and preceding selectors treated as ancestors.
func (r *StyleResolver) Resolve(selector string, opts *ResolveOptions) ResolvedStyle {
	values := make(map[string]string)
	if r == nil || r.sheet == nil {
		return ResolvedStyle{values: values}
	}

	tag, id, classes, extraAncestors := parseSelectorForResolve(selector)
	combinedOpts := mergeResolveOptions(opts, extraAncestors)
	node := buildSyntheticNode(tag, id, classes, combinedOpts)
	ancestors := buildAncestorNodes(combinedOpts)
	applyStylesWithAncestors(node, r.sheet, ancestors)

	for k, v := range node.Attrs {
		if k == "id" || k == "class" || k == "style" {
			continue
		}
		values[k] = v
	}
	return ResolvedStyle{values: values}
}

// Values returns a defensive copy of all resolved attributes.
func (s ResolvedStyle) Values() map[string]string {
	if len(s.values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out
}

// Raw fetches the raw attribute value.
func (s ResolvedStyle) Raw(name string) (string, bool) {
	v, ok := s.values[name]
	return v, ok
}

// Has reports whether an attribute is present.
func (s ResolvedStyle) Has(name string) bool {
	_, ok := s.values[name]
	return ok
}

// String returns the attribute string or fallback.
func (s ResolvedStyle) String(name, fallback string) string {
	if v, ok := s.values[name]; ok && v != "" {
		return v
	}
	return fallback
}

// Float parses the attribute as float64 (dp-aware), returning fallback when missing.
func (s ResolvedStyle) Float(name string, fallback float64) float64 {
	if v, ok := s.values[name]; ok {
		return ParseFloat(v, fallback)
	}
	return fallback
}

// Bool parses a boolean attribute.
func (s ResolvedStyle) Bool(name string, fallback bool) bool {
	if v, ok := s.values[name]; ok {
		return ParseBool(v, fallback)
	}
	return fallback
}

// Color parses a color attribute, honoring CSS names and rgba() forms.
func (s ResolvedStyle) Color(name string, fallback layout.Color) layout.Color {
	if v, ok := s.values[name]; ok {
		return ParseColor(v, fallback)
	}
	return fallback
}

// Insets parses an EdgeInsets attribute.
func (s ResolvedStyle) Insets(name string) (layout.EdgeInsets, bool) {
	if v, ok := s.values[name]; ok {
		return ParseInsets(v)
	}
	return layout.EdgeInsets{}, false
}

// Length parses a layout.Length attribute.
func (s ResolvedStyle) Length(name string) (layout.Length, bool) {
	if v, ok := s.values[name]; ok {
		return ParseLength(v)
	}
	return layout.Length{}, false
}

// helper builders ---------------------------------------------------------

func buildSyntheticNode(tag, id string, classes []string, opts *ResolveOptions) *Node {
	attrs := make(map[string]string)
	if opts != nil && len(opts.Attributes) > 0 {
		for k, v := range opts.Attributes {
			key := strings.ToLower(strings.TrimSpace(k))
			if key == "" {
				continue
			}
			attrs[key] = v
		}
	}
	if id != "" {
		attrs["id"] = id
	}
	if opts != nil && strings.TrimSpace(opts.InlineStyle) != "" {
		attrs["style"] = strings.TrimSpace(opts.InlineStyle)
	}
	attrs["class"] = mergeClasses(attrs["class"], classes)
	return finalizeSyntheticNode(tag, attrs)
}

func buildAncestorNodes(opts *ResolveOptions) []*Node {
	if opts == nil || len(opts.Ancestors) == 0 {
		return nil
	}
	ancestors := make([]*Node, 0, len(opts.Ancestors))
	for _, sel := range opts.Ancestors {
		ancestors = append(ancestors, sel.toNode())
	}
	return ancestors
}

func mergeResolveOptions(opts *ResolveOptions, extras []StyleSelector) *ResolveOptions {
	if len(extras) == 0 {
		return opts
	}
	clonedExtras := cloneStyleSelectors(extras)
	if opts == nil {
		return &ResolveOptions{Ancestors: clonedExtras}
	}
	copyOpts := *opts
	if len(copyOpts.Ancestors) > 0 {
		copyOpts.Ancestors = append(cloneStyleSelectors(copyOpts.Ancestors), clonedExtras...)
	} else {
		copyOpts.Ancestors = clonedExtras
	}
	return &copyOpts
}

func cloneStyleSelectors(src []StyleSelector) []StyleSelector {
	if len(src) == 0 {
		return nil
	}
	out := make([]StyleSelector, len(src))
	for i, sel := range src {
		out[i] = cloneStyleSelector(sel)
	}
	return out
}

func cloneStyleSelector(sel StyleSelector) StyleSelector {
	cloned := StyleSelector{
		Tag:         sel.Tag,
		ID:          sel.ID,
		InlineStyle: sel.InlineStyle,
	}
	if len(sel.Classes) > 0 {
		cloned.Classes = append([]string(nil), sel.Classes...)
	}
	if len(sel.Attributes) > 0 {
		attrs := make(map[string]string, len(sel.Attributes))
		for k, v := range sel.Attributes {
			attrs[k] = v
		}
		cloned.Attributes = attrs
	}
	return cloned
}

func parseSelectorForResolve(raw string) (tag, id string, classes []string, ancestors []StyleSelector) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "*", "", nil, nil
	}
	sel, ok := parseSelector(raw)
	if !ok || len(sel.Parts) == 0 {
		return "*", "", nil, nil
	}
	last := sel.Parts[len(sel.Parts)-1]
	tag = last.Tag
	if tag == "" {
		tag = "*"
	}
	id = last.ID
	if len(last.Classes) > 0 {
		classes = append([]string(nil), last.Classes...)
	}
	if len(sel.Parts) > 1 {
		ancestors = make([]StyleSelector, 0, len(sel.Parts)-1)
		for _, part := range sel.Parts[:len(sel.Parts)-1] {
			ancestors = append(ancestors, simpleSelectorToStyleSelector(part))
		}
	}
	return tag, id, classes, ancestors
}

func simpleSelectorToStyleSelector(part simpleSelector) StyleSelector {
	return StyleSelector{
		Tag:     part.Tag,
		ID:      part.ID,
		Classes: append([]string(nil), part.Classes...),
	}
}

func finalizeSyntheticNode(tag string, attrs map[string]string) *Node {
	n := &Node{Name: strings.TrimSpace(tag)}
	n.Attrs = make(map[string]string)
	for k, v := range attrs {
		if v == "" {
			continue
		}
		n.Attrs[k] = v
	}
	n.baseAttrs = cloneAttrs(n.Attrs)
	n.initClassSet()
	return n
}

func mergeClasses(existing string, extras []string) string {
	var tokens []string
	if existing != "" {
		tokens = append(tokens, strings.Fields(existing)...)
	}
	if len(extras) > 0 {
		tokens = append(tokens, extras...)
	}
	if len(tokens) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(tokens))
	unique := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		trim := strings.TrimSpace(tok)
		if trim == "" {
			continue
		}
		if _, ok := seen[trim]; ok {
			continue
		}
		seen[trim] = struct{}{}
		unique = append(unique, trim)
	}
	if len(unique) == 0 {
		return ""
	}
	return strings.Join(unique, " ")
}
