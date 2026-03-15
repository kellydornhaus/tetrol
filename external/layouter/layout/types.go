package layout

import "math"

// LengthUnit enumerates supported logical length measurement units.
type LengthUnit int

const (
	LengthUnitAuto LengthUnit = iota
	LengthUnitDP
	LengthUnitPercent
	LengthUnitViewportWidth
	LengthUnitViewportHeight
	LengthUnitViewportMin
	LengthUnitViewportMax
)

// Length represents a logical size that can be expressed in different units.
type Length struct {
	Value float64
	Unit  LengthUnit
	expr  *lengthExpr
}

type lengthExpr struct {
	node lengthNode
}

type lengthNode interface {
	resolve(parentMax, viewportW, viewportH float64) float64
	scaleBy(factor float64)
	clone() lengthNode
}

type linearNode struct {
	dp      float64
	percent float64
	vw      float64
	vh      float64
	vmin    float64
	vmax    float64
	scale   float64
}

func (n *linearNode) resolve(parentMax, viewportW, viewportH float64) float64 {
	sum := n.dp
	if n.percent != 0 && parentMax > 0 {
		sum += parentMax * n.percent
	}
	if n.vw != 0 && viewportW > 0 {
		sum += viewportW * n.vw
	}
	if n.vh != 0 && viewportH > 0 {
		sum += viewportH * n.vh
	}
	if n.vmin != 0 {
		min := minPositive(viewportW, viewportH)
		if min > 0 {
			sum += min * n.vmin
		}
	}
	if n.vmax != 0 {
		max := maxPositive(viewportW, viewportH)
		if max > 0 {
			sum += max * n.vmax
		}
	}
	return sum * n.scale
}

func (n *linearNode) scaleBy(factor float64) {
	n.scale *= factor
}

func (n *linearNode) clone() lengthNode {
	if n == nil {
		return nil
	}
	cpy := *n
	return &cpy
}

type sumTerm struct {
	weight float64
	node   lengthNode
}

type sumNode struct {
	terms []sumTerm
	scale float64
}

func (n *sumNode) resolve(parentMax, viewportW, viewportH float64) float64 {
	if n == nil {
		return 0
	}
	total := 0.0
	for _, term := range n.terms {
		if term.node == nil || term.weight == 0 {
			continue
		}
		total += term.weight * term.node.resolve(parentMax, viewportW, viewportH)
	}
	return total * n.scale
}

func (n *sumNode) scaleBy(factor float64) {
	n.scale *= factor
}

func (n *sumNode) clone() lengthNode {
	if n == nil {
		return nil
	}
	cpy := &sumNode{
		scale: n.scale,
		terms: make([]sumTerm, len(n.terms)),
	}
	for i, term := range n.terms {
		cpy.terms[i] = sumTerm{
			weight: term.weight,
			node:   term.node.clone(),
		}
	}
	return cpy
}

type minNode struct {
	args  []lengthNode
	scale float64
}

func (n *minNode) resolve(parentMax, viewportW, viewportH float64) float64 {
	if n == nil || len(n.args) == 0 {
		return 0
	}
	minVal := n.args[0].resolve(parentMax, viewportW, viewportH)
	for i := 1; i < len(n.args); i++ {
		val := n.args[i].resolve(parentMax, viewportW, viewportH)
		if val < minVal {
			minVal = val
		}
	}
	return minVal * n.scale
}

func (n *minNode) scaleBy(factor float64) {
	n.scale *= factor
}

func (n *minNode) clone() lengthNode {
	if n == nil {
		return nil
	}
	cpy := &minNode{
		scale: n.scale,
		args:  make([]lengthNode, len(n.args)),
	}
	for i, arg := range n.args {
		cpy.args[i] = arg.clone()
	}
	return cpy
}

type maxNode struct {
	args  []lengthNode
	scale float64
}

