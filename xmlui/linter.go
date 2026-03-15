package xmlui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IssueKind describes a linter finding category.
type IssueKind string

const (
	IssueUnusedXMLAttr IssueKind = "unused-xml-attr"
	IssueUnusedCSSProp IssueKind = "unused-css-property"
	IssueXMLShadowsCSS IssueKind = "xml-shadows-css"
)

// LintIssue captures a single lint finding.
type LintIssue struct {
	Kind      IssueKind
	NodePath  string
	Attribute string
	Property  string
	Selector  string
	Source    string
	Message   string
}

// LintReport is the result of linting an XML/CSS document.
type LintReport struct {
	Issues []LintIssue
}

// Lint walks the parsed XML node tree and stylesheet to flag unused/overridden properties.
func Lint(root *Node, sheet *Stylesheet) LintReport {
	if root == nil {
		return LintReport{}
	}
	known := knownAttributes()
	declUsage, declMeta := gatherCSSDecls(sheet)
	var issues []LintIssue
	lintNode(root, nil, sheet, known, declUsage, declMeta, &issues)
	for key, meta := range declMeta {
		if meta.attr == "" || !known[meta.attr] {
			continue
		}
		if declUsage[key] == 0 {
			issues = append(issues, LintIssue{
				Kind:      IssueUnusedCSSProp,
				Attribute: meta.attr,
				Property:  meta.property,
				Selector:  key.selector,
				Source:    key.source,
				Message:   fmt.Sprintf("CSS %q (%s) sets %q but it does not apply to any element", key.selector, key.source, meta.property),
			})
		}
	}
	sortIssues(issues)
	return LintReport{Issues: issues}
}

// LintFile parses an XML file (plus any linked/extra CSS) and runs the linter.
type LintFileOptions struct {
	ExtraCSS []string
}

