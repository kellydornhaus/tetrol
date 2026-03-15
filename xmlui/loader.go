package xmlui

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/kellydornhaus/layouter/layout"
)

// Options control parsing behavior.
type Options struct {
	IgnoreUnknown    bool
	Styles           *Stylesheet
	Classes          map[string][]string
	RootClasses      []string
	ImageLoader      ImageLoader
	ResolveImagePath func(ref string) (string, error)
}

// ImageLoader loads layout.Image resources for Image components and panel background images.
type ImageLoader interface {
	LoadImage(path string) (layout.Image, error)
}

// ImageLoaderFunc adapts a function into an ImageLoader.
type ImageLoaderFunc func(path string) (layout.Image, error)

// LoadImage implements ImageLoader.
func (f ImageLoaderFunc) LoadImage(path string) (layout.Image, error) { return f(path) }

// Result contains the built root component and a map of ids to components.
type Result struct {
	Root      layout.Component
	ByID      map[string]layout.Component
	Templates map[string]*Template
	styler    *styler
}

type styler struct {
	loader *Loader
}

// AddClass attaches a class to the provided component and reapplies styles when the class is new.
func (r Result) AddClass(comp layout.Component, class string) bool {
	if r.styler == nil {
		return false
	}
	return r.styler.addClass(comp, class)
}

// RemoveClass detaches a class from the component and reapplies styles when the class was present.
func (r Result) RemoveClass(comp layout.Component, class string) bool {
	if r.styler == nil {
		return false
	}
	return r.styler.removeClass(comp, class)
}

// HasClass reports whether the component currently owns the given class.
func (r Result) HasClass(comp layout.Component, class string) bool {
	if r.styler == nil {
		return false
	}
	return r.styler.hasClass(comp, class)
}

// Classes lists the classes associated with the component.
func (r Result) Classes(comp layout.Component) []string {
	if r.styler == nil {
		return nil
	}
	return r.styler.classes(comp)
}

// AddClassByID attaches a class to the component referenced by id.
func (r Result) AddClassByID(id, class string) bool {
	comp := r.ByID[id]
	if comp == nil {
		return false
	}
	return r.AddClass(comp, class)
}

// RemoveClassByID removes a class from the component referenced by id.
func (r Result) RemoveClassByID(id, class string) bool {
	comp := r.ByID[id]
	if comp == nil {
		return false
	}
	return r.RemoveClass(comp, class)
}

// HasClassByID reports whether the component referenced by id currently owns the class.
func (r Result) HasClassByID(id, class string) bool {
	return r.HasClass(r.ByID[id], class)
}

// ClassesByID lists classes for the component referenced by id.
func (r Result) ClassesByID(id string) []string {
	return r.Classes(r.ByID[id])
}

// Template returns the template registered with the provided id.
func (r Result) Template(id string) (*Template, bool) {
	if r.Templates == nil {
		return nil, false
	}
	tmpl, ok := r.Templates[id]
	return tmpl, ok
}

func (s *styler) addClass(comp layout.Component, class string) bool {
	if s == nil || s.loader == nil || comp == nil {
		return false
	}
	node := s.loader.componentToNode[comp]
	if node == nil {
		return false
	}
	if !node.AddClass(class) {
		return false
	}
	node.updateClassAttr()
	s.loader.restyleAll()
	return true
}

func (s *styler) removeClass(comp layout.Component, class string) bool {
	if s == nil || s.loader == nil || comp == nil {
		return false
	}
	node := s.loader.componentToNode[comp]
	if node == nil {
		return false
	}
	if !node.RemoveClass(class) {
		return false
	}
	node.updateClassAttr()
	s.loader.restyleAll()
	return true
}

func (s *styler) hasClass(comp layout.Component, class string) bool {
	if s == nil || s.loader == nil || comp == nil {
		return false
	}
	node := s.loader.componentToNode[comp]
	if node == nil {
		return false
	}
	return node.HasClass(class)
}

func (s *styler) classes(comp layout.Component) []string {
	if s == nil || s.loader == nil || comp == nil {
		return nil
	}
	node := s.loader.componentToNode[comp]
	if node == nil {
		return nil
	}
	classes := node.Classes()
	if len(classes) == 0 {
		return nil
	}
	out := make([]string, len(classes))
	copy(out, classes)
	return out
}

// Loader builds components from XML using a registry.
type Loader struct {
	Ctx     *layout.Context
	Reg     *Registry
	Options Options
	byID    map[string]layout.Component

	rootNode         *Node
	bindings         []*binding
	componentToNode  map[layout.Component]*Node
	nodeBindings     map[*Node]*binding
	imageCache       map[string]layout.Image
	imageFailures    map[string]struct{}
	resolveImagePath func(ref string) (string, error)
	templates        map[string]*Node
}

type binding struct {
	node      *Node
	component layout.Component
	update    func(*Loader, *Node, layout.Component)
}

func (l *Loader) trackBinding(n *Node, comp layout.Component, update func(*Loader, *Node, layout.Component)) {
	if n == nil || comp == nil || update == nil {
		return
	}
	b := &binding{node: n, component: comp, update: update}
	l.bindings = append(l.bindings, b)
	if l.componentToNode == nil {
		l.componentToNode = make(map[layout.Component]*Node)
	}
	l.componentToNode[comp] = n
	if l.nodeBindings == nil {
		l.nodeBindings = make(map[*Node]*binding)
	}
	l.nodeBindings[n] = b
}

func (l *Loader) restyleAll() {
	if l == nil || l.rootNode == nil {
		return
	}
	l.rootNode.resetRecursive()
	if l.Options.Styles != nil {
		applyStylesRecursive(l.rootNode, l.Options.Styles)
	}
	if l.componentToNode != nil {
		for comp, node := range l.componentToNode {
			if node != nil && comp != nil {
				l.applyVisibility(node, comp)
			}
		}
	}
	if l.componentToNode != nil && l.Ctx != nil && l.Ctx.DebugCSSEnabled() {
		for comp, node := range l.componentToNode {
			l.logCSSAttributes(node, comp)
		}
	}
	for _, b := range l.bindings {
		if b != nil && b.update != nil {
			b.update(l, b.node, b.component)
		}
	}
}

func (l *Loader) loadImage(ref string) layout.Image {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if l.resolveImagePath != nil {
		resolved, err := l.resolveImagePath(ref)
		if err != nil {
			log.Printf("xmlui: resolve image %q: %v", ref, err)
			return nil
		}
		if strings.TrimSpace(resolved) == "" {
			return nil
		}
		ref = resolved
	}
	if l.imageCache != nil {
		if img, ok := l.imageCache[ref]; ok {
			return img
		}
	}
	if l.imageFailures != nil {
		if _, failed := l.imageFailures[ref]; failed {
			return nil
		}
	}
	loader := l.Options.ImageLoader
	if loader == nil {
		log.Printf("xmlui: image reference %q requested but no ImageLoader provided", ref)
		if l.imageFailures != nil {
			l.imageFailures[ref] = struct{}{}
		}
		return nil
	}
	img, err := loader.LoadImage(ref)
	if err != nil {
		log.Printf("xmlui: load image %q: %v", ref, err)
		if l.imageFailures != nil {
			l.imageFailures[ref] = struct{}{}
		}
		return nil
	}
	if img == nil {
		if l.imageFailures != nil {
			l.imageFailures[ref] = struct{}{}
		}
		return nil
	}
	if l.imageCache != nil {
		l.imageCache[ref] = img
	}
	return img
}

func (l *Loader) logCSSAttributes(n *Node, comp layout.Component) {
	if l == nil || l.Ctx == nil || !l.Ctx.DebugCSSEnabled() || n == nil || comp == nil {
		return
	}
	if len(n.Attrs) == 0 {
		l.Ctx.DebugCSSf("node=%s component=%T id=%s class=%s attrs=<none>", n.Name, comp, strings.TrimSpace(n.Attrs["id"]), strings.TrimSpace(n.Attrs["class"]))
		return
	}
	keys := make([]string, 0, len(n.Attrs))
	for k := range n.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, strings.TrimSpace(n.Attrs[k])))
	}
	l.Ctx.DebugCSSf("node=%s component=%T id=%s class=%s attrs={%s}", n.Name, comp, strings.TrimSpace(n.Attrs["id"]), strings.TrimSpace(n.Attrs["class"]), strings.Join(pairs, ", "))
}

