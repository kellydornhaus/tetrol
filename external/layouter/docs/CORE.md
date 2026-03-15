**Overview**
- This package provides a tiny, backend‑agnostic layout engine for Go.
- It focuses on responsive layout in device‑independent units (dp) and efficient drawing via per‑component caching.
- Integrate with your renderer (e.g., Ebiten) and text stack (e.g., etxt) through small adapter interfaces.

**Core Concepts**
- Units: All layout math uses dp. Convert dp↔px using the device scale factor.
- Bounds: Components have local bounds relative to their parent.
- Constraints: Measurement happens within `Constraints{Min, Max}`; `Max.W/H == 0` means unbounded.
- Caching: Each component can cache its rendered output in a pixel surface sized to its dp bounds × scale.

**Key Types**
- `Size`, `Rect`: Logical sizes/rectangles in dp.
- `Constraints`: Min/Max in dp; helpers `Infinite()`, `Tight(Size)`, `TightWidth`, `TightHeight`.
- `Context`: Carries `Scale` (float64), `Renderer`, and `Text` engine.
- `Renderer`/`Surface`/`Canvas`: Minimal drawing API to create offscreens and composite them.
- `TextEngine`: Minimal text ops: `Measure` (returns px width,height) and `Draw` into a px rect.

**Component Interface**
- Methods:
  - `Measure(ctx, cs) Size`: Return desired size under constraints (in dp).
  - `Layout(ctx, parent, bounds Rect)`: Receive the parent and final local bounds (in dp). Assign children here.
  - `DrawTo(ctx, dst Surface)`: Draw self (and subtree) onto `dst`. Use internal caching for efficiency.
  - `Render(ctx, dst Surface)`: Paint into the component’s own cached surface at local origin (0,0).
  - `Dirty() bool`: Whether the cached rendering is invalid.
  - `Bounds() Rect`: Local bounds relative to the parent.
  - `GlobalBounds() Rect`: Bounds in root coordinates (dp). Useful for pointer interaction and tooling.
- Base support:
  - Embed `layout.Base` to get bounds tracking, dirty flag, and cached surface management.
  - `Base.SetFrame(parent, bounds)` wires the parent link and marks dirty on bounds change.
  - `Base.SetDirty()` when visual state changes (content, style, children layout that affects visuals).
  - `Base.DrawCached(ctx, dst, renderFn)` handles cache allocation/resizing, clears when dirty, calls `renderFn` into the cache, then composites onto `dst` at the component’s pixel position.

**Flow and Coordinate Spaces**
- dp space: `Measure` and `Layout` always use dp.
- px space: `Render` paints to a px surface sized to `bounds.ToPx(ctx.Scale)` at local origin.
- Composition: `Renderer.DrawSurface(dst, cache, pxX, pxY)` places the cached image at parent’s pixel coordinates.
- Global queries: call `component.GlobalBounds()` to get absolute dp coordinates; parents flow through `Layout` so `Base` maintains links automatically.

**Constraints and Measurement**
- `cs.Max.W/H == 0` means unbounded along that axis; otherwise it’s a strict cap.
- Always clamp measured sizes: container returns `cs.clamp(Size{...})` semantics (see `Constraints.clamp`).
- Use helpers:
  - `Infinite()` for unconstrained measure.
  - `Tight(Size{W,H})` to force a fixed size.
  - `TightWidth(w, maxH)` / `TightHeight(h, maxW)` for single‑axis tight constraints.

**Renderer, Surface, Canvas**
- `Surface`:
  - `SizePx() (w, h int)` returns dimensions in pixels.
  - `Clear()` clears the surface (e.g., transparent).
- `Renderer`:
  - `NewSurface(w,h int) Surface` creates offscreen images.
  - `DrawSurface(dst, src, x, y int)` composites `src` into `dst` at pixel position.
- `RoundedRenderer` (optional):
  - `FillRoundedRect(dst, rect, radii, color)` draws anti-aliased rounded rectangles.
  - `StrokeRoundedRect(dst, rect, radii, strokeWidth, color)` draws rounded borders matching the radii.
- `Canvas`:
  - A `Surface` that represents the frame destination (e.g., backbuffer/screen).
