**What This Is**
- Tiny, responsive layout engine for Go.
- Pure Go with ZERO required dependencies in the core.
- Layout-only — render with any engine by plugging a tiny interface.
- Optional adapters for Ebiten + etxt (separate modules).
- Code-first layout (Panel, HStack, VStack, FlowStack, Spacer, Grid, Image) or XML+CSS.
- Easy to extend: custom components and custom XML elements.

**Quick Start (Code-First)**
- Create a `Context` with your renderer + text engine, then lay out and draw each frame.

```
import (
  "github.com/kellydornhaus/layouter/layout"
  adp "github.com/kellydornhaus/layouter/adapters/ebiten"
  txt "github.com/kellydornhaus/layouter/adapters/etxt"
)

scale := adp.ScaleProvider{}
renderer := adp.NewRenderer()
text := txt.New(scale) // etxt-backed TextEngine; uses Go Regular by default
ctx := layout.NewContext(scale, renderer, text)

hello := layout.NewLabel("Hello", layout.TextStyle{SizeDp: 18})
root := layout.NewVStack(
  layout.NewPanelContainer(hello, layout.Insets(12)),
)
root.SetFillWidth(true)

// in your game/app draw:
layout.LayoutAndDraw(ctx, root, adp.WrapCanvas(screen))
```

**Stacks, Flow, Grid (Examples)**
- HStack with weighted spacers:

```
makeItem := func(label string) layout.Component {
  return layout.NewPanelContainer(
    layout.NewLabel(label, layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter}),
    layout.Insets(6),
  )
}
row := layout.NewHStack(
  makeItem("A"), layout.NewSpacer(1),
  makeItem("B"), layout.NewSpacer(2),
  makeItem("C"),
)
row.Spacing = 8
```

- FlowStack wraps chips across rows while staying centered:

```
cloud := layout.NewFlowStack()
cloud.Spacing = 8
cloud.LineSpacing = 12
cloud.Justify = layout.JustifyCenter
cloud.AlignItems = layout.AlignCenter
for _, label := range []string{"Primary", "Secondary", "Success", "Warning"} {
  pill := layout.NewPanelContainer(layout.NewLabel(label, layout.TextStyle{SizeDp: 13, AlignH: layout.AlignCenter}), layout.Insets(6))
  pill.SetCornerRadius(999)
  pill.SetBackgroundColor(layout.Color{R: 80, G: 60, B: 160, A: 255})
  cloud.Children = append(cloud.Children, pill)
}
```

- Simple 3-column Grid:

```
grid := layout.NewGrid(3, 12)
for _, item := range data {
  grid.Add(layout.NewPanelContainer(layout.NewLabel(item, layout.TextStyle{SizeDp: 14}), layout.Insets(8)))
}
```

**Pluggable Rendering + Text**
- Core interfaces you implement (or adapt):

```
type Renderer interface {
  NewSurface(pxW, pxH int) Surface
  DrawSurface(dst, src Surface, x, y int)
  FillRect(dst Surface, rect PxRect, color Color)
  DrawImage(dst Surface, img Image, rect PxRect)
}
type RoundedRenderer interface {
  FillRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, color Color)
  StrokeRoundedRect(dst Surface, rect PxRect, radii PxCornerRadii, strokeWidth float64, color Color)
}
type Surface interface { SizePx() (int,int); Clear() }
type Image interface { SizePx() (int,int) }
type TextEngine interface {
  Measure(text string, style TextStyle, maxWidthPx int) (w, h int)
  Draw(dst Surface, text string, rectPx PxRect, style TextStyle)
}
```
- Implementing the optional `RoundedRenderer` interface lets the engine request anti-aliased rounded fills and strokes when panels use corner radii.

- Optional adapters (separate modules; import them only when you need them):
  - `github.com/kellydornhaus/layouter/adapters/ebiten` (Surface/Renderer/ScaleProvider)
  - `github.com/kellydornhaus/layouter/adapters/etxt` (TextEngine for etxt)

**Custom Component (Example)**
- Fill the component’s bounds with a solid color:

```
type ColorBlock struct{ layout.Base; RGBA layout.Color }
func (c *ColorBlock) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size { return cs.Min }
func (c *ColorBlock) Layout(ctx *layout.Context, parent layout.Component, b layout.Rect) {
  c.SetFrame(parent, b)
}
func (c *ColorBlock) DrawTo(ctx *layout.Context, dst layout.Surface) {
  c.Base.DrawCached(ctx, dst, func(t layout.Surface) { c.Render(ctx, t) })
}
func (c *ColorBlock) Render(ctx *layout.Context, dst layout.Surface) {
  // paint into dst using your backend (see examples for Ebiten)
}
```

**Rounded Panels & Buttons**
- Panels expose `SetCornerRadius` for uniform rounding or `SetCornerRadii` for per-corner control (dp units). Backends that also satisfy `RoundedRenderer` get smooth fills and borders automatically.
- Use `SetWidth`/`SetHeight` (or their percent/length variants) to lock a panel's size without micromanaging min/max constraints; XML/CSS `width` and `height` map to the same helpers.