func buildFromRoot(ctx *layout.Context, rootNode *Node, reg *Registry, opts Options) (Result, error) {
	base := defaultRegistry()
	if reg != nil {
		for k, v := range reg.Snapshot() {
			base.Register(k, v)
		}
	}
	if len(opts.RootClasses) > 0 {
		changed := false
		for _, class := range opts.RootClasses {
			if rootNode.AddClass(class) {
				changed = true
			}
		}
		if changed {
			rootNode.updateClassAttr()
		}
	}
	if len(opts.Classes) > 0 {
		applyExtraClasses(rootNode, opts.Classes)
	}
	if opts.Styles != nil {
		applyStylesRecursive(rootNode, opts.Styles)
	}
	resolveImage := opts.ResolveImagePath
	if resolveImage == nil {
		resolveImage = func(ref string) (string, error) { return ref, nil }
	}
	l := &Loader{
		Ctx:              ctx,
		Reg:              base,
		Options:          opts,
		byID:             make(map[string]layout.Component),
		rootNode:         rootNode,
		componentToNode:  make(map[layout.Component]*Node),
		nodeBindings:     make(map[*Node]*binding),
		imageCache:       make(map[string]layout.Image),
		imageFailures:    make(map[string]struct{}),
		resolveImagePath: resolveImage,
	}
	comp, err := l.BuildNode(rootNode)
	if err != nil {
		return Result{}, err
	}
	var templates map[string]*Template
	if len(l.templates) > 0 {
		templates = make(map[string]*Template, len(l.templates))
		for id, node := range l.templates {
			templates[id] = &Template{id: id, node: node, loader: l}
		}
	}
	return Result{Root: comp, ByID: l.byID, Templates: templates, styler: &styler{loader: l}}, nil
}

// Build reads XML and constructs the component tree.
func Build(ctx *layout.Context, r io.Reader, reg *Registry, opts Options) (Result, error) {
	rootNode, err := parseXML(r)
	if err != nil {
		return Result{}, err
	}
	return buildFromRoot(ctx, rootNode, reg, opts)
}

// BuildFile reads an XML file from disk and constructs the component tree.
// It resolves any xml-stylesheet processing instructions relative to the XML file.
func BuildFile(ctx *layout.Context, filename string, reg *Registry, opts Options) (Result, error) {
	if strings.TrimSpace(filename) == "" {
		return Result{}, fmt.Errorf("xmlui: empty filename")
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return Result{}, fmt.Errorf("xmlui: resolve xml path %q: %w", filename, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return Result{}, fmt.Errorf("xmlui: read xml %q: %w", abs, err)
	}
	rootNode, styleRefs, err := parseXMLWithStyles(bytes.NewReader(data))
	if err != nil {
		return Result{}, err
	}

	if len(styleRefs) > 0 {
		loader := newStylesheetLoader(osCSSSource{})
		var rules []Rule
		for _, ref := range styleRefs {
			cssPath, err := osCSSSource{}.Resolve(abs, ref)
			if err != nil {
				return Result{}, fmt.Errorf("xmlui: resolve stylesheet %q from %q: %w", ref, abs, err)
			}
			ss, err := loader.parse(cssPath)
			if err != nil {
				return Result{}, fmt.Errorf("xmlui: parse stylesheet %q: %w", cssPath, err)
			}
			rules = append(rules, ss.Rules...)
		}
		opts.Styles = mergeStylesheets(opts.Styles, rules)
	}
	if opts.ResolveImagePath == nil {
		baseDir := filepath.Dir(abs)
		opts.ResolveImagePath = func(ref string) (string, error) {
			ref = strings.TrimSpace(ref)
			if ref == "" {
				return "", nil
			}
			if filepath.IsAbs(ref) {
				return filepath.Clean(ref), nil
			}
			return filepath.Clean(filepath.Join(baseDir, ref)), nil
		}
	}

	return buildFromRoot(ctx, rootNode, reg, opts)
}

// BuildFS reads an XML file from the provided filesystem and constructs the component tree.
// Any xml-stylesheet processing instructions are resolved relative to the XML file path.
func BuildFS(ctx *layout.Context, fsys fs.FS, name string, reg *Registry, opts Options) (Result, error) {
	if fsys == nil {
		return Result{}, fmt.Errorf("xmlui: nil filesystem")
	}
	src := fsCSSSource{fsys: fsys}
	cleanName, err := src.Canonical(name)
	if err != nil {
		return Result{}, fmt.Errorf("xmlui: resolve xml path %q: %w", name, err)
	}
	data, err := fs.ReadFile(fsys, cleanName)
	if err != nil {
		return Result{}, fmt.Errorf("xmlui: read xml %q: %w", cleanName, err)
	}
	rootNode, styleRefs, err := parseXMLWithStyles(bytes.NewReader(data))
	if err != nil {
		return Result{}, err
	}

	if len(styleRefs) > 0 {
		loader := newStylesheetLoader(src)
		var rules []Rule
		for _, ref := range styleRefs {
			cssPath, err := src.Resolve(cleanName, ref)
			if err != nil {
				return Result{}, fmt.Errorf("xmlui: resolve stylesheet %q from %q: %w", ref, cleanName, err)
			}
			ss, err := loader.parse(cssPath)
			if err != nil {
				return Result{}, fmt.Errorf("xmlui: parse stylesheet %q: %w", cssPath, err)
			}
			rules = append(rules, ss.Rules...)
		}
		opts.Styles = mergeStylesheets(opts.Styles, rules)
	}
	if opts.ResolveImagePath == nil {
		opts.ResolveImagePath = func(ref string) (string, error) {
			ref = strings.TrimSpace(ref)
			if ref == "" {
				return "", nil
			}
			return src.Resolve(cleanName, ref)
		}
	}

	return buildFromRoot(ctx, rootNode, reg, opts)
}

// BuildNode builds a component from a Node using the registry.
func (l *Loader) BuildNode(n *Node) (layout.Component, error) {
	if fn := l.Reg.Lookup(n.Name); fn != nil {
		c, err := fn(l, n)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n.Name, err)
		}
		l.applyVisibility(n, c)
		if l.componentToNode == nil {
			l.componentToNode = make(map[layout.Component]*Node)
		}
		l.componentToNode[c] = n
		if id := n.Attrs["id"]; id != "" {
			l.byID[id] = c
		}
		l.logCSSAttributes(n, c)
		return c, nil
	}
	if l.Options.IgnoreUnknown {
		return nil, nil
	}
	return nil, fmt.Errorf("unknown element: %s", n.Name)
}

// defaultRegistry returns a registry with built-in components.
func defaultRegistry() *Registry {
	r := NewRegistry()
	r.Register("VStack", buildVStack)
	r.Register("HStack", buildHStack)
	r.Register("FlowStack", buildFlowStack)
	r.Register("ZStack", buildZStack)
	r.Register("Grid", buildGrid)
	r.Register("Spacer", buildSpacer)
	r.Register("Panel", buildPanel)
	r.Register("Image", buildImage)
	return r
}

// Helpers to parse common attributes
func (l *Loader) parseFloat(val string, def float64) float64 {
	return ParseFloat(val, def)
}

func (l *Loader) parseBool(val string, def bool) bool {
	return ParseBool(val, def)
}

func (l *Loader) parseInsets(s string) *layout.EdgeInsets {
	if insets, ok := ParseInsets(s); ok {
		return &insets
	}
	return nil
}

func (l *Loader) parseLength(val string) (layout.Length, bool) {
	return parseLengthString(val)
}

func parseAlign(s string) layout.TextAlign {
	switch strings.ToLower(s) {
	case "center", "middle":
		return layout.AlignCenter
	case "end", "right", "bottom", "flex-end", "trailing":
		return layout.AlignEnd
	default:
		return layout.AlignStart
	}
}