- Ebiten adapter guidance:
  - Wrap `*ebiten.Image` to implement `Surface` and `Canvas`.
  - `NewSurface` → `ebiten.NewImage(w,h)`.
  - `DrawSurface` → `dst.DrawImage(src, opts)` with `opts.GeoM.Translate(x,y)`.
  - `Clear` → `img.Clear()`.

**TextEngine Integration**
- Interface:
  - `Measure(text, style, maxWidthPx) (w, h int)`; return px size with optional wrapping at `maxWidthPx`.
  - `Draw(dst, text, rectPx, style)`; draw within `rectPx` using the style.
- Style mapping:
  - `TextStyle{FontKey, SizeDp, Color, AlignH, AlignV, Wrap}`. Convert `SizeDp` → px via `round(SizeDp * ctx.Scale)`.
  - Align mapping must anchor to the given rect according to your text stack’s semantics.
- etxt specifics:
  - Initialize a renderer, set cache `Utils().SetCache8MiB()`.
  - `SetFont`, `SetColor`, `SetAlign`, `SetScale(scale)`, `SetSize(px)`.
  - Use `DrawWithWrap`/`MeasureWithWrap` when wrapping; width limit is px (not scaled by `SetScale`).

**Dirty and Caching**
- When to mark dirty:
  - Content/style changes that affect pixels.
  - Bounds changed (already handled in `Base.SetFrame`).
  - Device scale changes (cache is recreated automatically; `Base` tracks `cacheScale`).
  - Child became dirty (containers should check `child.Dirty()` during `Layout` or before drawing and call `SetDirty()`).
- To disable surface caching entirely (force redraw every frame), set `ctx.DisableCaching = true`.
- Do not mutate `ctx.Scale` mid‑layout; update it per frame before layout/draw.

**Common Containers**
- `Panel` / `PanelComponent`: Single-child container with padding helpers (`SetPadding`/`Insets`), decoration (`SetBackgroundColor`, `SetBorder`, `SetCornerRadius`, `SetCornerRadii`), and size locks (`SetWidth`/`SetHeight` alongside min/max variants). Radii are in dp; backends without `RoundedRenderer` fall back to rectangular fills/borders.
- `Box(Padding, Child)`: Adds padding around a single child.
- `HStack/VStack`:
  - `Spacing`: gap in dp between items.
  - `AlignV`/`AlignH`: cross-axis alignment.
  - `Justify`: main‑axis distribution (Start/Center/End/SpaceBetween).
  - `Spacer(weight)`: flexible items that divide extra space by weights.
  - Children can call `layout.SetAlignSelf` (or XML `align-self`) to override cross-axis alignment per element.
- `FlowStack`:
  - Lays out children horizontally and wraps onto additional rows when the available width is exhausted.
  - `Spacing`: gap between siblings; `LineSpacing`: gap between rows.
  - `AlignItems`: cross-axis alignment inside each row, `AlignContent`: vertical distribution across rows.
  - Shares `Justify` semantics with `HStack` for distributing extra space across a row.

**Writing a Custom Component**
- Embed `Base`.
- Implement `Measure`:
  - Compute natural size in dp, clamp with constraints.
- Implement `Layout`:
  - `SetFrame(parent, bounds)`, layout children in local coordinates.
- Implement `DrawTo`:
  - `Base.DrawCached(ctx, dst, func(cache) { Render(ctx, cache) })`.
- Implement `Render`:
  - Paint content into `dst` at local origin.
- Mark `SetDirty()` on state changes.

**End‑to‑End Example**
- Initialize adapters:
  - `renderer := MyRenderer{}`
  - `scale := MyScaleProvider{}`
  - `text := MyEtxtEngine{}`
  - `ctx := layout.NewContext(scale, renderer, text)`
- Build tree:
  - `title := layout.NewLabel("Hello", layout.TextStyle{SizeDp: 18})`
  - `root := layout.NewVStack(layout.NewPanelContainer(title, layout.Insets(8)))`
- Per frame:
  - `canvas := WrapScreen(screen)`
  - `layout.LayoutAndDraw(ctx, root, canvas)`

**Performance Tips**
- Build the component tree once; reuse across frames.
- Use per‑element `SetDirty()` when text/style changes; avoid redrawing everything.
- Avoid creating transient surfaces in custom `Render`.
- If showing/hidden toggles change layout, prefer swapping components or setting children to `nil` and calling `SetDirty()`.
