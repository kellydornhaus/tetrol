package xmlui

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Stylesheet is a minimal CSS subset with simple selectors and declarations.
// Supported selectors:
//   - Tag:    Text, VStack, HStack, Box, Spacer, or custom tag names
//   - Class:  .class (matches when element has class in its class attribute)
//   - ID:     #id (matches when element has matching id)
//   - Combined: Tag.class, Tag#id, .a.b (all classes must be present)
//   - Descendant: .mode Panel (matches when ancestors satisfy preceding parts)
//
// Multiple selectors can be separated by commas.
//
// Supported properties map directly to XML attributes with minor renames:
//
//	spacing, padding, justify, align-h, align-v, size, color, wrap,
//	max-width, weight, visibility, visibility-transition
//
// Conflicts are resolved by specificity (id > class > tag) and source order
// (last rule wins when specificity ties). Inline style has higher precedence
// than stylesheet rules; explicit XML attributes have the highest precedence.
type Stylesheet struct {
	Rules []Rule
}

type Rule struct {
	Source    string
	Selectors []selector
	Decls     map[string]string
}

type simpleSelector struct {
	Tag     string
	ID      string
	Classes []string
}

func (s simpleSelector) specificity() int {
	id := 0
	if s.ID != "" {
		id = 1
	}
	return id*100 + len(s.Classes)*10 + btoi(s.Tag != "")
}

type selector struct {
	Parts []simpleSelector
}

func (s selector) specificity() int {
	total := 0
	for _, part := range s.Parts {
		total += part.specificity()
	}
	return total
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ParseStylesheet parses a tiny CSS subset from r.
func ParseStylesheet(r io.Reader) (*Stylesheet, error) {
	css, err := preprocessCSSReader(r)
	if err != nil {
		return nil, err
	}
	_, cleaned := extractImports(css)
	return &Stylesheet{Rules: parseRules(cleaned)}, nil
}

// ParseStylesheetFile parses a CSS file from disk, resolving @import directives.
func ParseStylesheetFile(filename string) (*Stylesheet, error) {
	loader := newStylesheetLoader(osCSSSource{})
	ss, err := loader.parse(filename)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

// ParseStylesheetFS parses a CSS file located in fsys, resolving @import directives.
func ParseStylesheetFS(fsys fs.FS, name string) (*Stylesheet, error) {
	loader := newStylesheetLoader(fsCSSSource{fsys: fsys})
	ss, err := loader.parse(name)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

func preprocessCSSReader(r io.Reader) (string, error) {
	var sb strings.Builder
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scan.Scan() {
		line := scan.Text()
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	if err := scan.Err(); err != nil {
		return "", err
	}
	return removeBlockComments(sb.String()), nil
}

func preprocessCSSString(s string) string {
	res, err := preprocessCSSReader(strings.NewReader(s))
	if err != nil {
		return s
	}
	return res
}

func removeBlockComments(css string) string {
	for {
		start := strings.Index(css, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(css[start+2:], "*/")
		if end < 0 {
			css = css[:start]
			break
		}
		css = css[:start] + css[start+2+end+2:]
	}
	return css
}

func extractImports(css string) ([]string, string) {
	matches := cssImportPattern.FindAllStringSubmatchIndex(css, -1)
	if len(matches) == 0 {
		return nil, css
	}
	imports := make([]string, 0, len(matches))
	var b strings.Builder
	prev := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		pathStart, pathEnd := m[2], m[3]
		if start > prev {
			b.WriteString(css[prev:start])
		}
		if pathStart >= 0 && pathEnd >= 0 {
			imports = append(imports, strings.TrimSpace(css[pathStart:pathEnd]))
		}
		prev = end
	}
	if prev < len(css) {
		b.WriteString(css[prev:])
	}
	return imports, b.String()
}

func parseRules(css string) []Rule {
	var rules []Rule
	for _, chunk := range strings.Split(css, "}") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.SplitN(chunk, "{", 2)
		if len(parts) != 2 {
			continue
		}
		selPart := strings.TrimSpace(parts[0])
		declPart := strings.TrimSpace(parts[1])
		sels := parseSelectorList(selPart)
		decls := parseDecls(declPart)
		if len(sels) == 0 || len(decls) == 0 {
			continue
		}
		rules = append(rules, Rule{Selectors: sels, Decls: decls})
	}
	return rules
}

type cssSource interface {
	Read(path string) ([]byte, error)
	Canonical(path string) (string, error)
	Resolve(currentPath, importPath string) (string, error)
}

type osCSSSource struct{}

func (osCSSSource) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osCSSSource) Canonical(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty css path")
	}
	return filepath.Abs(path)
}

func (osCSSSource) Resolve(currentPath, importPath string) (string, error) {
	if importPath == "" {
		return "", fmt.Errorf("empty css import path")
	}
	if filepath.IsAbs(importPath) {
		return filepath.Clean(importPath), nil
	}
	baseDir := filepath.Dir(currentPath)
	resolved := filepath.Join(baseDir, importPath)
	return filepath.Clean(resolved), nil
}

type fsCSSSource struct {
	fsys fs.FS
}

func (s fsCSSSource) Read(path string) ([]byte, error) {
	return fs.ReadFile(s.fsys, path)
}

func (s fsCSSSource) Canonical(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("empty css path")
	}
	clean := path.Clean(strings.TrimPrefix(p, "./"))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid css path %q", p)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("css import escapes fs root: %q", p)
	}
	return clean, nil
}