func parseJustify(s string) layout.Justify {
	switch strings.ToLower(s) {
	case "center":
		return layout.JustifyCenter
	case "end", "right":
		return layout.JustifyEnd
	case "spacebetween", "space-between":
		return layout.JustifySpaceBetween
	default:
		return layout.JustifyStart
	}
}

func parseImageFit(s string) layout.ImageFit {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "contain":
		return layout.ImageFitContain
	case "center", "none":
		return layout.ImageFitCenter
	case "cover", "stretch", "fill":
		return layout.ImageFitStretch
	default:
		return layout.ImageFitStretch
	}
}

func parseImageReference(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "url(") && strings.HasSuffix(value, ")") {
		inner := strings.TrimSpace(value[4 : len(value)-1])
		value = inner
	}
	value = strings.Trim(value, "\"'")
	return strings.TrimSpace(value)
}

func parseSliceAttr(raw string) (layout.NineSlice, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return layout.NineSlice{}, false
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return layout.NineSlice{}, false
	}
	var slice layout.NineSlice
	values := []*int{&slice.Left, &slice.Top, &slice.Right, &slice.Bottom}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimSuffix(part, "dp")
		if part == "" {
			*values[i] = 0
			continue
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			return layout.NineSlice{}, false
		}
		*values[i] = val
	}
	return slice, true
}

func parseColorInternal(s string, def layout.Color) layout.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	if strings.EqualFold(s, "transparent") {
		return layout.Color{}
	}
	if c, ok := lookupNamedColor(s); ok {
		return c
	}
	if c, ok := parseFunctionalColor(s); ok {
		return c
	}
	if c, ok := parseHexColor(s); ok {
		return c
	}
	return def
}

func parseFunctionalColor(raw string) (layout.Color, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return layout.Color{}, false
	}
	lower := strings.ToLower(s)
	open := strings.IndexByte(lower, '(')
	if open == -1 || !strings.HasSuffix(lower, ")") {
		return layout.Color{}, false
	}
	mode := strings.TrimSpace(lower[:open])
	if mode != "rgb" && mode != "rgba" {
		return layout.Color{}, false
	}
	args := strings.TrimSpace(s[open+1 : len(s)-1])
	if args == "" {
		return layout.Color{}, false
	}
	var (
		rgbParts  []string
		alphaPart string
	)
	if strings.Contains(args, ",") {
		parts := strings.Split(args, ",")
		if len(parts) < 3 || len(parts) > 4 {
			return layout.Color{}, false
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
			if parts[i] == "" {
				return layout.Color{}, false
			}
		}
		rgbParts = parts[:3]
		if len(parts) == 4 {
			alphaPart = parts[3]
		}
	} else {
		slashParts := strings.Split(args, "/")
		if len(slashParts) > 2 {
			return layout.Color{}, false
		}
		rgbSegment := strings.TrimSpace(slashParts[0])
		if rgbSegment == "" {
			return layout.Color{}, false
		}
		rgbParts = strings.Fields(rgbSegment)
		if len(rgbParts) != 3 {
			return layout.Color{}, false
		}
		if len(slashParts) == 2 {
			alphaPart = strings.TrimSpace(slashParts[1])
			if alphaPart == "" {
				return layout.Color{}, false
			}
		}
	}
	for i := range rgbParts {
		rgbParts[i] = strings.TrimSpace(rgbParts[i])
		if rgbParts[i] == "" {
			return layout.Color{}, false
		}
	}
	r, ok := parseRGBComponent(rgbParts[0])
	if !ok {
		return layout.Color{}, false
	}
	g, ok := parseRGBComponent(rgbParts[1])
	if !ok {
		return layout.Color{}, false
	}
	b, ok := parseRGBComponent(rgbParts[2])
	if !ok {
		return layout.Color{}, false
	}
	alpha := 1.0
	if alphaPart != "" {
		var okAlpha bool
		alpha, okAlpha = parseAlphaComponent(alphaPart)
		if !okAlpha {
			return layout.Color{}, false
		}
	}
	return layout.Color{
		R: r,
		G: g,
		B: b,
		A: uint8(math.Round(clamp(alpha, 0, 1) * 255)),
	}, true
}

func parseRGBComponent(raw string) (uint8, bool) {
	if raw == "" {
		return 0, false
	}
	if strings.HasSuffix(raw, "%") {
		val := strings.TrimSpace(strings.TrimSuffix(raw, "%"))
		if val == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return uint8(math.Round(clamp(f, 0, 100) / 100 * 255)), true
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return uint8(math.Round(clamp(f, 0, 255))), true
}

func parseAlphaComponent(raw string) (float64, bool) {
	if raw == "" {
		return 0, false
	}
	if strings.HasSuffix(raw, "%") {
		val := strings.TrimSpace(strings.TrimSuffix(raw, "%"))
		if val == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return clamp(f, 0, 100) / 100, true
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return clamp(f, 0, 1), true
}

func clamp(val, min, max float64) float64 {
	return math.Min(math.Max(val, min), max)
}

func parseBorderSpec(l *Loader, val string) (color layout.Color, colorSet bool, width float64, widthSet bool) {
	replaced := strings.NewReplacer(",", " ", ";", " ").Replace(val)
	tokens := strings.Fields(replaced)
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		low := strings.ToLower(tok)
		switch low {
		case "none":
			width = 0
			widthSet = true
			color = layout.Color{}
			colorSet = true
			continue
		case "transparent":
			color = layout.Color{}
			colorSet = true
			continue
		case "solid":
			continue
		}
		if strings.HasPrefix(low, "#") || len(tok) == 6 || len(tok) == 8 || hasNamedColor(low) {
			color = parseColorInternal(tok, color)
			colorSet = true
			continue
		}
		if strings.ContainsAny(low, "0123456789") {
			width = l.parseFloat(tok, width)
			widthSet = true
			continue
		}
	}
	return color, colorSet, width, widthSet
}

func parseAspectRatio(val string) (float64, bool) {
	s := strings.TrimSpace(val)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, " ", "")
	if idx := strings.IndexAny(s, "/:"); idx >= 0 {
		num := strings.TrimSpace(s[:idx])
		den := strings.TrimSpace(s[idx+1:])
		if num == "" || den == "" {
			return 0, false
		}
		n, errN := strconv.ParseFloat(num, 64)
		d, errD := strconv.ParseFloat(den, 64)
		if errN != nil || errD != nil || n <= 0 || d <= 0 {
			return 0, false
		}
		return n / d, true
	}
	ratio, err := strconv.ParseFloat(s, 64)
	if err != nil || ratio <= 0 {
		return 0, false
	}
	return ratio, true
}

func (l *Loader) parseCornerRadiiAttr(raw string) (layout.CornerRadii, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return layout.CornerRadii{}, false
	}
	normalized := strings.ReplaceAll(raw, ",", " ")
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return layout.CornerRadii{}, false
	}
	vals := make([]float64, 0, len(fields))
	for _, f := range fields {
		v := l.parseFloat(f, 0)
		if v < 0 {
			v = 0
		}
		vals = append(vals, v)
	}
	var radii layout.CornerRadii
	switch len(vals) {
	case 1:
		radii = layout.CornerRadius(vals[0])
	case 2:
		radii.TopLeft = vals[0]
		radii.TopRight = vals[1]
		radii.BottomRight = vals[0]
		radii.BottomLeft = vals[1]
	case 3:
		radii.TopLeft = vals[0]
		radii.TopRight = vals[1]
		radii.BottomRight = vals[2]
		radii.BottomLeft = vals[1]
	default:
		radii.TopLeft = vals[0]
		radii.TopRight = vals[1]
		radii.BottomRight = vals[2]
		radii.BottomLeft = vals[3]
	}
	return radii, true
}

