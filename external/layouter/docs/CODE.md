**Code API Guide**
- This reference explains how to build UIs directly in Go using the `layout` package.
- Covers setup, component catalog, layout patterns, rendering flow, and best practices.

**1. Core Concepts**
- `layout.Component` is the interface for all UI nodes. Implementations embed `layout.Base` for caching.
- Measurements default to dp (device-independent pixels). Use `layout.Length` helpers (`LengthDP`, `LengthPercent`, `LengthVW`, `LengthVH`, `LengthVMin`, `LengthVMax`) when you need responsive or viewport-relative sizing. Rendering adapters convert resolved sizes to actual pixels using the device scale.
- Layout pass: `Measure` → `Layout` → `DrawTo`.
- Panels provide padding, background, text, and optional min sizes.

**2. Environmental Setup**
- Create a `layout.Context` each frame:
  ```go
  scaleProvider := adp.ScaleProvider{}
  renderer := adp.NewRenderer()
  textEngine := etxtadapter.New(scaleProvider)
  ctx := layout.NewContext(scaleProvider, renderer, textEngine)
  ```
- Integrate with backend by implementing `layout.Renderer` + `layout.Surface`. Ebiten adapter supplies both (plus wrappers for images).

**3. Rendering Loop**
- Typical frame:
  ```go
  root := buildComponentTree()
  canvas := adp.WrapCanvas(screen)
  layout.LayoutAndDraw(ctx, root, canvas)
  ```
- `LayoutAndDraw` handles measuring, layout, and drawing into the canvas using cached surfaces.
- Viewport-aware units rely on `ctx.SetViewportSize`. `LayoutAndDraw` manages this automatically; if you measure components off-screen, call `ctx.SetViewportSize` with the intended viewport before measuring and restore it afterward.

**4. Components Library**
- `layout.NewPanelComponent(child)` — general decorated container (`SetPadding`, `SetBackgroundColor`, `SetText`, `SetMinWidth`, `SetFillWidth`, etc.).
- `layout.NewLabel(text, style)` — convenience wrapper: panel with text.
- `layout.NewHStack(children...)` / `layout.NewVStack(children...)`
  - Configure with `Spacing`, `Justify`, `AlignH`/`AlignV`.
  - Children can participate in flex distribution by calling `child.SetFlexWeight(w)` (or `layout.SetFlexWeight(child, w)`). Remaining space is split in proportion to weights.
  - `layout.NewSpacer(weight)` is just a convenience weighted child—use it for gutters or pushing content apart.
- `layout.NewZStack(children...)`
  - Overlays children in order; use for backgrounds + foreground overlays.
- `layout.NewGrid(columns, spacing)`
  - Add children via `grid.Add(child)`.
  - Controls: `AlignH`, `AlignV`, `RowAlign`, plus `SetCellMinWidthLength(...)` / `SetCellMaxWidthLength(...)` for dp/percent/viewport (`vw`, `vh`, `vmin`, `vmax`) cell sizing.
- `layout.NewImage(image)`
  - Image from adapter: `img := layout.NewImage(adp.WrapImage(myEbitenImage))`.
  - Options: `SetFit(layout.ImageFitContain)`, `SetAlignment`, `SetExplicitSize`, `SetMaxWidth/Height`.
- `layout.NewPanelContainer(child, padding)` — panel with padding around child.

**5. Text Styling**
- `layout.TextStyle` fields:
  - `FontKey`: arbitrary key resolved by text engine.
  - `SizeDp`, `Color`.
  - `AlignH`, `AlignV`.
  - `Wrap`, `BaselineOffset`.
- Example:
  ```go
  label := layout.NewLabel("Top Stories", layout.TextStyle{
      SizeDp: 24, AlignH: layout.AlignCenter,
      Color: layout.Color{R: 230, G: 230, B: 255, A: 255},
  })
  ```
- Update text:
  ```go
  label.SetText("Updated Title")
  label.SetTextStyle(newStyle)
  ```

**6. Panels & Decoration**
- `panel := layout.NewPanelComponent(child)`
- `panel.SetPadding(layout.Insets(12))`
- `panel.SetBackgroundColor(layout.Color{R: 32, G: 48, B: 72, A: 255})`
- `panel.SetBackgroundImage(layout.ImagePaint{ Image: myImage, Fit: layout.ImageFitContain, AlignH: layout.AlignCenter, AlignV: layout.AlignCenter })`
- `panel.SetMinWidth(240)` or `panel.SetMinWidthLength(layout.LengthVW(0.3))` for responsive sizing (dp, percent, and viewport units all supported).
- `panel.SetMaxHeight(360)`, `panel.SetMaxHeightPercent(0.4)`, or `panel.SetMaxHeightLength(layout.LengthVH(0.5))` clamp overall bounds.
- `panel.SetFillWidth(true)` / `panel.SetFillHeight(true)` hint that the parent stack should stretch the child in that axis.
- `panel.SetFlexWeight(1)` makes the panel share leftover space with other weighted siblings.

