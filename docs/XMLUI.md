**Overview**
- `xmlui` lets you declare layouts in XML and build a tree of `layout.Component`s.
- It supports custom elements via a registry and exposes an `id → component` map and a binder to get strong references.
- XML is parsed with the stdlib `encoding/xml` and uses only this repository’s APIs (no external deps).

**Quick Start**
- Registry + build:
  - `reg := xmlui.NewRegistry()`
  - `reg.Register("BG", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) { ... })`
  - `res, err := xmlui.Build(ctx, reader, reg, xmlui.Options{})`
  - `root := res.Root`
- Get references:
  - `btn := res.ByID["okButton"].(*MyButton)`
  - or: `var refs struct { Title *layout.PanelComponent ` + "`ui:\"title\"`" + ` } ; _ = xmlui.BindByID(&refs, res)`

**Elements**
- Built‑ins registered by default:
  - `VStack`: vertical stack
  - `HStack`: horizontal stack
  - `FlowStack`: horizontal flow that wraps across rows
  - `ZStack`: overlay stack
  - `Grid`: responsive grid
  - `Panel`: decorated container (padding, background, text)
  - `Image`: bitmap node (source supplied by adapter/registry)
  - `Spacer`: flexible spacer for stacks
- Structural helpers:
  - `template`: captures a subtree without instantiating it immediately. See **Templates** below.

**Attributes**
- Common:
  - `id`: capture a reference for later code access.
  - Numbers accept optional `dp` suffix; e.g., `"8dp"` equals `8`.
  - `align-self`: overrides a child’s cross-axis alignment inside stacks/flow (accepts `start | center | end | auto | stretch`).
- `VStack` / `HStack`:
  - `spacing`: dp between items
  - `justify`: `start | center | end | spaceBetween`
  - `align-h` (for `VStack`), `align-v` (for `HStack`): `start | center | end`
- `FlowStack`:
  - `spacing`: dp between siblings
  - `line-spacing`: dp between rows
  - `justify`: `start | center | end | spaceBetween`
  - `align-items`: cross-axis alignment per row (`start | center | end`)
  - `align-content`: vertical distribution of rows (`start | center | end`)
- `Spacer`:
  - `weight`: number (default `1`)
- `Panel`:
  - `padding`: `all` or `top,right,bottom,left` or `vertical,horizontal`
  - `background`: `#RRGGBB`/`#RRGGBBAA` or CSS named colors
  - Text via `text` attribute or inner text body
  - Text style: `font`, `font-size`, `font-autosize`, `font-autosize-min`, `font-autosize-max`, `color`, `align-h`, `align-v`, `wrap`, `baseline-offset` (colors accept hex or CSS keywords like `navy`, `rebeccapurple`)
  - Layout hints: `fill-width`, `fill-height`, `weight`
  - Size hints: `min-width`, `min-height`, `max-width`, `max-height`, and `aspect-ratio`
    - Length values accept dp (`320dp`), percentages (`50%`, `.5`), viewport units (`15vw`, `40vh`, `12vmin`, `18vmax`), or functional expressions (`calc(...)`, `min(...)`, `max(...)`, `clamp(...)`) combining those units (e.g., `calc(100vh - max(150dp, 10%))/1.4375`).
- `Image`:
  - `fit`: `stretch | contain | center`
  - `align-h`, `align-v` (when not stretching)
  - `width`, `height`, `max-width`, `max-height`

**API**
- `type BuildFunc func(l *Loader, n *Node) (layout.Component, error)`
  - Use `l.BuildChildren(n)` to build child nodes.
  - Use `l.BuildNode(child)` for selective builds.
- `func Build(ctx *layout.Context, r io.Reader, reg *Registry, opts Options) (Result, error)`
  - Merges default builders with any provided custom `reg`.
  - `Options.IgnoreUnknown`: skip unknown tags instead of error.
- `type Result struct { Root layout.Component; ByID map[string]layout.Component; Templates map[string]*Template }`
- `func BindByID(dest any, r Result) error`
  - Set fields tagged with `ui:"id"` or fields named exactly like the id.
  - Allowed field types: `layout.Component` or concrete component pointer (e.g., `*layout.Text`).
- `func (Result) Template(id string) (*Template, bool)` looks up a captured template by id.
- `type Template struct { ... }`
  - `Instantiate() (TemplateInstance, error)` builds a fresh set of components using the template definition.
- `type TemplateInstance struct { Components []layout.Component; ByID map[string]layout.Component }`
  - Call `AddClass(comp, class)` / `RemoveClass` / `Classes` on the instance to toggle classes applied inside the template clone.
  - Components produced by a template instance are not added to `Result.ByID`; manage their references from the returned slice/map.