func parseCalcLength(raw string) (layout.Length, bool) {
	expr := strings.TrimSpace(raw)
	lower := strings.ToLower(expr)
	if !(strings.HasPrefix(lower, "calc(") || strings.HasPrefix(lower, "min(") || strings.HasPrefix(lower, "max(") || strings.HasPrefix(lower, "clamp(")) {
		return layout.Length{}, false
	}
	start := strings.Index(expr, "(") + 1
	depth := 1
	end := start
	for end < len(expr) && depth > 0 {
		switch expr[end] {
		case '(':
			depth++
		case ')':
			depth--
		}
		end++
	}
	if depth != 0 {
		log.Printf("xmlui: invalid functional length %q: mismatched parentheses", raw)
		return layout.Length{}, false
	}
	head := strings.TrimSpace(expr[:end])
	base, ok := parseCalcExpression(head)
	if !ok {
		log.Printf("xmlui: invalid functional length %q", raw)
		return layout.Length{}, false
	}
	tail := strings.TrimSpace(expr[end:])
	scale := 1.0
	for len(tail) > 0 {
		op := tail[0]
		if op != '*' && op != '/' {
			log.Printf("xmlui: invalid functional length scale %q", raw)
			return layout.Length{}, false
		}
		tail = strings.TrimSpace(tail[1:])
		if tail == "" {
			log.Printf("xmlui: invalid functional length scale %q", raw)
			return layout.Length{}, false
		}
		numStr, rest := readNumber(tail)
		if numStr == "" {
			log.Printf("xmlui: invalid functional length scale %q", raw)
			return layout.Length{}, false
		}
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil || math.IsNaN(num) || math.IsInf(num, 0) {
			log.Printf("xmlui: invalid functional length scale %q: %v", raw, err)
			return layout.Length{}, false
		}
		if op == '/' {
			if math.Abs(num) < 1e-9 {
				log.Printf("xmlui: invalid functional length scale %q: division by zero", raw)
				return layout.Length{}, false
			}
			scale /= num
		} else {
			scale *= num
		}
		tail = strings.TrimSpace(rest)
	}
	if base.ExprDefined() || scale != 1 {
		base.ScaleExpr(scale)
	}
	return base, true
}

func parseCalcExpression(expr string) (layout.Length, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return layout.Length{}, false
	}

	terms := make([]layout.LengthTerm, 0, 4)
	sign := 1.0
	for i := 0; i < len(expr); {
		r := rune(expr[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if r == '+' {
			sign = 1
			i++
			continue
		}
		if r == '-' {
			sign = -1
			i++
			continue
		}
		length, next, ok := parseCalcTerm(expr, i)
		if !ok {
			return layout.Length{}, false
		}
		terms = append(terms, layout.LengthTerm{Length: length, Weight: sign})
		i = next
		sign = 1
	}
	if len(terms) == 0 {
		return layout.Length{}, false
	}
	return layout.LengthSum(terms...), true
}

func parseCalcTerm(expr string, start int) (layout.Length, int, bool) {
	i := start
	for i < len(expr) && unicode.IsSpace(rune(expr[i])) {
		i++
	}
	if i >= len(expr) {
		return layout.Length{}, start, false
	}
	if unicode.IsLetter(rune(expr[i])) {
		if length, next, ok := parseCalcFunction(expr, i); ok {
			return length, next, true
		}
	}
	j := i
	dotSeen := false
	for j < len(expr) {
		r := expr[j]
		if r == '.' {
			if dotSeen {
				break
			}
			dotSeen = true
			j++
			continue
		}
		if !unicode.IsDigit(rune(r)) {
			break
		}
		j++
	}
	if j == i {
		return layout.Length{}, start, false
	}
	numStr := expr[i:j]
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return layout.Length{}, start, false
	}
	k := j
	for k < len(expr) && unicode.IsLetter(rune(expr[k])) {
		k++
	}
	unit := strings.ToLower(expr[j:k])
	if k < len(expr) && expr[k] == '%' {
		unit = "%"
		k++
	}
	for k < len(expr) && unicode.IsSpace(rune(expr[k])) {
		k++
	}
	switch unit {
	case "":
		if val >= 0 && val <= 1 {
			return layout.NewLengthComputed(0, val, 0, 0, 0, 0, 1), k, true
		}
		return layout.NewLengthComputed(val, 0, 0, 0, 0, 0, 1), k, true
	case "dp":
		return layout.NewLengthComputed(val, 0, 0, 0, 0, 0, 1), k, true
	case "%":
		return layout.NewLengthComputed(0, val/100, 0, 0, 0, 0, 1), k, true
	case "vw":
		if val > 1 {
			val = val / 100
		}
		return layout.NewLengthComputed(0, 0, val, 0, 0, 0, 1), k, true
	case "vh":
		if val > 1 {
			val = val / 100
		}
		return layout.NewLengthComputed(0, 0, 0, val, 0, 0, 1), k, true
	case "vmin":
		if val > 1 {
			val = val / 100
		}
		return layout.NewLengthComputed(0, 0, 0, 0, val, 0, 1), k, true
	case "vmax":
		if val > 1 {
			val = val / 100
		}
		return layout.NewLengthComputed(0, 0, 0, 0, 0, val, 1), k, true
	default:
		log.Printf("xmlui: unsupported unit %q in calc expression %q", unit, expr)
		return layout.Length{}, start, false
	}
}

func parseCalcFunction(expr string, start int) (layout.Length, int, bool) {
	j := start
	for j < len(expr) && unicode.IsLetter(rune(expr[j])) {
		j++
	}
	name := strings.ToLower(expr[start:j])
	k := j
	for k < len(expr) && unicode.IsSpace(rune(expr[k])) {
		k++
	}
	if k >= len(expr) || expr[k] != '(' {
		return layout.Length{}, start, false
	}
	depth := 1
	argStart := k + 1
	pos := k + 1
	args := make([]string, 0, 3)
	for pos < len(expr) && depth > 0 {
		switch expr[pos] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				arg := strings.TrimSpace(expr[argStart:pos])
				if arg != "" {
					args = append(args, arg)
				} else {
					return layout.Length{}, start, false
				}
				pos++
				goto parsed
			}
		case ',':
			if depth == 1 {
				arg := strings.TrimSpace(expr[argStart:pos])
				if arg == "" {
					return layout.Length{}, start, false
				}
				args = append(args, arg)
				argStart = pos + 1
			}
		}
		pos++
	}
	if depth != 0 {
		return layout.Length{}, start, false
	}
parsed:
	for pos < len(expr) && unicode.IsSpace(rune(expr[pos])) {
		pos++
	}
	switch name {
	case "calc":
		if len(args) != 1 {
			return layout.Length{}, start, false
		}
		length, ok := parseCalcExpression(args[0])
		if !ok {
			return layout.Length{}, start, false
		}
		return length, pos, true
	case "min":
		if len(args) == 0 {
			return layout.Length{}, start, false
		}
		lengths := make([]layout.Length, 0, len(args))
		for _, arg := range args {
			length, ok := parseCalcExpression(arg)
			if !ok {
				return layout.Length{}, start, false
			}
			lengths = append(lengths, length)
		}
		return layout.LengthMin(lengths...), pos, true
	case "max":
		if len(args) == 0 {
			return layout.Length{}, start, false
		}
		lengths := make([]layout.Length, 0, len(args))
		for _, arg := range args {
			length, ok := parseCalcExpression(arg)
			if !ok {
				return layout.Length{}, start, false
			}
			lengths = append(lengths, length)
		}
		return layout.LengthMax(lengths...), pos, true
	case "clamp":
		if len(args) != 3 {
			return layout.Length{}, start, false
		}
		minLen, okMin := parseCalcExpression(args[0])
		if !okMin {
			return layout.Length{}, start, false
		}
		valueLen, okValue := parseCalcExpression(args[1])
		if !okValue {
			return layout.Length{}, start, false
		}
		maxLen, okMax := parseCalcExpression(args[2])
		if !okMax {
			return layout.Length{}, start, false
		}
		return layout.LengthClamp(minLen, valueLen, maxLen), pos, true
	default:
		log.Printf("xmlui: unsupported function %q in calc expression %q", name, expr)
		return layout.Length{}, start, false
	}
}