func (s fsCSSSource) Resolve(currentPath, importPath string) (string, error) {
	if strings.TrimSpace(importPath) == "" {
		return "", fmt.Errorf("empty css import path")
	}
	target := strings.TrimSpace(importPath)
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else {
		dir := path.Dir(currentPath)
		if dir != "." {
			target = path.Join(dir, target)
		}
	}
	clean := path.Clean(target)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid css import path %q", importPath)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("css import escapes fs root: %q", importPath)
	}
	return clean, nil
}

type stylesheetLoader struct {
	src   cssSource
	cache map[string][]Rule
}

func newStylesheetLoader(src cssSource) *stylesheetLoader {
	return &stylesheetLoader{
		src:   src,
		cache: make(map[string][]Rule),
	}
}

func (l *stylesheetLoader) parse(name string) (*Stylesheet, error) {
	rules, err := l.parseRecursive(name, make(map[string]bool))
	if err != nil {
		return nil, err
	}
	return &Stylesheet{Rules: rules}, nil
}

func (l *stylesheetLoader) parseRecursive(name string, stack map[string]bool) ([]Rule, error) {
	canon, err := l.src.Canonical(name)
	if err != nil {
		return nil, err
	}
	if stack[canon] {
		return nil, fmt.Errorf("css import cycle detected: %s", name)
	}
	if cached, ok := l.cache[canon]; ok {
		return cloneRules(cached), nil
	}

	stack[canon] = true
	defer delete(stack, canon)

	data, err := l.src.Read(canon)
	if err != nil {
		return nil, fmt.Errorf("read css %s: %w", name, err)
	}

	css := preprocessCSSString(string(data))
	imports, cleaned := extractImports(css)

	var combined []Rule
	for _, imp := range imports {
		resolved, err := l.src.Resolve(canon, imp)
		if err != nil {
			return nil, fmt.Errorf("resolve css import %q from %q: %w", imp, name, err)
		}
		rules, err := l.parseRecursive(resolved, stack)
		if err != nil {
			return nil, err
		}
		combined = append(combined, rules...)
	}

	own := parseRules(cleaned)
	if len(own) > 0 {
		for i := range own {
			own[i].Source = canon
		}
		combined = append(combined, own...)
	}

	l.cache[canon] = cloneRules(combined)
	return cloneRules(combined), nil
}

func cloneRules(rules []Rule) []Rule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]Rule, len(rules))
	for i, r := range rules {
		out[i].Source = r.Source
		if len(r.Selectors) > 0 {
			out[i].Selectors = make([]selector, len(r.Selectors))
			for j, sel := range r.Selectors {
				out[i].Selectors[j] = cloneSelector(sel)
			}
		}
		if len(r.Decls) > 0 {
			decls := make(map[string]string, len(r.Decls))
			for k, v := range r.Decls {
				decls[k] = v
			}
			out[i].Decls = decls
		} else if r.Decls != nil {
			out[i].Decls = make(map[string]string)
		}
	}
	return out
}

