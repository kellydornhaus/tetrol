**XML + CSS Reference**
- XML UI builds component trees declaratively. Each tag maps to a builder registered with `xmlui.Registry`.
- CSS-like stylesheets apply attributes to matching nodes before construction.
- This document describes supported elements, attributes, styling rules, and practical examples.

**1. XML Document Structure**
- Root element becomes the root component. Whitespace is trimmed.
- Nodes may include:
  - Attributes (`key="value"`)
  - Inner text (for panels/Text)
  - Child elements (built recursively)
- Unknown tags raise an error unless `Options.IgnoreUnknown = true`.
- IDs (`id="hero"`) are collected for later lookup or binding.
- Classes (`class="title hot"`) feed into style matching.

**2. Built-In Elements**
- `VStack`, `HStack`, `ZStack`
  - `spacing`, `justify`, `align-h`, `align-v` (depending on orientation)
  - Content: zero or more child components.
- `Grid`
  - `columns="3"` or `columns="auto"`
  - `spacing`, `align-h`, `align-v`, `row-align`
  - `cell-min-width`, `cell-max-width` (dp, %, `vw`, `vh`) constrain per-cell width for responsive layouts.
  - Children populate cells row-major.
- `Panel`
  - Decorated container (padding/background/text).
  - Optional child when multiple nodes nested (extra nodes are wrapped in a vertical stack).
- `Image`
  - Bitmap rendering. Source supplied externally (adapter or registry).
- `Spacer`
  - `weight` (float) for flex growth.

**3. Panel Attributes**
- `padding="8"` or `padding="top,right,bottom,left"`, `padding="vertical,horizontal"`.
- `background="#RRGGBB"` or `#RRGGBBAA`.
- `tint-color="#RRGGBBAA"` (alias `tint`). Applies a translucent wash above the panel background/children; alpha controls intensity. Panel text is drawn above the tint.
- `text="Hello"` or inner text body.
- Text styling: `font`, `font-size`, `color`, `align-h`, `align-v`, `wrap="true"`, `baseline-offset`.
- Automatic text sizing: `font-autosize="true"` picks the largest font size that fits the inner bounds; optional `font-autosize-min` / `font-autosize-max` (dp) clamp the search.
- Colors accept hex (`#RRGGBB`, `#RRGGBBAA`) or standard CSS names (e.g., `navy`, `rebeccapurple`, `gainsboro`).
- Layout hints:
  - `fill-width="true"` / `fill-height="true"` stretch the panel inside stacks.
  - `weight="1"` participates in flex distribution (unused on its own, but stacks read it).
- Size hints:
  - `min-width`, `min-height`, `max-width`, `max-height` accept dp values (`320dp`), percentages (`50%`, `.5`), and viewport units (`15vw`, `40vh`, `12vmin`, `18vmax`). Unitless numbers default to dp; numbers between 0 and 1 are treated as percentages.
  - All size attributes also accept functional expressions (e.g. `calc(100vh - max(150dp, 10%))` or `clamp(120dp, 60%, 320dp)/1.5`) mixing dp, viewport units, and percentages with `+` / `-`, followed by optional `*` or `/` scalars outside the parentheses.
- Aspect ratio:
  - `aspect-ratio="16:9"` (or `aspect-ratio="1.777"`) constrains the panel to keep the specified width:height ratio after min/max hints have been applied.
- Panel text max width respects padding; values in dp (device-independent pixels).
- `Panel` nodes can nest children; when multiple child tags exist, they are wrapped in a `VStack`.

**4. Stack Attributes**
- `HStack` / `VStack`:
  - `spacing`: dp between items
  - `justify`: `start | center | end | spaceBetween`
  - `align-h` (for `VStack`) or `align-v` (for `HStack`): `start | center | end`.
  - Children can set `weight="1"` (or any positive number) to share leftover space; unweighted children keep their measured size.

**5. Grid Attributes**
- `columns="auto"` derives columns from available width.
- `spacing="dp"` adds gaps between cells.
- `align-h` / `align-v` center the entire grid within its bounds.
- `row-align` aligns partially filled last rows (`start|center|end`).

**6. Image Attributes**
- `src` (registry-provided path or ID).
- `fit="stretch|contain|center"` maps to `layout.ImageFit`.
- `align-h` / `align-v` adjust anchoring when `fit` does not stretch.
- `max-width`, `max-height` constrain logical size and accept dp, percentages, or viewport units; `width`, `height` (dp) override intrinsic dimensions.
- `width="auto"` / `height="auto"` defer that axis to the engine; when the other side is fixed, the image keeps its intrinsic ratio.

**7. CSS Stylesheets**
- Syntax: mini subset of CSS. Example:
  ```
  Panel.hero { padding: 24dp; background: #203040; }
  .title { font-size: 22dp; align-h: center; }
  #cta { background: #FF8800; color: #000; }
  ```
- Supported selectors:
  - Tag: `Panel`, `HStack`, etc.
  - Class: `.hero`
  - ID: `#cta`
  - Combinations: `Panel.hero`, `Panel#header`.
  - Multiple classes: `.card.primary`.