func readNumber(s string) (string, string) {
	if s == "" {
		return "", ""
	}
	i := 0
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	dotSeen := false
	start := i
	for i < len(s) {
		switch s[i] {
		case '.':
			if dotSeen {
				return "", ""
			}
			dotSeen = true
			i++
		default:
			if !unicode.IsDigit(rune(s[i])) {
				goto done
			}
			i++
		}
	}
done:
	if i == start {
		return "", ""
	}
	num := strings.TrimSpace(s[:i])
	rest := strings.TrimSpace(s[i:])
	return num, rest
}

func (l *Loader) applyVisibility(n *Node, comp layout.Component) {
	if comp == nil {
		return
	}
	raw := strings.TrimSpace(n.Attrs["visibility"])
	if raw == "" {
		layout.SetVisibility(comp, layout.VisibilityVisible)
		return
	}
	if visibility, ok := layout.ParseVisibility(raw); ok {
		layout.SetVisibility(comp, visibility)
	}
}

func (l *Loader) applyVisibilityTransition(n *Node, comp layout.Component) {
	if comp == nil {
		return
	}
	raw := strings.TrimSpace(n.Attrs["visibility-transition"])
	if raw == "" {
		layout.SetVisibilityTransition(comp, layout.VisibilityTransition{})
		return
	}
	if transition, ok := layout.ParseVisibilityTransition(raw); ok {
		layout.SetVisibilityTransition(comp, transition)
	}
}

func parsePositionMode(raw string) (layout.PositionMode, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "relative":
		return layout.PositionRelative, true
	case "absolute":
		return layout.PositionAbsolute, true
	case "fixed":
		return layout.PositionFixed, true
	case "static", "":
		return layout.PositionStatic, true
	default:
		return layout.PositionStatic, false
	}
}

