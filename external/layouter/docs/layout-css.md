# Layouter CSS Reference

This document outlines the declarative layout surface exposed by **Layouter**’s XML + CSS front‑end. It is written in a specification style so that authors can reason about what a property means, which values are accepted, and how different layout containers behave. Unless stated otherwise, property names correspond 1:1 with CSS identifiers (case-insensitive); unsupported properties are ignored.

---

## 1. Value grammar

| Value class | Description & syntax |
|-------------|----------------------|
| `<number>` | A floating-point number (`42`, `3.5`, `-10`). |
| `<integer>` | Whole number (`0`, `12`, `-4`). |
| `<bool>` | `true`, `false`, `yes`, `no`, `on`, `off` (case-insensitive). |
| `<duration>` | Go duration literal such as `150ms`, `1s`, `1.25s`, `2m`. Negative durations are clamped to zero. |
| `<length>` | Accepts:<br>• dp units: `24dp` or bare numbers (`24`).<br>• Percentages relative to the container (`35%`).<br>• Viewport units: `10vw`, `5vh`, `3vmin`, `4vmax`.<br>• Functional expressions combining the above via `calc(...)`, `min(...)`, `max(...)`, or `clamp(...)` (e.g. `calc(40% - max(12dp, 5%))`). |
| `<color>` | Hex (`#RRGGBB`, `#AARRGGBB`), short hex (`#RGB`), `rgba()` / `rgb()`, or CSS color keywords (`cornflowerblue`, `transparent`). |
| `<align-keyword>` | `start`, `center`, `end` (aliases recognised: `top`, `bottom`, `middle`, `left`, `right`, `leading`, `trailing`, `flex-start`, `flex-end`). |
| `<justify-keyword>` | `start`, `center`, `end`, `space-between`. |
| `<size-mode>` | `size`, `size-in`, `size-out`, or `none`. |
| `<image-fit>` | `contain`, `center` (`none`), `stretch` (`cover`, `fill`). |
| `<position-mode>` | `static`, `relative`, `absolute`, `fixed`. |
| `<ratio>` | `1.5`, `16/9`, `4:3`. |

Unless specified, unspecified properties fall back to component defaults. Percentages operate on the maximum size provided by the parent container (similar to CSS flex items).

---

## 2. Global properties

These properties apply to every element, regardless of type.

| Property | Value | Default | Notes |
|----------|-------|---------|-------|
| `visibility` | `visible` \| `hidden` \| `collapse` | `visible` | `hidden` omits rendering but still participates in layout; `collapse` removes the element from layout. |
| `visibility-transition` | `<size-mode> <duration> <scale>` | none | Enables size-based visibility animation. `<size-mode>` controls direction (`size` for both in/out, `size-in`, `size-out`). `<scale>` is the minimum scale (0–1 or `%`). |
| `position` | `<position-mode>` | `static` | Governs offsets. `relative` adjusts after layout; `absolute` removes from flow within nearest positioned ancestor; `fixed` anchors to viewport. |
| `top`, `right`, `bottom`, `left` | `<length>` | unset | Honoured for `relative`, `absolute`, `fixed`. `auto` or omitted clears the offset. |
| `z-index` | `<integer>` \| `auto` | `auto` | Applies when `position` is not `static`. Higher values render on top. |
| `align-self` | `<align-keyword>` \| `auto` | `auto` | Overrides cross-axis alignment inside `VStack`, `HStack`, `FlowStack`, `Grid`. |

Child elements inside stack-like containers may also declare:

| Property | Value | Default | Notes |
|----------|-------|---------|-------|
| `weight` | `<number>` | `0` | When greater than zero, participates in flex distribution inside stacks. |

---

## 3. Component catalogue

### 3.1 `<Panel>` (general container)

Panels provide padding, background, borders, text decoration, intrinsic sizing, and flex hints. Many other components embed a `Panel` internally.