**Responsive units**
- Express constraints with `layout.Length` helpers: `LengthDP(...)`, `LengthPercent(...)`, `LengthVW(...)`, `LengthVH(...)`, `LengthVMin(...)`, and `LengthVMax(...)`.
- Shorthand setters (`SetMinWidth`, `SetMinWidthPercent`, etc.) remain for convenience and internally build the matching `Length`.
- Percentages are relative to the parent constraints after padding; viewport units resolve each frame using the active viewport size (`LayoutAndDraw` keeps `ctx.SetViewportSize` up to date).
- Mixed constraints (e.g., min width in `vw` with a dp max) are supported—the engine clamps in order: min → measured content → max.
- Legacy factor-based APIs are still recognized but now emit warnings; prefer the length-based setters for future compatibility.

**7. Example Layout**
```go
func buildDashboard() layout.Component {
    title := layout.NewLabel("Dashboard", layout.TextStyle{
        SizeDp: 28, AlignH: layout.AlignCenter,
        Color: layout.Color{R: 240, G: 240, B: 255, A: 255},
    })
    titlePanel := layout.NewPanelContainer(title, layout.Insets(12))
    titlePanel.SetBackgroundColor(layout.Color{R: 28, G: 36, B: 54, A: 255})
    titlePanel.SetFillWidth(true)

    makeCard := func(name, value string) layout.Component {
        line := layout.NewLabel(value, layout.TextStyle{SizeDp: 16})
        caption := layout.NewLabel(name, layout.TextStyle{SizeDp: 12, Color: layout.Color{R: 160, G: 170, B: 190, A: 255}})
        block := layout.NewVStack(line, caption)
        block.Spacing = 4
        block.AlignH = layout.AlignCenter

        panel := layout.NewPanelContainer(block, layout.Insets(14))
        panel.SetBackgroundColor(layout.Color{R: 26, G: 34, B: 56, A: 255})
        panel.SetMinWidth(140)
        return panel
    }

    grid := layout.NewGrid(0, 12)
    grid.SetCellMaxWidthLength(layout.LengthVW(0.22)) // responsive max width
    grid.Add(makeCard("Latency", "82 ms"))
    grid.Add(makeCard("Requests", "12.4k/min"))
    grid.Add(makeCard("Errors", "0.14%"))
    grid.Add(makeCard("Active Users", "3,980"))

    column := layout.NewVStack(titlePanel, grid)
    column.Spacing = 16
    column.SetFillWidth(true)

    column.SetBackgroundColor(layout.Color{R: 18, G: 22, B: 34, A: 255})
    return column
}
```

**8. Custom Components**
- Embed `layout.Base` and implement `Measure`, `Layout`, `DrawTo`, `Render`.
- Use `Base.DrawCached` to benefit from automatic caching.
- Example overlay painter:
  ```go
  type PainterLayer struct { layout.Base; Paint func(ctx *layout.Context, dst layout.Surface) }
  func (p *PainterLayer) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size { return cs.Min }
  func (p *PainterLayer) Layout(ctx *layout.Context, parent layout.Component, b layout.Rect) { p.SetFrame(parent, b) }
  func (p *PainterLayer) DrawTo(ctx *layout.Context, dst layout.Surface) { p.Paint(ctx, dst) }
  func (p *PainterLayer) Render(ctx *layout.Context, dst layout.Surface) {}
  ```
- Compose with `layout.NewZStack(content, painterLayer)`.

**9. State Updates**
- Mutate component properties (text, color, min width). Call `SetDirty()` only when implementing custom components; built-ins mark themselves dirty automatically after property setters.
- For animations: update values each frame then rely on `DrawCached`.

**10. Tips**
- Use dp values consistently; avoid mixing with pixels.
- For flexible rows/columns: assign weights to the children that should expand.
- For fill behaviors, prefer `SetFillWidth(true)` / `SetFillHeight(true)` (or the XML/CSS equivalents) instead of measuring manually.
- Combine `Grid` with `columns=0` to auto-wrap based on available width.
- Cache adapters (images/text) for performance rather than re-creating surfaces.
- Bind to XML results: treat XML and code-built components interchangeably—they share the same interfaces.

**11. Debug Utilities**
- `layout.Base.SetDirty()` forces redraw.
- Inspect `Component.Bounds()` / `GlobalBounds()` to debug layout positions.
- Write unit tests using fake renderer/text engine (see `layout/layout_test.go`).

**12. Extending**
- Register custom XML tags mapped to Go builders.
- Add new components to examples: follow patterns shown in `examples/zoo`.
- Implement new adapters by satisfying `layout.Renderer` and `layout.Surface`.