func parsePositionValue(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "auto") {
		return 0, false
	}
	lower := strings.ToLower(raw)
	if strings.HasSuffix(lower, "dp") {
		raw = strings.TrimSpace(raw[:len(raw)-2])
	}
	if strings.HasSuffix(lower, "px") {
		raw = strings.TrimSpace(raw[:len(raw)-2])
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func (l *Loader) resolvePositionOffsets(n *Node) layout.PositionOffsets {
	var offsets layout.PositionOffsets
	if val, ok := parsePositionValue(n.Attrs["top"]); ok {
		offsets.Top = layout.PositionValue{Value: val, Defined: true}
	}
	if val, ok := parsePositionValue(n.Attrs["right"]); ok {
		offsets.Right = layout.PositionValue{Value: val, Defined: true}
	}
	if val, ok := parsePositionValue(n.Attrs["bottom"]); ok {
		offsets.Bottom = layout.PositionValue{Value: val, Defined: true}
	}
	if val, ok := parsePositionValue(n.Attrs["left"]); ok {
		offsets.Left = layout.PositionValue{Value: val, Defined: true}
	}
	return offsets
}

func (l *Loader) applyPositionAttributes(n *Node, comp layout.Component) {
	if comp == nil {
		return
	}
	rawPos := strings.TrimSpace(n.Attrs["position"])
	if mode, ok := parsePositionMode(rawPos); ok {
		layout.SetPositionMode(comp, mode)
	} else {
		layout.SetPositionMode(comp, layout.PositionStatic)
	}
	offsets := l.resolvePositionOffsets(n)
	layout.SetPositionOffsets(comp, offsets)

	rawZ := strings.TrimSpace(n.Attrs["z-index"])
	switch {
	case rawZ == "", strings.EqualFold(rawZ, "auto"):
		layout.SetZIndex(comp, 0, false)
	default:
		if z, err := strconv.Atoi(rawZ); err == nil {
			layout.SetZIndex(comp, z, true)
		} else {
			layout.SetZIndex(comp, 0, false)
		}
	}
}

func (l *Loader) applyPanelAttributes(n *Node, comp layout.Component) {
	l.applyVisibility(n, comp)
	l.applyVisibilityTransition(n, comp)
	l.applyAlignmentAttributes(n, comp)

	rawNoCache := strings.TrimSpace(n.Attrs["nocache"])
	if rawNoCache != "" {
		layout.SetCacheEnabled(comp, !l.parseBool(rawNoCache, false))
	} else {
		layout.SetCacheEnabled(comp, true)
	}

	provider, ok := comp.(interface{ PanelRef() *layout.Panel })
	if !ok {
		l.applyPositionAttributes(n, comp)
		return
	}
	panel := provider.PanelRef()

	if pad := l.parseInsets(n.Attrs["padding"]); pad != nil {
		panel.SetPadding(*pad)
	}
	if v := strings.TrimSpace(n.Attrs["padding-top"]); v != "" {
		panel.SetPaddingTop(l.parseFloat(v, panel.Padding.Top))
	}
	if v := strings.TrimSpace(n.Attrs["padding-right"]); v != "" {
		panel.SetPaddingRight(l.parseFloat(v, panel.Padding.Right))
	}
	if v := strings.TrimSpace(n.Attrs["padding-bottom"]); v != "" {
		panel.SetPaddingBottom(l.parseFloat(v, panel.Padding.Bottom))
	}
	if v := strings.TrimSpace(n.Attrs["padding-left"]); v != "" {
		panel.SetPaddingLeft(l.parseFloat(v, panel.Padding.Left))
	}

	var bgColor layout.Color
	hasBackgroundAttr := false
	if bg := strings.TrimSpace(n.Attrs["background"]); bg != "" {
		bgColor = parseColorInternal(bg, layout.Color{})
		hasBackgroundAttr = true
	} else if bg := strings.TrimSpace(n.Attrs["background-color"]); bg != "" {
		bgColor = parseColorInternal(bg, layout.Color{})
		hasBackgroundAttr = true
	}
	if hasBackgroundAttr && bgColor.A > 0 {
		panel.SetBackgroundColor(bgColor)
	} else {
		panel.ClearBackgroundColor()
	}

	tintValue := strings.TrimSpace(n.Attrs["tint-color"])
	if tintValue == "" {
		tintValue = strings.TrimSpace(n.Attrs["tint"])
	}
	if tintValue != "" {
		tintColor := parseColorInternal(tintValue, layout.Color{})
		if tintColor.A > 0 {
			panel.SetTintColor(tintColor)
		} else {
			panel.ClearTintColor()
		}
	} else {
		panel.ClearTintColor()
	}

	bgImageRef := strings.TrimSpace(n.Attrs["background-image"])
	if bgImageRef != "" {
		if img := l.loadImage(parseImageReference(bgImageRef)); img != nil {
			paint := layout.ImagePaint{Image: img}
			if fit := strings.TrimSpace(n.Attrs["background-fit"]); fit != "" {
				paint.Fit = parseImageFit(fit)
			}
			if align := strings.TrimSpace(n.Attrs["background-align-h"]); align != "" {
				paint.AlignH = parseAlign(align)
			} else {
				paint.AlignH = layout.AlignCenter
			}
			if align := strings.TrimSpace(n.Attrs["background-align-v"]); align != "" {
				paint.AlignV = parseAlign(align)
			} else {
				paint.AlignV = layout.AlignCenter
			}
			if slice, ok := parseSliceAttr(n.Attrs["background-slice"]); ok {
				paint.Slice = slice
			}
			panel.SetBackgroundImage(paint)
		} else {
			panel.ClearBackgroundImage()
		}
	} else {
		panel.ClearBackgroundImage()
	}

	if ref := strings.TrimSpace(n.Attrs["background-image"]); ref != "" {
		if img := l.loadImage(parseImageReference(ref)); img != nil {
			paint := layout.ImagePaint{Image: img, AlignH: layout.AlignCenter, AlignV: layout.AlignCenter}
			if fit := strings.TrimSpace(n.Attrs["background-fit"]); fit != "" {
				paint.Fit = parseImageFit(fit)
			}
			if alignH := strings.TrimSpace(n.Attrs["background-align-h"]); alignH != "" {
				paint.AlignH = parseAlign(alignH)
			}
			if alignV := strings.TrimSpace(n.Attrs["background-align-v"]); alignV != "" {
				paint.AlignV = parseAlign(alignV)
			}
			panel.SetBackgroundImage(paint)
		} else {
			panel.ClearBackgroundImage()
		}
	} else {
		panel.ClearBackgroundImage()
	}

	var borderColor layout.Color
	borderColorSet := false
	var borderWidth float64
	borderWidthSet := false
	if b := strings.TrimSpace(n.Attrs["border"]); b != "" {
		if c, hasColor, w, hasWidth := parseBorderSpec(l, b); hasColor || hasWidth {
			if hasColor {
				borderColor = c
				borderColorSet = true
			}
			if hasWidth {
				borderWidth = w
				borderWidthSet = true
			}
		}
	}
	if b := strings.TrimSpace(n.Attrs["border-color"]); b != "" {
		borderColor = parseColorInternal(b, borderColor)
		borderColorSet = true
	}
	if b := strings.TrimSpace(n.Attrs["border-width"]); b != "" {
		borderWidth = l.parseFloat(b, borderWidth)
		borderWidthSet = true
	}
	if !borderColorSet && !borderWidthSet {
		panel.ClearBorder()
	} else {
		if !borderWidthSet {
			borderWidth = 1
		}
		panel.SetBorder(borderColor, borderWidth)
	}

	edges := []struct {
		attr      string
		widthAttr string
		colorAttr string
		setFull   func(float64, layout.Color)
		setWidth  func(float64)
		setColor  func(layout.Color)
		getWidth  func() float64
		getColor  func() layout.Color
	}{
		{
			attr:      "border-top",
			widthAttr: "border-top-width",
			colorAttr: "border-top-color",
			setFull:   panel.SetBorderTop,
			setWidth:  panel.SetBorderTopWidth,
			setColor:  panel.SetBorderTopColor,
			getWidth:  panel.BorderTopWidth,
			getColor:  panel.BorderTopColor,
		},
		{
			attr:      "border-right",
			widthAttr: "border-right-width",
			colorAttr: "border-right-color",
			setFull:   panel.SetBorderRight,
			setWidth:  panel.SetBorderRightWidth,
			setColor:  panel.SetBorderRightColor,
			getWidth:  panel.BorderRightWidth,
			getColor:  panel.BorderRightColor,
		},
		{
			attr:      "border-bottom",
			widthAttr: "border-bottom-width",
			colorAttr: "border-bottom-color",
			setFull:   panel.SetBorderBottom,
			setWidth:  panel.SetBorderBottomWidth,
			setColor:  panel.SetBorderBottomColor,
			getWidth:  panel.BorderBottomWidth,
			getColor:  panel.BorderBottomColor,
		},
		{
			attr:      "border-left",
			widthAttr: "border-left-width",
			colorAttr: "border-left-color",
			setFull:   panel.SetBorderLeft,
			setWidth:  panel.SetBorderLeftWidth,
			setColor:  panel.SetBorderLeftColor,
			getWidth:  panel.BorderLeftWidth,
			getColor:  panel.BorderLeftColor,
		},
	}
	for _, edge := range edges {
		if spec := strings.TrimSpace(n.Attrs[edge.attr]); spec != "" {
			if c, hasColor, w, hasWidth := parseBorderSpec(l, spec); hasColor || hasWidth {
				switch {
				case hasColor && hasWidth:
					edge.setFull(w, c)
				case hasWidth:
					edge.setWidth(w)
				case hasColor:
					edge.setColor(c)
				}
			}
		}
		if val := strings.TrimSpace(n.Attrs[edge.widthAttr]); val != "" {
			edge.setWidth(l.parseFloat(val, edge.getWidth()))
		}
		if colAttr := strings.TrimSpace(n.Attrs[edge.colorAttr]); colAttr != "" {
			edge.setColor(parseColorInternal(colAttr, edge.getColor()))
		}
	}

	text := strings.TrimSpace(n.Attrs["text"])
	if text == "" {
		text = strings.TrimSpace(n.Text)
	}
	panel.SetText(text)

	var style layout.TextStyle
	if v := strings.TrimSpace(n.Attrs["font"]); v != "" {
		style.FontKey = v
	}
	if raw := strings.TrimSpace(n.Attrs["font-size"]); raw != "" {
		style.SizeDp = l.parseFloat(raw, 0)
	}
	if v := strings.TrimSpace(n.Attrs["color"]); v != "" {
		style.Color = parseColorInternal(v, layout.Color{})
	}
	if v := strings.TrimSpace(n.Attrs["align-h"]); v != "" {
		style.AlignH = parseAlign(v)
	}
	if v := strings.TrimSpace(n.Attrs["align-v"]); v != "" {
		style.AlignV = parseAlign(v)
	}
	if v := strings.TrimSpace(n.Attrs["wrap"]); v != "" {
		style.Wrap = l.parseBool(v, false)
	}
	if v := strings.TrimSpace(n.Attrs["baseline-offset"]); v != "" {
		style.BaselineOffset = l.parseFloat(v, 0)
	}
	panel.SetTextStyle(style)

	panel.SetTextAutoFit(l.parseBool(strings.TrimSpace(n.Attrs["font-autosize"]), false))
	panel.SetTextAutoFitMin(l.parseFloat(strings.TrimSpace(n.Attrs["font-autosize-min"]), 0))
	panel.SetTextAutoFitMax(l.parseFloat(strings.TrimSpace(n.Attrs["font-autosize-max"]), 0))

	panel.SetFillWidth(l.parseBool(strings.TrimSpace(n.Attrs["fill-width"]), false))
	panel.SetFillHeight(l.parseBool(strings.TrimSpace(n.Attrs["fill-height"]), false))

	widthAttr := strings.TrimSpace(n.Attrs["width"])
	if widthAttr != "" {
		if length, okLen := l.parseLength(widthAttr); okLen {
			panel.SetWidthLength(length)
		} else {
			panel.SetWidthLength(layout.Length{})
		}
	} else {
		panel.SetWidthLength(layout.Length{})
	}
	heightAttr := strings.TrimSpace(n.Attrs["height"])
	if heightAttr != "" {
		if length, okLen := l.parseLength(heightAttr); okLen {
			panel.SetHeightLength(length)
		} else {
			panel.SetHeightLength(layout.Length{})
		}
	} else {
		panel.SetHeightLength(layout.Length{})
	}

	cornerAttr := strings.TrimSpace(n.Attrs["corner-radius"])
	var corner layout.CornerRadii
	cornerSet := false
	if cornerAttr != "" {
		if radii, ok := l.parseCornerRadiiAttr(cornerAttr); ok {
			corner = radii
			cornerSet = true
		}
	}
	applyCorner := func(attr string, apply func(float64)) {
		if val := strings.TrimSpace(n.Attrs[attr]); val != "" {
			r := l.parseFloat(val, 0)
			if r < 0 {
				r = 0
			}
			apply(r)
			cornerSet = true
		}
	}
	applyCorner("corner-top-left-radius", func(v float64) { corner.TopLeft = v })
	applyCorner("corner-top-right-radius", func(v float64) { corner.TopRight = v })
	applyCorner("corner-bottom-right-radius", func(v float64) { corner.BottomRight = v })
	applyCorner("corner-bottom-left-radius", func(v float64) { corner.BottomLeft = v })
	if cornerSet {
		panel.SetCornerRadii(corner)
	} else {
		panel.ClearCornerRadius()
	}

	if raw := strings.TrimSpace(n.Attrs["aspect-ratio"]); raw != "" {
		if ratio, ok := parseAspectRatio(raw); ok {
			panel.SetAspectRatio(ratio)
		} else {
			panel.SetAspectRatio(0)
		}
	} else {
		panel.SetAspectRatio(0)
	}

	if raw := strings.TrimSpace(n.Attrs["weight"]); raw != "" {
		if weight := l.parseFloat(raw, 0); weight > 0 {
			layout.SetFlexWeight(comp, weight)
		} else {
			layout.SetFlexWeight(comp, 0)
		}
	} else {
		layout.SetFlexWeight(comp, 0)
	}

	if v := strings.TrimSpace(n.Attrs["max-width"]); v != "" {
		if length, okLen := l.parseLength(v); okLen {
			panel.SetMaxWidthLength(length)
			if length.Unit == layout.LengthUnitDP && length.Value > 0 {
				panel.SetTextMaxWidth(length.Value)
			} else {
				panel.SetTextMaxWidth(0)
			}
		} else {
			panel.SetMaxWidthLength(layout.Length{})
			panel.SetTextMaxWidth(0)
		}
	} else if widthAttr == "" {
		panel.SetMaxWidthLength(layout.Length{})
		panel.SetTextMaxWidth(0)
	}
	if v := strings.TrimSpace(n.Attrs["max-height"]); v != "" {
		if length, okLen := l.parseLength(v); okLen {
			panel.SetMaxHeightLength(length)
		} else {
			panel.SetMaxHeightLength(layout.Length{})
		}
	} else if heightAttr == "" {
		panel.SetMaxHeightLength(layout.Length{})
	}
	if v := strings.TrimSpace(n.Attrs["min-width"]); v != "" {
		if length, okLen := l.parseLength(v); okLen {
			panel.SetMinWidthLength(length)
		} else {
			panel.SetMinWidthLength(layout.Length{})
		}
	} else if widthAttr == "" {
		panel.SetMinWidthLength(layout.Length{})
	}
	if v := strings.TrimSpace(n.Attrs["min-height"]); v != "" {
		if length, okLen := l.parseLength(v); okLen {
			panel.SetMinHeightLength(length)
		} else {
			panel.SetMinHeightLength(layout.Length{})
		}
	} else if heightAttr == "" {
		panel.SetMinHeightLength(layout.Length{})
	}

	l.applyPositionAttributes(n, comp)
}

func (l *Loader) applyAlignmentAttributes(n *Node, comp layout.Component) {
	if comp == nil {
		return
	}
	raw := strings.ToLower(strings.TrimSpace(n.Attrs["align-self"]))
	if raw == "" || raw == "auto" || raw == "stretch" {
		layout.SetAlignSelf(comp, layout.AlignStart, false)
		return
	}
	switch raw {
	case "center", "middle":
		layout.SetAlignSelf(comp, layout.AlignCenter, true)
	case "end", "flex-end", "bottom", "trailing":
		layout.SetAlignSelf(comp, layout.AlignEnd, true)
	case "start", "flex-start", "top", "leading":
		layout.SetAlignSelf(comp, layout.AlignStart, true)
	default:
		layout.SetAlignSelf(comp, layout.AlignStart, false)
	}
}

// Builders for built-in types
func buildVStack(l *Loader, n *Node) (layout.Component, error) {
	children, err := l.BuildChildren(n)
	if err != nil {
		return nil, err
	}
	v := layout.NewVStack(children...)
	l.updateVStackAttributes(n, v)
	l.trackBinding(n, v, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.VStack); ok {
			loader.updateVStackAttributes(node, actual)
		}
	})
	return v, nil
}