| Property | Value | Default | Description |
|----------|-------|---------|-------------|
| `padding` | `<length>` \| `v1 v2` \| `top right bottom left` | `0` | Shorthand. Individual sides: `padding-top/right/bottom/left`. |
| `background` / `background-color` | `<color>` | none | Sets solid background. `background` also accepts hex/rgba values. |
| `tint-color` | `<color>` | none | Draws a translucent overlay above the background and rendered children; panel text is painted above the tint. |
| `background-image` | `url(...)` or path | none | Loads image via registry. |
| `background-fit` | `<image-fit>` | `stretch` | Scales background image. |
| `background-align-h` / `background-align-v` | `<align-keyword>` | `center` | Image alignment. |
| `background-slice` | `left,top,right,bottom` (integers, optional `dp`) | `0` | Enables nine-slice scaling. |
| `border` | shorthand (`<width> <color>`) | none | Sets all edges. Individual overrides: `border-{top,right,bottom,left}`, `border-*-width`, `border-*-color`. |
| `border-width`, `border-color` | `<length>`, `<color>` | none | Uniform border adjustments. |
| `corner-radius` | `<length>` \| `r1 r2 r3 r4` | `0` | Uniform radii. Per-corner: `corner-top-left-radius` etc. |
| `text` | string | inline text | Explicit text content. |
| `font` | font key | default registered font | Use `RegisterFont` keys. |
| `font-size` | `<length>` | `0` (auto) | Text size in dp. |
| `color` | `<color>` | adapter default | Text color. |
| `align-h`, `align-v` | `<align-keyword>` | `start` | Text alignment inside the panel. |
| `wrap` | `<bool>` | `false` | Enables text wrapping. |
| `baseline-offset` | `<length>` | `0` | Adjust baseline in dp. |
| `font-autosize` | `<bool>` | `false` | Scales font to fit content bounds. |
| `font-autosize-min`, `font-autosize-max` | `<length>` | `0` | Lower/upper clamps for auto sizing (`0` disables clamp). |
| `fill-width`, `fill-height` | `<bool>` | `false` | Hint to stretch across available space in parent stacks. |
| `width`, `height` | `<length>` | auto | Fixed size. Clears min/max constraints when unset. |
| `min-width`, `min-height`, `max-width`, `max-height` | `<length>` | auto | Bounding box constraints. |
| `aspect-ratio` | `<ratio>` | unset | Maintains width:height ratio when computing size. |

Panels honour the global properties (`visibility`, `position`, etc.).

### 3.2 `<VStack>`

Vertical stack with optional spacing and justification. Accepts Panel properties (background, padding, etc.).

| Property | Value | Default | Notes |
|----------|-------|---------|-------|
| `spacing` | `<length>` | `0` | Gap between children (dp). |
| `align-h` | `<align-keyword>` | `start` | Cross-axis alignment (affects children lacking `align-self`). |
| `justify` | `start` \| `center` \| `end` \| `space-between` | `start` | Main-axis distribution. |

Child nodes can use `weight` to participate in flexible spacing.

### 3.3 `<HStack>`

Horizontal analogue of `VStack`.

| Property | Value | Default | Notes |
|----------|-------|---------|-------|
| `spacing` | `<length>` | `0` | Horizontal gap. |
| `align-v` | `<align-keyword>` | `start` | Cross-axis alignment. |
| `justify` | `<justify-keyword>` | `start` | Main-axis distribution. |

### 3.4 `<FlowStack>`

Multi-line flow layout.

| Property | Value | Default | Description |
|----------|-------|---------|-------------|
| `spacing` | `<length>` | `0` | Horizontal gap between items. |
| `line-spacing` | `<length>` | `0` | Vertical gap between rows. |
| `align-items` | `<align-keyword>` | `start` | Alignment within each row. |
| `align-content` | `<align-keyword>` | `start` | Alignment across rows. |
| `justify` | `<justify-keyword>` | `start` | Horizontal alignment of rows. |

### 3.5 `<ZStack>`

Overlapping stack (painter’s order). Accepts panel properties for background/border/padding; children can set `z-index`.

### 3.6 `<Grid>`

Responsive grid of cells.

| Property | Value | Default | Description |
|----------|-------|---------|-------------|
| `columns` | `<integer>` \| `auto` | `auto` | Fixed column count; `auto` chooses based on content. |
| `spacing` | `<length>` | `20dp` | Gap between cells. |
| `align-h`, `align-v` | `<align-keyword>` | `center` | Alignment within each cell. |
| `row-align` | `<align-keyword>` | `start` | Alignment of partially filled last row. |
| `cell-min-width`, `cell-max-width` | `<length>` | none | Per-cell width bounds. |

Children inside grid may specify `weight` (used when cells wrap) and global properties.

### 3.7 `<Spacer>`

Fills remaining space inside stacks. Properties:

| Property | Value | Default | Notes |
|----------|-------|---------|-------|
| `weight` | `<number>` | `1` | Relative flex factor. |

Spacers ignore other styling attributes.

### 3.8 `<Text>`

Shorthand for a `Panel` with text content. Supports the text-related panel properties (`font-size`, `color`, `wrap`, etc.) plus global ones.

