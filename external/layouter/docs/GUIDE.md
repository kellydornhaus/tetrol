# Layouter Guide

This guide walks through each component and feature with both code and XML/CSS examples. The core is layout‑only; rendering and text are pluggable.

## 1) Quick Start (Code)

```
import (
  "github.com/kellydornhaus/layouter/layout"
  adp "github.com/kellydornhaus/layouter/adapters/ebiten"
  txt "github.com/kellydornhaus/layouter/adapters/etxt"
)

scale := adp.ScaleProvider{}
renderer := adp.NewRenderer()
text := txt.New(scale) // etxt-backed TextEngine
ctx := layout.NewContext(scale, renderer, text)

title := layout.NewLabel("Hello", layout.TextStyle{SizeDp: 18})
root := layout.NewVStack(layout.NewPanelContainer(title, layout.Insets(8)))

// Per frame
canvas := adp.WrapCanvas(screen)
layout.LayoutAndDraw(ctx, root, canvas)
```

## 2) Quick Start (XML + CSS)

```
import (
  "strings"
  "github.com/kellydornhaus/layouter/layout"
  "github.com/kellydornhaus/layouter/xmlui"
)

xml := `<VStack spacing="8">\n  <Text id="title" font-size="18" align-h="center">Hello</Text>\n  <HStack spacing="6" justify="spaceBetween">\n    <Text>Left</Text><Spacer weight="1"/><Text>Right</Text>\n  </HStack>\n</VStack>`

res, _ := xmlui.Build(ctx, strings.NewReader(xml), nil, xmlui.Options{})
root := res.Root

var refs struct { Title *layout.PanelComponent `ui:"title"` }
_ = xmlui.BindByID(&refs, res)
refs.Title.SetText("World")
```

Minimal CSS (optional):

```
.title { font-size: 22dp; align-h: center; }
.row { spacing: 8dp; justify: spaceBetween; }
```

---

## 3) Components

All sizes/positions are in dp. The context’s scale factor converts dp↔px.

### Text

Code:

```
title := layout.NewLabel("Title", layout.TextStyle{SizeDp: 22, AlignH: layout.AlignCenter})
para  := layout.NewLabel("Wrapped paragraph.", layout.TextStyle{SizeDp: 14, Wrap: true})
para.SetTextMaxWidth(240)
```

XML:

```
<Text font-size="22" align-h="center">Title</Text>
<Text font-size="14" wrap="true" max-width="240">Wrapped paragraph.</Text>
```

### Box (padding)

```
panel := layout.NewPanelContainer(para, layout.Insets(8))
```

XML:

```
<Box padding="8"><Text>Inside padding</Text></Box>
```

### Spacer (flex)

```
row := layout.NewHStack(
  layout.NewLabel("A", layout.TextStyle{SizeDp:14}), layout.NewSpacer(1),
  layout.NewLabel("B", layout.TextStyle{SizeDp:14}), layout.NewSpacer(2),
  layout.NewLabel("C", layout.TextStyle{SizeDp:14}),
)
row.Spacing = 8
```

XML:

```
<HStack spacing="8">
  <Text>A</Text><Spacer weight="1"/><Text>B</Text><Spacer weight="2"/><Text>C</Text>
</HStack>
```

### HStack

```
row := layout.NewHStack(
  layout.NewPanelContainer(layout.NewLabel("Left",   layout.TextStyle{SizeDp:14}), layout.Insets(6)),
  layout.NewPanelContainer(layout.NewLabel("Center", layout.TextStyle{SizeDp:14}), layout.Insets(6)),
  layout.NewPanelContainer(layout.NewLabel("Right",  layout.TextStyle{SizeDp:14}), layout.Insets(6)),
)
row.Spacing = 8
row.AlignV  = layout.AlignCenter
row.Justify = layout.JustifySpaceBetween
```

XML:

```
<HStack spacing="8" align-v="center" justify="spaceBetween">
  <Text>Left</Text><Text>Center</Text><Text>Right</Text>
</HStack>
```

### VStack

```
col := layout.NewVStack(
  layout.NewLabel("Title", layout.TextStyle{SizeDp:20}),
  layout.NewLabel("Line 1", layout.TextStyle{SizeDp:14}),
)
col.Spacing = 8
col.AlignH  = layout.AlignCenter
col.Justify = layout.JustifyStart
```

XML:

```
<VStack spacing="8" align-h="center" justify="start">
  <Text font-size="20">Title</Text>
  <Text font-size="14">Line 1</Text>
</VStack>
```

### Grid (example component)

Grid is provided as a small helper in the examples (`examples/zoo/internal/components/grid.go`).

```
cells := []layout.Component{}
for i := 0; i < 6; i++ {
  t := layout.NewLabel(fmt.Sprintf("%d", i+1), layout.TextStyle{SizeDp: 14})
  cells = append(cells, layout.NewPanelContainer(t, layout.Insets(8)))
}
grid := cmp.NewGrid(3, 8, cells...) // 3 cols, 8dp spacing
```

XML (register a custom tag):

```
reg := xmlui.NewRegistry()
reg.Register("Grid", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {
  kids, _ := l.BuildChildren(n)
  return cmp.NewGrid(3, 8, kids...), nil
})
```

```
<Grid cols="3" spacing="8">
  <Text>1</Text><Text>2</Text><Text>3</Text>
  <Text>4</Text><Text>5</Text><Text>6</Text>
</Grid>
```

### BG / Border (example wrappers)