func buildHStack(l *Loader, n *Node) (layout.Component, error) {
	children, err := l.BuildChildren(n)
	if err != nil {
		return nil, err
	}
	h := layout.NewHStack(children...)
	l.updateHStackAttributes(n, h)
	l.trackBinding(n, h, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.HStack); ok {
			loader.updateHStackAttributes(node, actual)
		}
	})
	return h, nil
}

func buildFlowStack(l *Loader, n *Node) (layout.Component, error) {
	children, err := l.BuildChildren(n)
	if err != nil {
		return nil, err
	}
	f := layout.NewFlowStack(children...)
	l.updateFlowStackAttributes(n, f)
	l.trackBinding(n, f, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.FlowStack); ok {
			loader.updateFlowStackAttributes(node, actual)
		}
	})
	return f, nil
}

func buildSpacer(l *Loader, n *Node) (layout.Component, error) {
	w := l.parseFloat(n.Attrs["weight"], 1)
	sp := layout.NewSpacer(w)
	l.updateSpacerAttributes(n, sp)
	l.trackBinding(n, sp, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.Spacer); ok {
			loader.updateSpacerAttributes(node, actual)
		}
	})
	return sp, nil
}

func buildPanel(l *Loader, n *Node) (layout.Component, error) {
	children, err := l.BuildChildren(n)
	if err != nil {
		return nil, err
	}
	var child layout.Component
	switch len(children) {
	case 0:
		child = nil
	case 1:
		child = children[0]
	default:
		child = layout.NewVStack(children...)
	}
	panel := layout.NewPanelComponent(child)
	l.updatePanelComponentAttributes(n, panel)
	l.trackBinding(n, panel, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.PanelComponent); ok {
			loader.updatePanelComponentAttributes(node, actual)
		}
	})
	return panel, nil
}

func buildZStack(l *Loader, n *Node) (layout.Component, error) {
	children, err := l.BuildChildren(n)
	if err != nil {
		return nil, err
	}
	z := layout.NewZStack(children...)
	l.updateZStackAttributes(n, z)
	l.trackBinding(n, z, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.ZStack); ok {
			loader.updateZStackAttributes(node, actual)
		}
	})
	return z, nil
}

func buildGrid(l *Loader, n *Node) (layout.Component, error) {
	rawCols := strings.TrimSpace(n.Attrs["columns"])
	columns := 0
	if rawCols != "" && !strings.EqualFold(rawCols, "auto") {
		if val, err := strconv.Atoi(rawCols); err == nil {
			columns = val
		}
	}

	spacing := l.parseFloat(n.Attrs["spacing"], 20)
	grid := layout.NewGrid(columns, spacing)
	l.updateGridAttributes(n, grid)
	children, err := l.BuildChildren(n)
	if err != nil {
		return nil, err
	}
	for _, ch := range children {
		if ch != nil {
			grid.Add(ch)
		}
	}
	l.trackBinding(n, grid, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.Grid); ok {
			loader.updateGridAttributes(node, actual)
		}
	})
	return grid, nil
}

func buildImage(l *Loader, n *Node) (layout.Component, error) {
	if len(n.Children) > 0 {
		return nil, fmt.Errorf("Image cannot contain child elements")
	}
	img := layout.NewImage(nil)
	l.updateImageAttributes(n, img)
	l.trackBinding(n, img, func(loader *Loader, node *Node, comp layout.Component) {
		if actual, ok := comp.(*layout.ImageComponent); ok {
			loader.updateImageAttributes(node, actual)
		}
	})
	return img, nil
}

func (l *Loader) updateVStackAttributes(n *Node, v *layout.VStack) {
	spacing := l.parseFloat(n.Attrs["spacing"], 0)
	if math.Abs(v.Spacing-spacing) > 1e-6 {
		v.Spacing = spacing
		v.SetDirty()
	}
	alignH := parseAlign(n.Attrs["align-h"])
	if v.AlignH != alignH {
		v.AlignH = alignH
		v.SetDirty()
	}
	justify := parseJustify(n.Attrs["justify"])
	if v.Justify != justify {
		v.Justify = justify
		v.SetDirty()
	}
	l.applyPanelAttributes(n, v)
}

func (l *Loader) updateHStackAttributes(n *Node, h *layout.HStack) {
	spacing := l.parseFloat(n.Attrs["spacing"], 0)
	if math.Abs(h.Spacing-spacing) > 1e-6 {
		h.Spacing = spacing
		h.SetDirty()
	}
	alignV := parseAlign(n.Attrs["align-v"])
	if h.AlignV != alignV {
		h.AlignV = alignV
		h.SetDirty()
	}
	justify := parseJustify(n.Attrs["justify"])
	if h.Justify != justify {
		h.Justify = justify
		h.SetDirty()
	}
	l.applyPanelAttributes(n, h)
}