func cloneSelector(sel selector) selector {
	if len(sel.Parts) == 0 {
		return selector{}
	}
	out := selector{Parts: make([]simpleSelector, len(sel.Parts))}
	for i, part := range sel.Parts {
		cloned := simpleSelector{Tag: part.Tag, ID: part.ID}
		if len(part.Classes) > 0 {
			cloned.Classes = append([]string(nil), part.Classes...)
		}
		out.Parts[i] = cloned
	}
	return out
}

func parseSelectorList(s string) []selector {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]selector, 0, len(parts))
	for _, p := range parts {
		if sel, ok := parseSelector(strings.TrimSpace(p)); ok {
			out = append(out, sel)
		}
	}
	return out
}

func parseSelector(s string) (selector, bool) {
	var chain []simpleSelector
	if s == "" {
		return selector{}, false
	}
	i := 0
	cur := simpleSelector{}
	flush := func() {
		if cur.Tag == "" && cur.ID == "" && len(cur.Classes) == 0 {
			return
		}
		chain = append(chain, cur)
		cur = simpleSelector{}
	}
	for i < len(s) {
		switch s[i] {
		case '.':
			i++
			start := i
			for i < len(s) && isIdentChar(s[i]) {
				i++
			}
			if start < i {
				cur.Classes = append(cur.Classes, s[start:i])
			}
		case ' ', '\t', '\n', '\r':
			flush()
			i++
		case '#':
			i++
			start := i
			for i < len(s) && isIdentChar(s[i]) {
				i++
			}
			if start < i {
				if cur.ID == "" {
					cur.ID = s[start:i]
				}
			}
		default:
			// tag
			start := i
			for i < len(s) && isIdentChar(s[i]) {
				i++
			}
			if start < i {
				if cur.Tag == "" {
					cur.Tag = s[start:i]
				}
			}
		}
	}
	flush()
	if len(chain) == 0 {
		return selector{}, false
	}
	return selector{Parts: chain}, true
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_'
}