```
card := layout.NewPanelContainer(content, layout.Insets(12))
card.SetBackgroundColor(layout.Color{R: 48, G: 36, B: 82, A: 255})
card.SetCornerRadii(layout.CornerRadii{
  TopLeft: 24, TopRight: 24,
  BottomRight: 8, BottomLeft: 8,
})
card.SetBorder(layout.Color{R: 220, G: 180, B: 255, A: 255}, 2)
card.SetWidth(240)
card.SetHeight(120)

pill := layout.NewPanelContainer(
  layout.NewLabel("Action", layout.TextStyle{SizeDp: 14, AlignH: layout.AlignCenter}),
  layout.Insets(10),
)
pill.SetBackgroundColor(layout.Color{R: 120, G: 90, B: 210, A: 255})
pill.SetCornerRadius(999) // pill-style button
```

**Debugging Layout/CSS**
- Enable verbose logging per `Context`:

```
ctx := layout.NewContext(scale, renderer, text)
ctx.Debug = layout.DebugOptions{
  LogLayoutDecisions: true, // dumps measure/layout bounds
  LogCSSQueries:      true, // dumps resolved CSS attrs (XML/CSS builds)
  LogSurfaceAllocations: true, // logs cache surface sizes/allocs
  LogSurfaceUsage:       true, // logs cache hits vs redraws
}
```
- Debug output is buffered per frame and only flushed if that frame’s log differs from the previous one; the flush is prefixed with frame number/time (individual entries have no timestamps).

- Or flip env vars globally (picked up by every `NewContext`): set `LAYOUT_DEBUG_LAYOUT=1`, `LAYOUT_DEBUG_CSS=1`, `LAYOUT_DEBUG_SURFACE_ALLOC=1`, `LAYOUT_DEBUG_SURFACE_USAGE=1` (or `LAYOUT_DEBUG_SURFACES=1` for both surface options).

- The example zoo exposes flags: `go run ./examples/zoo -log-layout -log-css -log-surfaces` (headless: add `-headless -out screenshots`).

- To disable component surface caching (force full redraws), set `ctx.DisableCaching = true`.

**XML + CSS (Optional)**
- Declare UI with XML (plus a tiny CSS subset) and build at runtime:

```
xml := `<VStack spacing="8">
  <Panel id="title" font-size="18" align-h="center" text="Hello"/>
  <HStack spacing="6" justify="spaceBetween">
    <Panel text="Left"/><Spacer/><Panel text="Right"/>
  </HStack>
</VStack>`

res, _ := xmlui.Build(ctx, strings.NewReader(xml), nil, xmlui.Options{})
root := res.Root

var refs struct { Title *layout.PanelComponent `ui:"title"` }
_ = xmlui.BindByID(&refs, res)
refs.Title.SetText("World")
```

- CSS properties include: `spacing`, `padding`, `justify`, `align-h`, `align-v`, `font-size`, `font-autosize`, `font-autosize-min`, `font-autosize-max`, `color`, `wrap`, `max-width`, `weight`, `border`, `border-color`, `border-width`, `position`, `top`, `right`, `bottom`, `left`, `z-index`.
- Color values accept hex (`#RRGGBB`, `#RRGGBBAA`) or standard CSS named colors (`navy`, `rebeccapurple`, etc.).
- Custom XML tags: register builders:

```
reg := xmlui.NewRegistry()
reg.Register("BG", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {
  col := parseHex(n.Attrs["color"]) // you implement
  kids, _ := l.BuildChildren(n)
  var child layout.Component
  if len(kids)==1 { child = kids[0] } else { child = layout.NewVStack(kids...) }
panel := layout.NewPanelComponent(child)
panel.SetBackgroundColor(col)
panel.SetFillWidth(true)
return panel, nil
})

// Particles element shows an animated background powered by a custom component.
reg.Register("Particles", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {
  field := components.NewParticleField()
  if rate := strings.TrimSpace(n.Attrs["rate"]); rate != "" {
    field.SetEmissionRate(parseFloat(rate))
  }
  l.applyPanelAttributes(n, field) // reuse padding/background/border helpers
  return field, nil
})
res, _ := xmlui.Build(ctx, strings.NewReader(xml), reg, xmlui.Options{Styles: css})
// or load from disk + xml-stylesheet directive:
// res, _ := xmlui.BuildFile(ctx, "layout.xml", reg, xmlui.Options{})
```

**Use In Your Project (Local)**
- In your app’s `go.mod`:

```
require github.com/kellydornhaus/layouter v0.0.0
require github.com/kellydornhaus/layouter/adapters/ebiten v0.0.0
require github.com/kellydornhaus/layouter/adapters/etxt v0.0.0

replace github.com/kellydornhaus/layouter => /absolute/path/to/layouter
replace github.com/kellydornhaus/layouter/adapters/ebiten => /absolute/path/to/layouter/adapters/ebiten
replace github.com/kellydornhaus/layouter/adapters/etxt   => /absolute/path/to/layouter/adapters/etxt
```

- Imports:

```
import "github.com/kellydornhaus/layouter/layout"
import "github.com/kellydornhaus/layouter/xmlui"
import adp "github.com/kellydornhaus/layouter/adapters/ebiten"
import txt "github.com/kellydornhaus/layouter/adapters/etxt"
```

**Run The Example Zoo**
- A complete Ebiten + etxt demo lives in `examples/zoo`.
- Run: `cd examples/zoo && go run .`
- Check out the “Rounded Corners” screen to see corner radii, borders, and pill buttons rendered with the Ebiten adapter.
- The new “Positioning” page demonstrates CSS `position`/`top`/`left` offsets, absolute overlays, and viewport-fixed badges driven entirely by XML + CSS.

**More Docs**
- Core API: `docs/CORE.md` (components, caching, scale, tips)
- XML UI: `docs/XMLUI.md` (XML/CSS, custom tags, binding)