```
panel := layout.NewPanelContainer(para, layout.Insets(8))
panel.SetBackgroundColor(layout.Color{R: 30, G: 30, B: 60, A: 255})
panel.SetBorder(layout.Color{R: 220, G: 220, B: 220, A: 255}, 2)
panel.SetCornerRadius(12)
panel.SetWidth(240)
panel.SetHeight(120)
card := panel

badge := layout.NewPanelContainer(layout.NewLabel("New", layout.TextStyle{SizeDp: 13, AlignH: layout.AlignCenter}), layout.Insets(6))
badge.SetBackgroundColor(layout.Color{R: 120, G: 90, B: 210, A: 255})
badge.SetCornerRadii(layout.CornerRadii{TopLeft: 14, TopRight: 14, BottomRight: 4, BottomLeft: 4})
```

Backends that implement `layout.RoundedRenderer` (the Ebiten adapter does) draw those rounded fills/borders anti-aliased; other renderers fall back to rectangular edges.

Need to pin dimensions? Call `SetWidth` / `SetHeight` (or declare `width` / `height` in XML/CSS) instead of juggling matching min/max constraints.


### Visibility transitions

Components support a declarative visibility animation that scales them when toggling between hidden and visible. Configure it in Go by calling `layout.SetVisibilityTransition`, or directly in XML/CSS via the `visibility-transition` property.

```xml
<Panel id="toast"
       visibility="hidden"
       visibility-transition="size 600ms 50%"/>
```

The syntax is:

```
visibility-transition = <mode> <duration> <scale>
```

- `<mode>`: `size` animates both directions, `size-in` only when showing, `size-out` only when hiding.
- `<duration>` uses Go's duration strings (e.g., `150ms`, `1.2s`).
- `<scale>` accepts fractional values (`0.6`) or percentages (`60%`) and is clamped between 0 and 1.

From Go you can build the same configuration: 

```go
layout.SetVisibilityTransition(panel, layout.VisibilityTransition{
    Mode:     layout.VisibilityTransitionSize,
    Duration: 600 * time.Millisecond,
    Scale:    0.5,
})
```

While a transition runs the component is kept dirty, ensuring parents repaint each frame. Renderers can opt into the `layout.SurfaceScaler` interface for smooth sub-pixel scaling—Ebiten's adapter does—otherwise the engine falls back to integer-rounded rectangles and still works.
Animated elements can also be exposed as custom tags. The zoo demo registers a `<Particles>` element backed by `components.NewParticleField`, letting XML layouts drop in a live particle background with standard `background`, `border`, and custom `rate` attributes.

XML (register custom tags):

```
reg := xmlui.NewRegistry()
reg.Register("BG", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {
  col := parseHex(n.Attrs["color"]) // your parser
  kids, _ := l.BuildChildren(n)
  var child layout.Component
  if len(kids)==1 { child = kids[0] } else { child = layout.NewVStack(kids...) }
  panel := layout.NewPanelComponent(child)
  panel.SetBackgroundColor(layout.Color{R: col.R, G: col.G, B: col.B, A: col.A})
  panel.SetFillWidth(true)
  return panel, nil
})
```

```
<BG color="#202840"><Box padding="8"><Text>Banner</Text></Box></BG>
<Panel padding="6" border-color="#AA4444" border-width="2"><Box padding="6"><Text>Card</Text></Box></Panel>
```

---

## 4) Rendering + Text (Adapters)

Use the optional adapters to quickly hook Ebiten + etxt:

```
import (
  adp "github.com/kellydornhaus/layouter/adapters/ebiten"
  txt "github.com/kellydornhaus/layouter/adapters/etxt"
)
scale := adp.ScaleProvider{}
renderer := adp.NewRenderer()
text := txt.New(scale)
ctx := layout.NewContext(scale, renderer, text)
```

- Ebiten adapter: `Surface`, `Renderer`, `WrapCanvas`, `ScaleProvider`
- etxt adapter: `TextEngine` (as `txt.New(...)`)

---

## 5) XML + CSS Details

Built‑ins: `VStack`, `HStack`, `Box`, `Spacer`, `Text`.

Common attributes:
- `spacing`, `padding` (e.g., "8", "8,12" or "8,12,8,12")
- `justify`: `start|center|end|spaceBetween`
- `align-h`, `align-v`: `start|center|end`
- `font-size`, `font-autosize`, `font-autosize-min`, `font-autosize-max`, `color` (hex or CSS named colors), `wrap`, `max-width`, `weight`
- Any element can have `id` for later reference

CSS subset:
- Properties: `spacing`, `padding`, `justify`, `align-h`, `align-v`, `font-size`, `font-autosize`, `font-autosize-min`, `font-autosize-max`, `color`, `wrap`, `max-width`, `weight`
- Selectors: tag (e.g., `Text`), class (`.title`), id (`#title`), combined (`Text.title`)
- Precedence: XML attribute > inline `style` > stylesheet (id > class > tag; last wins)

Binding by id:

```
type Refs struct { Title *layout.PanelComponent `ui:"title"` }
var refs Refs
_ = xmlui.BindByID(&refs, result)
```

---

## 6) Scale + Caching

- Units are dp; set `ctx.Scale` from your platform (e.g., Ebiten’s `Monitor().DeviceScaleFactor()`).
- Embed `layout.Base` in components for bounds/dirty/cache handling.
- Draw via `Base.DrawCached(ctx, dst, renderFn)`, and call `SetDirty()` when visuals change.
- Caches recreate automatically on scale/size change.

---

## 7) Example App

Run the demo:

```
cd examples/zoo
go run .
```

It includes a “component zoo” showing stacks, alignment, spacers, wrapping, nesting, outlines, grid, and XML/CSS with hot‑reload.