func parseDecls(s string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(s, ";") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(strings.ToLower(kv[0]))
		v := strings.TrimSpace(kv[1])
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

// applyStyles merges CSS and inline style into node attributes following precedence.
// Precedence: explicit XML attribute > inline style > stylesheet (#id > .class > tag, last wins).
func applyStyles(n *Node, ss *Stylesheet) {
	applyStylesWithAncestors(n, ss, nil)
}

func applyStylesWithAncestors(n *Node, ss *Stylesheet, ancestors []*Node) {
	if ss == nil {
		return
	}
	// Compute best declarations for this node
	var bestSpec map[string]int
	var values map[string]string
	bestSpec = make(map[string]int)
	values = make(map[string]string)
	classes := splitClasses(n.Attrs["class"])

	for ri, rule := range ss.Rules {
		for _, sel := range rule.Selectors {
			if selectorMatches(sel, n, classes, ancestors) {
				spec := sel.specificity()*100000 + ri // tie-break by source order
				for prop, val := range rule.Decls {
					if cur, ok := bestSpec[prop]; !ok || spec >= cur {
						bestSpec[prop] = spec
						values[prop] = val
					}
				}
			}
		}
	}

	// Inline style has higher precedence than stylesheet
	if inline := n.Attrs["style"]; inline != "" {
		for k, v := range parseDecls(inline) {
			values[k] = v
			bestSpec[k] = 1 << 30
		}
	}

	// Merge into attributes if not set explicitly
	for prop, val := range values {
		key := cssPropToAttr(prop)
		if key == "" {
			continue
		}
		if _, exists := n.Attrs[key]; !exists {
			n.Attrs[key] = val
		}
	}
}

func selectorMatches(sel selector, node *Node, nodeClasses map[string]bool, ancestors []*Node) bool {
	if len(sel.Parts) == 0 {
		return false
	}
	last := sel.Parts[len(sel.Parts)-1]
	if !simpleSelectorMatches(last, node, nodeClasses) {
		return false
	}
	if len(sel.Parts) == 1 {
		return true
	}
	idx := len(ancestors) - 1
	for i := len(sel.Parts) - 2; i >= 0; i-- {
		part := sel.Parts[i]
		found := false
		for idx >= 0 {
			if simpleSelectorMatches(part, ancestors[idx], nil) {
				found = true
				idx--
				break
			}
			idx--
		}
		if !found {
			return false
		}
	}
	return true
}

func simpleSelectorMatches(sel simpleSelector, node *Node, cachedClasses map[string]bool) bool {
	if sel.Tag != "" && !strings.EqualFold(sel.Tag, node.Name) {
		return false
	}
	if sel.ID != "" && sel.ID != node.Attrs["id"] {
		return false
	}
	if len(sel.Classes) == 0 {
		return true
	}
	classes := cachedClasses
	if classes == nil {
		classes = splitClasses(node.Attrs["class"])
	}
	for _, c := range sel.Classes {
		if !classes[c] {
			return false
		}
	}
	return true
}

func splitClasses(s string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range strings.Fields(s) {
		if tok != "" {
			out[tok] = true
		}
	}
	return out
}

func cssPropToAttr(p string) string {
	switch strings.ToLower(p) {
	case "align-h":
		return "align-h"
	case "align-v":
		return "align-v"
	case "align-self":
		return "align-self"
	case "align-items":
		return "align-items"
	case "align-content":
		return "align-content"
	case "justify":
		return "justify"
	case "max-width":
		return "max-width"
	case "max-height":
		return "max-height"
	case "cell-max-width":
		return "cell-max-width"
	case "cell-min-width":
		return "cell-min-width"
	case "spacing":
		return "spacing"
	case "line-spacing":
		return "line-spacing"
	case "fill-width":
		return "fill-width"
	case "fill-height":
		return "fill-height"
	case "row-align":
		return "row-align"
	case "background-color":
		return "background-color"
	case "border-color":
		return "border-color"
	case "border-width":
		return "border-width"
	case "border":
		return "border"
	case "border-top":
		return "border-top"
	case "border-right":
		return "border-right"
	case "border-bottom":
		return "border-bottom"
	case "border-left":
		return "border-left"
	case "border-top-width":
		return "border-top-width"
	case "border-right-width":
		return "border-right-width"
	case "border-bottom-width":
		return "border-bottom-width"
	case "border-left-width":
		return "border-left-width"
	case "border-top-color":
		return "border-top-color"
	case "border-right-color":
		return "border-right-color"
	case "border-bottom-color":
		return "border-bottom-color"
	case "border-left-color":
		return "border-left-color"
	case "padding-top":
		return "padding-top"
	case "padding-right":
		return "padding-right"
	case "padding-bottom":
		return "padding-bottom"
	case "padding-left":
		return "padding-left"
	case "border-radius":
		return "corner-radius"
	case "corner-radius":
		return "corner-radius"
	case "border-top-left-radius":
		return "corner-top-left-radius"
	case "border-top-right-radius":
		return "corner-top-right-radius"
	case "border-bottom-right-radius":
		return "corner-bottom-right-radius"
	case "border-bottom-left-radius":
		return "corner-bottom-left-radius"
	case "corner-top-left-radius":
		return "corner-top-left-radius"
	case "corner-top-right-radius":
		return "corner-top-right-radius"
	case "corner-bottom-right-radius":
		return "corner-bottom-right-radius"
	case "corner-bottom-left-radius":
		return "corner-bottom-left-radius"
	case "width":
		return "width"
	case "height":
		return "height"
	case "aspect-ratio":
		return "aspect-ratio"
	case "background-image":
		return "background-image"
	case "background-fit":
		return "background-fit"
	case "background-align-h":
		return "background-align-h"
	case "background-align-v":
		return "background-align-v"
	case "tint", "tint-color":
		return "tint-color"
	case "font-autosize", "font-auto-size":
		return "font-autosize"
	case "font-autosize-min", "font-auto-size-min":
		return "font-autosize-min"
	case "font-autosize-max", "font-auto-size-max":
		return "font-autosize-max"
	case "src":
		return "src"
	case "visibility":
		return "visibility"
	case "visibility-transition":
		return "visibility-transition"
	case "nocache":
		return "nocache"
	case "size":
		return ""
	case "fontsize":
		return ""
	default:
		return p
	}
}

var cssImportPattern = regexp.MustCompile(`(?i)@import\s+(?:url\()?["']?([^"')\s;]+)["']?\)?\s*;`)