### 3.9 `<Image>`

Displays an image resource.

| Property | Value | Default | Description |
|----------|-------|---------|-------------|
| `src` | `url(...)` or path | required | Resource identifier. |
| `fit` | `<image-fit>` | `stretch` | Controls scaling. |
| `align-h`, `align-v` | `<align-keyword>` | `center` | Placement within bounds. |
| `width`, `height` | `<number>` (dp) | auto | Explicit size (dp). |
| `max-width`, `max-height` | `<length>` | none | Constraints similar to panels. |

### 3.10 `<Panel>` aliases

Layouter doesn’t ship a distinct `<Box>` element; examples often use `<Panel>` for boxed content. Custom registries can wrap panels to expose domain-specific tags (`<Card>`, `<BG>`, etc.).

---

## 4. Backgrounds & borders

- **Background color** applies after padding.
- **Background images** honour nine-slice (`background-slice`) before fitting.
- **Border shorthands** accept any combination of width and color tokens (`border: 2 transparent`, `border-top: 1 #ff0`).
- **Corner radii** clamp negative values to zero and support per-corner overrides.

---

## 5. Positioning model

- `position: static` (default) keeps the element in normal flow.
- `position: relative` offsets the element visually without affecting layout space.
- `position: absolute` removes the element from flow and positions it relative to the nearest positioned ancestor (or root).
- `position: fixed` anchors to the viewport; offsets interpreted in dp.
- Offsets accept `dp`, `px` (treated as dp), or raw numbers; missing sides default to auto.
- `z-index` only applies to non-static elements.

---

## 6. Visibility animations

When `visibility-transition` is present:

1. Switching to `hidden` or `visible` animates scale according to the chosen mode.
2. The component stays dirty across the transition so parents repaint intermediate frames.
3. After completion, rendering stops for hidden elements unless a transition keeps them visible (e.g., `size-in` on show).
4. Setting `visibility-transition="none"` or omitting the property yields immediate toggles.
5. Renderers implementing `SurfaceScaler` achieve smooth sub-pixel scaling; otherwise the engine falls back to integer-rounded rectangles.

---

## 7. Flex hints (`weight`, `fill-width`, `fill-height`)

- Child `weight` values participate in remaining-space distribution for `VStack`, `HStack`, and `FlowStack`. Zero means “size to content.”
- `fill-width` and `fill-height` advertise to parent stacks/grids that the element can stretch. Parents may honour or ignore the hint depending on container rules.

---

## 8. Text auto-fit

When `font-autosize` is true:

1. The engine searches for a font size that fits text within the panel’s content rectangle.
2. `font-autosize-min` and `font-autosize-max` clamp the search range (defaults: `0` meaning unbounded).
3. The resolved size is accessible via `Panel.AutoTextSize()` if queried from Go.

---

## 9. Imaging specifics

- `src` resolves through the loader’s `ImageLoader` callback; relative paths may be rewritten via `ResolveImagePath`.
- `fit=contain` preserves aspect ratio while fitting inside bounds.
- `fit=center` draws at natural size, centered inside the box.
- `fit=stretch` scales to fill bounds (distorting aspect ratio).
- Alignment keywords map to top/center/bottom and start/center/end.

---

## 10. Example

```xml
<VStack spacing="12" padding="16" background="#101522">
  <Panel id="banner"
         background="#24304a"
         padding="12"
         corner-radius="10dp"
         visibility-transition="size-in 400ms 70%">
    <Text font-size="18" color="#f5f6ff">Welcome</Text>
    <Text wrap="true" color="#c8cee6">
      Explore the controls below. They expand using weight-based flex distribution.
    </Text>
  </Panel>

  <HStack spacing="8" justify="space-between">
    <Panel class="tile" weight="1" fill-height="true">First tile</Panel>
    <Panel class="tile" weight="1" fill-height="true">Second tile</Panel>
    <Spacer weight="0.25"/>
    <Panel class="tile" width="120" visibility="hidden"
           visibility-transition="size-out 500ms 40%">Hidden until toggled</Panel>
  </HStack>
</VStack>
```

With stylesheet:

```css
.tile {
  padding: 10dp;
  background-color: #1c2335;
  border-radius: 8dp;
  align-h: center;
  align-v: center;
  color: #dde3f7;
}
```

---

This reference captures the current state of the Layouter CSS surface. As new properties or components are introduced, extend this document so authors can rely on a single, MDN-style specification.*** End Patch