func LintFile(filename string, opts LintFileOptions) (LintReport, error) {
	if strings.TrimSpace(filename) == "" {
		return LintReport{}, fmt.Errorf("xmlui: lint: empty filename")
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return LintReport{}, fmt.Errorf("xmlui: lint: resolve xml path %q: %w", filename, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return LintReport{}, fmt.Errorf("xmlui: lint: read xml %q: %w", abs, err)
	}
	root, styleRefs, err := parseXMLWithStyles(strings.NewReader(string(data)))
	if err != nil {
		return LintReport{}, err
	}

	var rules []Rule
	if len(styleRefs) > 0 || len(opts.ExtraCSS) > 0 {
		loader := newStylesheetLoader(osCSSSource{})
		for _, ref := range styleRefs {
			cssPath, err := osCSSSource{}.Resolve(abs, ref)
			if err != nil {
				return LintReport{}, fmt.Errorf("xmlui: lint: resolve stylesheet %q from %q: %w", ref, abs, err)
			}
			ss, err := loader.parse(cssPath)
			if err != nil {
				return LintReport{}, fmt.Errorf("xmlui: lint: parse stylesheet %q: %w", cssPath, err)
			}
			rules = append(rules, ss.Rules...)
		}
		baseDir := filepath.Dir(abs)
		for _, extra := range opts.ExtraCSS {
			target := extra
			if !filepath.IsAbs(target) {
				target = filepath.Join(baseDir, target)
			}
			ss, err := loader.parse(target)
			if err != nil {
				return LintReport{}, fmt.Errorf("xmlui: lint: parse extra stylesheet %q: %w", target, err)
			}
			rules = append(rules, ss.Rules...)
		}
	}
	var sheet *Stylesheet
	if len(rules) > 0 {
		sheet = &Stylesheet{Rules: rules}
	}
	return Lint(root, sheet), nil
}

func lintNode(n *Node, ancestors []*Node, sheet *Stylesheet, known map[string]bool, declUsage map[cssDeclKey]int, declMeta map[cssDeclKey]cssDeclMeta, issues *[]LintIssue) {
	if n == nil {
		return
	}
	base := n.baseAttrs
	if len(base) == 0 {
		base = n.Attrs
	}
	for attr := range base {
		if !known[attr] {
			*issues = append(*issues, LintIssue{
				Kind:      IssueUnusedXMLAttr,
				NodePath:  nodePath(n),
				Attribute: attr,
				Message:   fmt.Sprintf("%s: attribute %q is not recognized", nodePath(n), attr),
			})
		}
	}

	matches := cssMatches(n, ancestors, sheet)
	for _, m := range matches {
		if m.attr == "" || !known[m.attr] {
			*issues = append(*issues, LintIssue{
				Kind:      IssueUnusedCSSProp,
				NodePath:  nodePath(n),
				Property:  m.property,
				Attribute: m.attr,
				Selector:  m.selector,
				Source:    m.source,
				Message:   fmt.Sprintf("%s: CSS %q (%s) sets %q which is not a known attribute", nodePath(n), m.selector, m.source, m.property),
			})
			continue
		}
		if _, exists := base[m.attr]; exists {
			*issues = append(*issues, LintIssue{
				Kind:      IssueXMLShadowsCSS,
				NodePath:  nodePath(n),
				Attribute: m.attr,
				Property:  m.property,
				Selector:  m.selector,
				Source:    m.source,
				Message:   fmt.Sprintf("%s: XML sets %q, CSS %q (%s) also sets it so the CSS is ignored", nodePath(n), m.attr, m.selector, m.source),
			})
			continue
		}
		if m.declKey != nil {
			if _, ok := declUsage[*m.declKey]; ok {
				declUsage[*m.declKey]++
			}
		}
	}

	for _, ch := range n.Children {
		lintNode(ch, append(ancestors, n), sheet, known, declUsage, declMeta, issues)
	}
}

type cssMatch struct {
	property string
	attr     string
	selector string
	source   string
	declKey  *cssDeclKey
	spec     int
}

func cssMatches(n *Node, ancestors []*Node, sheet *Stylesheet) []cssMatch {
	if sheet == nil {
		return nil
	}
	classes := splitClasses(n.Attrs["class"])
	best := make(map[string]cssMatch)
	for ri, rule := range sheet.Rules {
		for _, sel := range rule.Selectors {
			if selectorMatches(sel, n, classes, ancestors) {
				spec := sel.specificity()*100000 + ri
				for prop, val := range rule.Decls {
					propLower := strings.ToLower(prop)
					if cur, ok := best[propLower]; ok && cur.spec > spec {
						continue
					}
					attr := cssPropToAttr(propLower)
					key := cssDeclKey{source: rule.Source, selector: selectorString(sel), property: propLower}
					best[propLower] = cssMatch{
						property: propLower,
						attr:     attr,
						selector: selectorString(sel),
						source:   rule.Source,
						spec:     spec,
						declKey:  &key,
					}
					_ = val
				}
			}
		}
	}
	if inline := strings.TrimSpace(n.Attrs["style"]); inline != "" {
		for prop, val := range parseDecls(inline) {
			propLower := strings.ToLower(prop)
			attr := cssPropToAttr(propLower)
			best[propLower] = cssMatch{
				property: propLower,
				attr:     attr,
				selector: "inline style",
				source:   "inline",
				spec:     1 << 30,
			}
			_ = val
		}
	}
	out := make([]cssMatch, 0, len(best))
	for _, m := range best {
		out = append(out, m)
	}
	return out
}

type cssDeclKey struct {
	source   string
	selector string
	property string
}

type cssDeclMeta struct {
	attr     string
	property string
}

func gatherCSSDecls(sheet *Stylesheet) (map[cssDeclKey]int, map[cssDeclKey]cssDeclMeta) {
	if sheet == nil {
		return map[cssDeclKey]int{}, map[cssDeclKey]cssDeclMeta{}
	}
	usage := make(map[cssDeclKey]int)
	meta := make(map[cssDeclKey]cssDeclMeta)
	for _, rule := range sheet.Rules {
		for _, sel := range rule.Selectors {
			selText := selectorString(sel)
			for prop := range rule.Decls {
				propLower := strings.ToLower(prop)
				key := cssDeclKey{source: rule.Source, selector: selText, property: propLower}
				usage[key] = 0
				meta[key] = cssDeclMeta{attr: cssPropToAttr(propLower), property: propLower}
			}
		}
	}
	return usage, meta
}

func knownAttributes() map[string]bool {
	names := []string{
		"align-content", "align-h", "align-items", "align-self", "align-v",
		"aspect-ratio", "background", "background-align-h", "background-align-v",
		"background-color", "background-fit", "background-image", "background-slice",
		"baseline-offset", "border", "border-color", "border-width", "bottom",
		"cell-max-width", "cell-min-width", "class", "color", "columns",
		"corner-radius", "fill-height", "fill-width", "fit", "font",
		"font-autosize", "font-autosize-max", "font-autosize-min", "font-size",
		"height", "id", "justify", "left", "line-spacing", "max-height",
		"max-width", "min-height", "min-width", "padding", "padding-bottom",
		"padding-left", "padding-right", "padding-top", "position", "right",
		"row-align", "spacing", "src", "style", "text", "tint", "tint-color",
		"top", "visibility", "visibility-transition", "weight", "width", "wrap", "z-index",
		"rate", "radius", "gravity", "max",
	}
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

func nodeLabel(n *Node) string {
	if n == nil {
		return ""
	}
	label := n.Name
	if id := strings.TrimSpace(n.Attrs["id"]); id != "" {
		label += "#" + id
	}
	if cls := strings.TrimSpace(n.Attrs["class"]); cls != "" {
		parts := strings.Fields(cls)
		for _, c := range parts {
			label += "." + c
		}
	}
	return label
}

func nodePath(n *Node) string {
	if n == nil {
		return ""
	}
	var parts []string
	cur := n
	for cur != nil {
		parts = append(parts, nodeLabel(cur))
		cur = cur.Parent
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, " > ")
}

func selectorString(sel selector) string {
	var parts []string
	for _, p := range sel.Parts {
		var piece string
		if p.Tag != "" {
			piece = p.Tag
		}
		if p.ID != "" {
			piece += "#" + p.ID
		}
		for _, c := range p.Classes {
			if c != "" {
				piece += "." + c
			}
		}
		if piece == "" {
			piece = "*"
		}
		parts = append(parts, piece)
	}
	return strings.Join(parts, " ")
}

func sortIssues(issues []LintIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Kind != issues[j].Kind {
			return issues[i].Kind < issues[j].Kind
		}
		if issues[i].NodePath != issues[j].NodePath {
			return issues[i].NodePath < issues[j].NodePath
		}
		return issues[i].Message < issues[j].Message
	})
}