func (n *maxNode) resolve(parentMax, viewportW, viewportH float64) float64 {
	if n == nil || len(n.args) == 0 {
		return 0
	}
	maxVal := n.args[0].resolve(parentMax, viewportW, viewportH)
	for i := 1; i < len(n.args); i++ {
		val := n.args[i].resolve(parentMax, viewportW, viewportH)
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal * n.scale
}

func (n *maxNode) scaleBy(factor float64) {
	n.scale *= factor
}

func (n *maxNode) clone() lengthNode {
	if n == nil {
		return nil
	}
	cpy := &maxNode{
		scale: n.scale,
		args:  make([]lengthNode, len(n.args)),
	}
	for i, arg := range n.args {
		cpy.args[i] = arg.clone()
	}
	return cpy
}

type clampNode struct {
	value lengthNode
	min   lengthNode
	max   lengthNode
	scale float64
}

func (n *clampNode) resolve(parentMax, viewportW, viewportH float64) float64 {
	if n == nil || n.value == nil {
		return 0
	}
	val := n.value.resolve(parentMax, viewportW, viewportH)
	lo := 0.0
	hi := math.Inf(1)
	if n.min != nil {
		lo = n.min.resolve(parentMax, viewportW, viewportH)
	}
	if n.max != nil {
		hi = n.max.resolve(parentMax, viewportW, viewportH)
	}
	if lo > hi {
		lo, hi = hi, lo
	}
	if val < lo {
		val = lo
	} else if val > hi {
		val = hi
	}
	return val * n.scale
}

func (n *clampNode) scaleBy(factor float64) {
	n.scale *= factor
}

func (n *clampNode) clone() lengthNode {
	if n == nil {
		return nil
	}
	return &clampNode{
		value: n.value.clone(),
		min:   n.min.clone(),
		max:   n.max.clone(),
		scale: n.scale,
	}
}

// LengthAuto returns an undefined length (use default behavior).
func LengthAuto() Length { return Length{} }

// LengthDP creates a dp-based length.
func LengthDP(v float64) Length {
	if v <= 0 {
		return Length{}
	}
	return Length{Value: v, Unit: LengthUnitDP}
}

// LengthPercent creates a parent-relative length (0-1 range recommended).
func LengthPercent(fraction float64) Length {
	if fraction <= 0 {
		return Length{}
	}
	return Length{Value: fraction, Unit: LengthUnitPercent}
}

// LengthVW creates a viewport-width-relative length (Value represents ratio).
func LengthVW(fraction float64) Length {
	if fraction <= 0 {
		return Length{}
	}
	return Length{Value: fraction, Unit: LengthUnitViewportWidth}
}

// LengthVH creates a viewport-height-relative length (Value represents ratio).
func LengthVH(fraction float64) Length {
	if fraction <= 0 {
		return Length{}
	}
	return Length{Value: fraction, Unit: LengthUnitViewportHeight}
}

// LengthVMin creates a viewport-min-relative length (min of width/height).
func LengthVMin(fraction float64) Length {
	if fraction <= 0 {
		return Length{}
	}
	return Length{Value: fraction, Unit: LengthUnitViewportMin}
}

// LengthVMax creates a viewport-max-relative length (max of width/height).
func LengthVMax(fraction float64) Length {
	if fraction <= 0 {
		return Length{}
	}
	return Length{Value: fraction, Unit: LengthUnitViewportMax}
}

// Defined reports whether this length applies a constraint.
func (l Length) Defined() bool {
	if l.expr != nil {
		return true
	}
	return l.Unit != LengthUnitAuto && l.Value > 0
}

func (l Length) resolve(parentMax float64, viewportW float64, viewportH float64) float64 {
	if l.expr != nil {
		val := l.expr.node.resolve(parentMax, viewportW, viewportH)
		if val < 0 {
			return 0
		}
		return val
	}
	if !l.Defined() {
		return 0
	}
	switch l.Unit {
	case LengthUnitDP:
		if l.Value < 0 {
			return 0
		}
		return l.Value
	case LengthUnitPercent:
		if parentMax <= 0 {
			return 0
		}
		return maxFloat(0, parentMax*l.Value)
	case LengthUnitViewportWidth:
		if viewportW <= 0 {
			return 0
		}
		return maxFloat(0, viewportW*l.Value)
	case LengthUnitViewportHeight:
		if viewportH <= 0 {
			return 0
		}
		return maxFloat(0, viewportH*l.Value)
	case LengthUnitViewportMin:
		v := minPositive(viewportW, viewportH)
		if v <= 0 {
			return 0
		}
		return maxFloat(0, v*l.Value)
	case LengthUnitViewportMax:
		v := maxPositive(viewportW, viewportH)
		if v <= 0 {
			return 0
		}
		return maxFloat(0, v*l.Value)
	default:
		return 0
	}
}

// ResolveWidth converts the stored length to dp units for width calculations.
func (l Length) ResolveWidth(ctx *Context, parentMax float64) float64 {
	var vw, vh float64
	if ctx != nil {
		vw = ctx.ViewportWidth()
		vh = ctx.ViewportHeight()
	}
	return l.resolve(parentMax, vw, vh)
}

func minPositive(a, b float64) float64 {
	min := 0.0
	if a > 0 {
		min = a
	}
	if b > 0 && (min == 0 || b < min) {
		min = b
	}
	return min
}

func maxPositive(a, b float64) float64 {
	max := 0.0
	if a > 0 {
		max = a
	}
	if b > max {
		max = b
	}
	return max
}

// ResolveHeight converts the stored length to dp units for height calculations.
func (l Length) ResolveHeight(ctx *Context, parentMax float64) float64 {
	var vw, vh float64
	if ctx != nil {
		vw = ctx.ViewportWidth()
		vh = ctx.ViewportHeight()
	}
	return l.resolve(parentMax, vw, vh)
}

// NewLengthComputed constructs a Length from linear combination of viewport/parent metrics.
// percent is applied to parent max (0-1). viewport units expect coefficients in fraction form (1 = 100%).
// scale is applied after summing contributions.
func NewLengthComputed(dp, percent, vw, vh, vmin, vmax, scale float64) Length {
	return Length{
		Unit: LengthUnitAuto,
		expr: &lengthExpr{
			node: &linearNode{
				dp:      dp,
				percent: percent,
				vw:      vw,
				vh:      vh,
				vmin:    vmin,
				vmax:    vmax,
				scale:   scale,
			},
		},
	}
}

// ExprDefined reports whether the length uses a computed expression.
func (l Length) ExprDefined() bool { return l.expr != nil }

// ScaleExpr scales the computed expression (no-op for fixed lengths).
func (l *Length) ScaleExpr(scale float64) {
	if scale == 0 {
		return
	}
	if l.expr != nil {
		l.expr.node.scaleBy(scale)
	} else {
		l.Value *= scale
	}
}

// LengthTerm represents a weighted length contribution used when building composite expressions.
type LengthTerm struct {
	Length Length
	Weight float64
}

// LengthSum returns the weighted sum of the supplied lengths. Undefined or zero-weight terms are ignored.
func LengthSum(terms ...LengthTerm) Length {
	sumTerms := make([]sumTerm, 0, len(terms))
	for _, t := range terms {
		if t.Weight == 0 {
			continue
		}
		node := lengthToNode(t.Length)
		if node == nil {
			continue
		}
		sumTerms = append(sumTerms, flattenTerms(node, t.Weight)...)
	}
	switch len(sumTerms) {
	case 0:
		return Length{}
	case 1:
		node := sumTerms[0].node.clone()
		node.scaleBy(sumTerms[0].weight)
		return lengthFromNode(node)
	default:
		node := &sumNode{scale: 1, terms: make([]sumTerm, len(sumTerms))}
		for i, term := range sumTerms {
			node.terms[i] = sumTerm{
				weight: term.weight,
				node:   term.node.clone(),
			}
		}
		return lengthFromNode(node)
	}
}

// LengthMin returns the minimum of the provided lengths.
func LengthMin(args ...Length) Length {
	if len(args) == 0 {
		return Length{}
	}
	nodes := make([]lengthNode, 0, len(args))
	for _, arg := range args {
		node := lengthToNode(arg)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return Length{}
	}
	return lengthFromNode(&minNode{args: nodes, scale: 1})
}

// LengthMax returns the maximum of the provided lengths.
func LengthMax(args ...Length) Length {
	if len(args) == 0 {
		return Length{}
	}
	nodes := make([]lengthNode, 0, len(args))
	for _, arg := range args {
		node := lengthToNode(arg)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return Length{}
	}
	return lengthFromNode(&maxNode{args: nodes, scale: 1})
}

// LengthClamp clamps value between min and max lengths.
func LengthClamp(minLen, valueLen, maxLen Length) Length {
	valueNode := lengthToNode(valueLen)
	if valueNode == nil {
		return Length{}
	}
	return lengthFromNode(&clampNode{
		value: valueNode,
		min:   lengthToNode(minLen),
		max:   lengthToNode(maxLen),
		scale: 1,
	})
}

func lengthToNode(l Length) lengthNode {
	if l.expr != nil {
		return l.expr.node.clone()
	}
	switch l.Unit {
	case LengthUnitDP:
		return &linearNode{dp: l.Value, scale: 1}
	case LengthUnitPercent:
		return &linearNode{percent: l.Value, scale: 1}
	case LengthUnitViewportWidth:
		return &linearNode{vw: l.Value, scale: 1}
	case LengthUnitViewportHeight:
		return &linearNode{vh: l.Value, scale: 1}
	case LengthUnitViewportMin:
		return &linearNode{vmin: l.Value, scale: 1}
	case LengthUnitViewportMax:
		return &linearNode{vmax: l.Value, scale: 1}
	default:
		if l.Value == 0 {
			return &linearNode{scale: 0}
		}
		return &linearNode{dp: l.Value, scale: 1}
	}
}

func lengthFromNode(node lengthNode) Length {
	if node == nil {
		return Length{}
	}
	return Length{
		Unit: LengthUnitAuto,
		expr: &lengthExpr{node: node},
	}
}

func flattenTerms(node lengthNode, weight float64) []sumTerm {
	switch n := node.(type) {
	case *sumNode:
		factor := weight * n.scale
		terms := make([]sumTerm, 0, len(n.terms))
		for _, term := range n.terms {
			childTerms := flattenTerms(term.node.clone(), term.weight)
			for i := range childTerms {
				childTerms[i].weight *= factor
				terms = append(terms, childTerms[i])
			}
		}
		return terms
	default:
		return []sumTerm{{weight: weight, node: node.clone()}}
	}
}

// Size represents a logical (device-independent) size in dp units.
type Size struct {
	W float64
	H float64
}

// Rect represents a logical (device-independent) rectangle in dp units.
type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

// PxSize represents an integer pixel size.
type PxSize struct {
	W int
	H int
}

// PxRect represents an integer pixel-space rect.
type PxRect struct {
	X int
	Y int
	W int
	H int
}

// Constraints represent min/max logical size bounds (in dp units).
// A zero Max (0) means unbounded/infinite in that axis.
type Constraints struct {
	Min Size
	Max Size
}

func (c Constraints) clamp(sz Size) Size {
	w := sz.W
	h := sz.H
	if c.Max.W > 0 {
		w = math.Min(w, c.Max.W)
	}
	if c.Max.H > 0 {
		h = math.Min(h, c.Max.H)
	}
	w = math.Max(w, c.Min.W)
	h = math.Max(h, c.Min.H)
	return Size{W: w, H: h}
}

// Infinite constraints in both axes.
func Infinite() Constraints { return Constraints{} }

// Tight constraints to a fixed size.
func Tight(sz Size) Constraints { return Constraints{Min: sz, Max: sz} }

// TightWidth constrains width exactly and height within optional max.
func TightWidth(w float64, maxH float64) Constraints {
	return Constraints{Min: Size{W: w}, Max: Size{W: w, H: maxH}}
}

// TightHeight constrains height exactly and width within optional max.
func TightHeight(h float64, maxW float64) Constraints {
	return Constraints{Min: Size{H: h}, Max: Size{W: maxW, H: h}}
}

// Helpers for dp<->px conversion.
func dpToPx(dp float64, scale float64) int {
	return int(math.Round(dp * scale))
}

func pxToDp(px int, scale float64) float64 {
	return float64(px) / scale
}

func (r Rect) ToPx(scale float64) PxRect {
	return PxRect{
		X: dpToPx(r.X, scale),
		Y: dpToPx(r.Y, scale),
		W: dpToPx(r.W, scale),
		H: dpToPx(r.H, scale),
	}
}

func (sz Size) ToPx(scale float64) PxSize {
	return PxSize{W: dpToPx(sz.W, scale), H: dpToPx(sz.H, scale)}
}

func (pr PxRect) ToDp(scale float64) Rect {
	return Rect{
		X: pxToDp(pr.X, scale),
		Y: pxToDp(pr.Y, scale),
		W: pxToDp(pr.W, scale),
		H: pxToDp(pr.H, scale),
	}
}

func (ps PxSize) ToDp(scale float64) Size {
	return Size{W: pxToDp(ps.W, scale), H: pxToDp(ps.H, scale)}
}