func (l *Loader) updateFlowStackAttributes(n *Node, f *layout.FlowStack) {
	spacing := l.parseFloat(n.Attrs["spacing"], 0)
	if spacing < 0 {
		spacing = 0
	}
	if math.Abs(f.Spacing-spacing) > 1e-6 {
		f.Spacing = spacing
		f.SetDirty()
	}
	lineSpacing := l.parseFloat(n.Attrs["line-spacing"], 0)
	if lineSpacing < 0 {
		lineSpacing = 0
	}
	if math.Abs(f.LineSpacing-lineSpacing) > 1e-6 {
		f.LineSpacing = lineSpacing
		f.SetDirty()
	}
	if v := strings.TrimSpace(n.Attrs["align-items"]); v != "" {
		align := parseAlign(v)
		if f.AlignItems != align {
			f.AlignItems = align
			f.SetDirty()
		}
	}
	if v := strings.TrimSpace(n.Attrs["align-content"]); v != "" {
		align := parseAlign(v)
		if f.AlignContent != align {
			f.AlignContent = align
			f.SetDirty()
		}
	}
	if v := strings.TrimSpace(n.Attrs["justify"]); v != "" {
		just := parseJustify(v)
		if f.Justify != just {
			f.Justify = just
			f.SetDirty()
		}
	}
	l.applyPanelAttributes(n, f)
}

func (l *Loader) updatePanelComponentAttributes(n *Node, panel *layout.PanelComponent) {
	l.applyPanelAttributes(n, panel)
}

func (l *Loader) updateZStackAttributes(n *Node, z *layout.ZStack) {
	l.applyPanelAttributes(n, z)
}

func (l *Loader) updateSpacerAttributes(n *Node, sp *layout.Spacer) {
	l.applyVisibility(n, sp)
	l.applyAlignmentAttributes(n, sp)
	l.applyPositionAttributes(n, sp)
	weight := l.parseFloat(n.Attrs["weight"], 1)
	layout.SetFlexWeight(sp, weight)
}

func (l *Loader) updateGridAttributes(n *Node, grid *layout.Grid) {
	rawCols := strings.TrimSpace(n.Attrs["columns"])
	columns := 0
	if rawCols != "" && !strings.EqualFold(rawCols, "auto") {
		if val, err := strconv.Atoi(rawCols); err == nil {
			columns = val
		}
	}
	grid.SetColumns(columns)

	spacing := l.parseFloat(n.Attrs["spacing"], 20)
	grid.SetSpacing(spacing)

	alignH := layout.AlignCenter
	if v := strings.TrimSpace(n.Attrs["align-h"]); v != "" {
		alignH = parseAlign(v)
	}
	if grid.AlignH != alignH {
		grid.AlignH = alignH
		grid.SetDirty()
	}

	alignV := layout.AlignCenter
	if v := strings.TrimSpace(n.Attrs["align-v"]); v != "" {
		alignV = parseAlign(v)
	}
	if grid.AlignV != alignV {
		grid.AlignV = alignV
		grid.SetDirty()
	}

	rowAlign := layout.AlignStart
	if v := strings.TrimSpace(n.Attrs["row-align"]); v != "" {
		rowAlign = parseAlign(v)
	}
	if grid.RowAlign != rowAlign {
		grid.RowAlign = rowAlign
		grid.SetDirty()
	}

	if v := strings.TrimSpace(n.Attrs["cell-min-width"]); v != "" {
		if length, ok := l.parseLength(v); ok {
			grid.SetCellMinWidthLength(length)
		}
	} else {
		grid.SetCellMinWidthLength(layout.Length{})
	}
	if v := strings.TrimSpace(n.Attrs["cell-max-width"]); v != "" {
		if length, ok := l.parseLength(v); ok {
			grid.SetCellMaxWidthLength(length)
		}
	} else {
		grid.SetCellMaxWidthLength(layout.Length{})
	}

	l.applyPanelAttributes(n, grid)
}

func (l *Loader) updateImageAttributes(n *Node, img *layout.ImageComponent) {
	if src := strings.TrimSpace(n.Attrs["src"]); src != "" {
		if image := l.loadImage(parseImageReference(src)); image != nil {
			img.SetSource(image)
		} else {
			img.SetSource(nil)
		}
	} else {
		img.SetSource(nil)
	}

	img.SetFit(parseImageFit(n.Attrs["fit"]))

	size := layout.Size{}
	widthAttr := strings.TrimSpace(n.Attrs["width"])
	heightAttr := strings.TrimSpace(n.Attrs["height"])
	widthAuto := strings.EqualFold(widthAttr, "auto")
	heightAuto := strings.EqualFold(heightAttr, "auto")
	if widthAttr != "" && !widthAuto {
		size.W = l.parseFloat(widthAttr, 0)
	}
	if heightAttr != "" && !heightAuto {
		size.H = l.parseFloat(heightAttr, 0)
	}
	img.SetExplicitSizeAuto(size, widthAuto, heightAuto)

	if v := strings.TrimSpace(n.Attrs["max-width"]); v != "" {
		if length, ok := l.parseLength(v); ok {
			img.SetMaxWidthLength(length)
		}
	} else {
		img.SetMaxWidthLength(layout.Length{})
	}
	if v := strings.TrimSpace(n.Attrs["max-height"]); v != "" {
		if length, ok := l.parseLength(v); ok {
			img.SetMaxHeightLength(length)
		}
	} else {
		img.SetMaxHeightLength(layout.Length{})
	}

	img.SetAlignment(parseAlign(n.Attrs["align-h"]), parseAlign(n.Attrs["align-v"]))
	l.applyPanelAttributes(n, img)
}

func (l *Loader) registerTemplateNode(n *Node) error {
	if n == nil {
		return nil
	}
	id := strings.TrimSpace(n.Attrs["id"])
	if id == "" {
		return fmt.Errorf("xmlui: template missing id attribute")
	}
	if l.templates == nil {
		l.templates = make(map[string]*Node)
	}
	if _, exists := l.templates[id]; exists {
		return fmt.Errorf("xmlui: duplicate template id %q", id)
	}
	cloned := cloneNode(n)
	if cloned != nil {
		cloned.Parent = nil
		cloned.resetRecursive()
	}
	l.templates[id] = cloned
	return nil
}

// BuildChildren builds all child nodes and returns the resulting components.
func (l *Loader) BuildChildren(n *Node) ([]layout.Component, error) {
	var out []layout.Component
	for _, ch := range n.Children {
		if strings.EqualFold(ch.Name, "template") {
			if err := l.registerTemplateNode(ch); err != nil {
				return nil, err
			}
			continue
		}
		c, err := l.BuildNode(ch)
		if err != nil {
			return nil, err
		}
		if c != nil {
			if weightAttr := ch.Attrs["weight"]; weightAttr != "" {
				if weight := l.parseFloat(weightAttr, 0); weight > 0 {
					layout.SetFlexWeight(c, weight)
				}
			}
			out = append(out, c)
		}
	}
	return out, nil
}

func mergeStylesheets(existing *Stylesheet, additions []Rule) *Stylesheet {
	if len(additions) == 0 {
		return existing
	}
	additionalCopy := cloneRules(additions)
	if existing == nil {
		return &Stylesheet{Rules: additionalCopy}
	}
	merged := make([]Rule, 0, len(existing.Rules)+len(additionalCopy))
	merged = append(merged, cloneRules(existing.Rules)...)
	merged = append(merged, additionalCopy...)
	return &Stylesheet{Rules: merged}
}

func applyStylesRecursive(n *Node, ss *Stylesheet) {
	applyStylesRecursiveWithAncestors(n, ss, nil)
}

func applyStylesRecursiveWithAncestors(n *Node, ss *Stylesheet, ancestors []*Node) {
	applyStylesWithAncestors(n, ss, ancestors)
	for _, ch := range n.Children {
		applyStylesRecursiveWithAncestors(ch, ss, append(ancestors, n))
	}
}

func applyExtraClasses(n *Node, classes map[string][]string) {
	if n == nil || len(classes) == 0 {
		return
	}
	if id := n.Attrs["id"]; id != "" {
		if extra, ok := classes[id]; ok && len(extra) > 0 {
			changed := false
			for _, class := range extra {
				if n.AddClass(class) {
					changed = true
				}
			}
			if changed {
				n.updateClassAttr()
			}
		}
	}
	for _, ch := range n.Children {
		applyExtraClasses(ch, classes)
	}
}