- Properties:
  - Layout: `spacing`, `padding`, `justify`, `align-h`, `align-v`, `width`, `height`, `min-width`, `min-height`, `max-width`, `max-height`, `aspect-ratio`, `weight`, `fill-width`, `fill-height`, `position`, `top`, `right`, `bottom`, `left`, `z-index`.
  - Panel: `text`, `font`, `font-size`, `font-autosize`, `font-autosize-min`, `font-autosize-max`, `color`, `background`, `tint-color`, `wrap`, `baseline-offset`.
  - Grid: `columns`, `row-align`.
  - Image: `fit`, `align-h`, `align-v`.
- Units:
  - Numbers default to dp; add `dp` suffix for clarity.
  - Values ending with `%` (or numbers between 0 and 1) are treated as percentages relative to the parent constraints.
  - Values ending with `vw` / `vh` / `vmin` / `vmax` resolve against the current viewport (CSS semantics, so `15vw` → 15% of viewport width; `10vmin` uses the smaller viewport axis). Fractions like `0.2vw` are also supported.
  - Functional expressions (`calc`, `min`, `max`, `clamp`) mix dp, percentages, and viewport units with `+` / `-` inside; you can multiply or divide the result by a scalar directly after the closing parenthesis (e.g. `calc(100vh - clamp(150dp, 20%, 40%))/1.4375`).
  - Colors use hex (6 or 8 digits) or standard CSS names.
  - Booleans (`wrap: true;`) case-insensitive.
- Precedence:
  - Inline `style="..."` overrides stylesheet rules.
  - More specific selectors beat generic ones (ID > class > tag).
  - Positioning notes: `position: static` keeps flow order; `relative` applies offsets after layout; `absolute` removes the element from flow and positions against the nearest positioned ancestor (or the root); `fixed` anchors to the viewport. Offsets are in dp, and opposing sides clamp size (e.g., `left`+`right`).

**8. Inline Styles**
- `style` attribute accepts semicolon-separated `key: value` pairs, e.g.
  ```
  <Panel style="padding:16dp; background:#223; font-size:18dp;">
      Inline styled panel
  </Panel>
  ```
- Keys follow the same names as CSS properties.

**9. Data Flow & Registry**
- Use `xmlui.NewRegistry()` to add or override builders.
- `Build(ctx, reader, registry, Options)`:
  1. Parses XML.
  2. Applies `Options.Classes` (extra class map).
  3. Executes stylesheet (`Options.Styles`).
  4. Builds components; collects ID map.
- Register new elements: `reg.Register("MyButton", func(l, node) (layout.Component, error) { ... })`.
- Builders can read arbitrary attributes, build children with `l.BuildChildren`, and return custom components.

**10. Binding Components**
- Results from `Build` expose `res.ByID`.
- Grab references: `title := res.ByID["title"]`.
- Use `xmlui.BindByID(&struct{ Title *layout.PanelComponent `ui:"title"` }, res)` for typed assignment.

**11. Practical Example**
```
<VStack class="page" spacing="12dp">
  <Panel id="title" class="hero" text="Dashboard"/>
  <HStack class="stats" spacing="10dp" justify="spaceBetween">
    <Panel class="stat" text="Users 1,204"/>
    <Spacer weight="1"/>
    <Panel class="stat" text="Revenue $35k"/>
  </HStack>
  <Grid columns="auto" spacing="8dp" cell-max-width="22vw">
    <Panel class="card" text="Task 1"/>
    <Panel class="card" text="Task 2"/>
    <Panel class="card" text="Task 3"/>
  </Grid>
</VStack>
```
CSS:
```
.page { padding: 24dp; }
.hero { font-size: 26dp; align-h: center; background: #1E2A40; color: #EEF; }
.stats { align-v: center; }
.stat { font-size: 18dp; padding: 16dp; background: #152032; }
.card { padding: 12dp; background: #1A273A; }
```
- Result: vertical layout with hero heading, stats row, responsive grid.

**12. Tips / Best Practices**
- Prefer dp units; engine converts to pixels using device scale.
- Wrap raw text in `Panel` (with `text=` or inner body) to maintain consistent visuals.
- For backgrounds or borders, use panel helpers (`SetBackgroundColor`, `SetBorder`) or embed child inside custom components (e.g., `BGImage`).
- Use `fill-width` / `fill-height` when elements should stretch with their parent, and assign `weight` to siblings that must share leftover space.
- Viewport units (`vw`, `vh`, `vmin`, `vmax`) resolve automatically each build; combine them with percentages to express responsive layouts (e.g., `min-height="40vh"`, `cell-max-width="18vmin"`).
- When customizing heavily, create Go builders that compose primitives, then register under an XML tag.
- Keep CSS files small; selectors apply globally within one build invocation.
- Use `Options.Classes` to add run-time state classes (e.g., highlight navigation tab).

**13. Troubleshooting**
- Nothing appears: ensure root element resolves to a registered tag.
- Styles not applied: check selector specificity and confirm stylesheet parsed without error.
- IDs missing: attribute must be set prior to build; dynamic changes require manual tracking.
- Alignment off: remember `Panel` defaults to content width; use CSS `width` along with code-set `SetFillWidth` when building custom components.
- Alignment off: remember `Panel` defaults to content width; add `fill-width: true` (or explicit min/max widths) to stretch with its container.

**14. Next Steps**
- Combine with Go code via `xmlui.Build`.
- Extend registries with custom tags for domain-specific layouts.
- Use binding helpers to modify text/colors at runtime and call `SetText`/`SetBackgroundColor` as needed.