- `layout.SetDialogVariable(component, name, value)` overrides a dialog variable on the component and all descendants.
- `layout.ClearDialogVariable(component, name)` removes a local override.
- `layout.DialogVariable(component, name)` looks up a variable, searching up the parent chain.

**Templates**
- Declare reusable markup inside a `<template id="card"> … </template>` element anywhere in the XML tree. The template body is parsed and styled, but not added to the live component tree.
- Retrieve the template via `tmpl, _ := res.Template("card")` and build concrete copies on demand:
  ```go
  inst, _ := tmpl.Instantiate()
  card := inst.Components[0]
  layout.SetDialogVariable(card, "Name", "Commander Vega")
  layout.SetDialogVariable(card, "Role", "Navigation Lead")
  flow.Add(card)
  ```
- Every call to `Instantiate` returns new components and its own `TemplateInstance` helper for class management. Add/remove classes with `inst.AddClass(...)` to keep stylesheet styling in sync.
- Template IDs must be unique. Missing or duplicate ids trigger load errors.
- Common use cases: repeatable cards, list tiles, or menu rows populated from Go collections.
- Placeholders inside text use `{{VariableName}}`. Descendants inherit variables from their ancestors unless they override them with `layout.SetDialogVariable`.

**Dialog variables (general use)**
- The same `{{VariableName}}` syntax works for any `Panel` text, not just inside templates. Call `layout.SetDialogVariable(component, "Name", "Joseph")` on the container that owns the data and the formatted text updates the next time it is measured or drawn. Use `layout.ClearDialogVariable` to drop a local override, and `layout.DialogVariable` to read the current value.
- Variables resolve by walking up the parent chain. A child that defines `Name` shadows the value provided by its ancestors; clearing the child override reveals the parent value again. Missing variables fall back to an empty string.

**Custom Elements**
- Register a builder that returns your component. Example `BG` tag wrapping children:
  - `reg.Register("BG", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {`
  - `  col := parseColor(n.Attrs["color"])`
  - `  children, _ := l.BuildChildren(n)`
  - `  var child layout.Component`
  - `  if len(children) == 1 { child = children[0] } else { child = layout.NewVStack(children...) }`
  - `  panel := layout.NewPanelComponent(child)`
  - `  panel.SetBackgroundColor(col)`
  - `  panel.SetFillWidth(true)`
  - `  return panel, nil })`
- Guidelines:
  - Keep layout responsibilities in components; builders just adapt XML attributes to constructor parameters and properties.
  - When setting properties that affect visuals or layout after build, call `SetDirty()` on those components.

**XML Semantics**
- Whitespace handling: inner text is trimmed (leading/trailing whitespace removed). Use explicit tags and `padding` for spacing.
- Unknown tags: default is error; set `Options.IgnoreUnknown = true` to skip them.
- Comments are ignored.
- Nesting and order are preserved.

**From XML to Code**
- Build layout once at init, then keep references and mutate:
  - `res, _ := xmlui.Build(ctx, file, reg, xmlui.Options{})`
  - `root := res.Root`
  - `title := res.ByID["title"].(*layout.PanelComponent)`
  - `title.SetText("Updated")`
  - Next draw: `LayoutAndDraw(ctx, root, canvas)`

**Error Handling**
- Parsing: returns descriptive errors on malformed XML or unexpected structure (e.g., stray end tags).
- Unknown elements: error unless `IgnoreUnknown` is true.
- Binding: `BindByID` ignores unknown ids and type mismatches; it doesn’t panic.

**Performance**
- Parse XML and build once; reuse `Root` across frames.
- Use `ByID`/binding to update text or toggle properties and call `SetDirty()` on the affected components.
- Avoid rebuilding the tree every frame.

**Example**
- XML:
  - `<VStack spacing="8">`
  - `  <Panel id="title" font-size="18" text="Hello"/>`
  - `  <HStack spacing="6" justify="spaceBetween">`
  - `    <Panel text="Left"/><Spacer/><Panel text="Right"/>`
  - `  </HStack>`
  - `</VStack>`
- Code:
  - `reg := xmlui.NewRegistry()`
  - `res, _ := xmlui.Build(ctx, reader, reg, xmlui.Options{})`
  - `var refs struct { Title *layout.PanelComponent ` + "`ui:\"title\"`" + ` }`
  - `_ = xmlui.BindByID(&refs, res)`
  - `refs.Title.SetText("World")`

**CSS Support (Simple Subset)**
- Stylesheets can be provided via `Options.Styles` using `xmlui.ParseStylesheet(io.Reader)`.
- File helpers: `xmlui.ParseStylesheetFile(path)` and `xmlui.ParseStylesheetFS(fsys, path)` resolve `@import` directives before parsing.
- Supported selectors:
  - Tag selectors: `Panel`, `VStack`, `HStack`, `ZStack`, `Grid`, `Spacer`, `Image`, and any custom tag names
  - Class selectors: `.class` (matches `class="class"` on the element). Supports multiple classes, e.g. `.row.space-between`
  - ID selectors: `#id` (matches `id="id"`)
  - Combined: `Text.title`, `HStack#main`, `.a.b` (all classes required). Comma‑separated lists supported.
- Supported properties → mapped to XML attributes:
  - `spacing` → `spacing`
  - `padding` → `padding` (same formats)
  - `justify` → `justify`
  - `align-h` → `align-h`
  - `align-v` → `align-v`
  - `font-size` → `font-size`
  - `font-autosize` → `font-autosize`
  - `font-autosize-min` → `font-autosize-min`
  - `font-autosize-max` → `font-autosize-max`
  - `color` → `color` (hex or CSS named colors)
  - `background` → `background`
  - `text` → `text`
  - `wrap` → `wrap`
  - `position` → `position` (`static`, `relative`, `absolute`, `fixed`)
  - `top` / `right` / `bottom` / `left` → corresponding offsets (dp)
  - `z-index` → `z-index`
  - `min-width` → `min-width`
  - `min-height` → `min-height`
  - `max-width` → `max-width`
  - `max-height` → `max-height`
  - `weight` → `weight`
  - `align-self` → `align-self`
  - `align-items` → `align-items`
  - `align-content` → `align-content`
  - `line-spacing` → `line-spacing`
- Precedence:
  - Explicit XML attributes (e.g., `font-size="18"`) take highest precedence and are never overridden by CSS.
  - Inline `style="..."` overrides stylesheet rules.
  - Stylesheet rules resolve by specificity (id > class > tag) and source order (later wins on ties).
- Positioning semantics:
  - `position: static` (default) keeps elements in normal stack/grid flow.
  - `position: relative` offsets the element visually after layout without affecting siblings.
  - `position: absolute` removes the element from flow and positions it relative to the nearest positioned ancestor (or the root).
  - `position: fixed` targets the viewport; offsets ignore scrolling containers.
- Offsets (`top`, `right`, `bottom`, `left`) are expressed in dp. When both opposing sides are provided they clamp the size to the remaining space in that axis. `z-index` controls draw order when elements overlap.
- Imports: file-based helpers resolve `@import "other.css";` in source order, so shared tokens can be reused across stylesheets.
- Example stylesheet:
  - `Text { font-size: 14dp; color: #EEEEEE; }`
  - `.title { font-size: 20dp; align-h: center; color: #FFFFFF; }`

**Linting XML/CSS**
- Use the bundled linter to catch unused or shadowed attributes: `go run ./cmd/xmlui-lint -xml examples/zoo/screens/layouts/scoreboard_demo.xml`
- Add extra stylesheets that are not imported from XML with repeatable `-css` flags: `go run ./cmd/xmlui-lint -xml layout.xml -css shared.css -css tokens.css`
- Findings include:
  - `unused-xml-attr`: attributes not recognized by the loader
  - `unused-css-property`: CSS props that map to no element (e.g., unused selectors)
  - `xml-shadows-css`: XML attribute overrides a CSS declaration on the same node
- Sample output:
  ```bash
  xml-shadows-css: VStack#viewport.viewport-root: XML sets "fill-height", CSS ".viewport-root" (examples/zoo/screens/layouts/viewport_column.css) also sets it so the CSS is ignored
  unused-css-property: CSS ".viewport-column" (examples/zoo/screens/layouts/viewport_column.css) sets "spacing" but it does not apply to any element
  ```
  - `.row.space-between { justify: spaceBetween; }`
  - `.panel { padding: 8dp; }`
  - `.para { color: #DDDDDD; max-width: 280dp; wrap: true; }`
  - `.badge { position: absolute; top: 12; right: 12; z-index: 5; }`
- Usage:
- `ss, _ := xmlui.ParseStylesheet(cssReader)` or `ss, _ := xmlui.ParseStylesheetFile("styles.css")`
  - `res, _ := xmlui.Build(ctx, xmlReader, reg, xmlui.Options{Styles: ss})`
- XML convenience: place `<?xml-stylesheet href="styles.css" type="text/css"?>` at the top of your XML file and call `xmlui.BuildFile(...)` or `xmlui.BuildFS(...)` to load XML + CSS together.
