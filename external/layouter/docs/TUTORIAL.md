**Quick Tutorial**
- Build a small “status dashboard” in both Go code and XML/CSS.
- Designed to run inside the examples/zoo harness or any ebiten project.

**1. Prerequisites**
- Go 1.21+.
- Ebiten + layouter repo checked out locally.
- For XML tutorial: familiarity with the registry pattern.

**2. Code-First Version**
1. **Create Styles**
   ```go
   var (
       titleStyle = layout.TextStyle{
           SizeDp: 26, AlignH: layout.AlignCenter,
           Color: layout.Color{R: 240, G: 240, B: 255, A: 255},
       }
       labelStyle = layout.TextStyle{
           SizeDp: 16, Color: layout.Color{R: 210, G: 220, B: 240, A: 255},
       }
   )
   ```
2. **Build Components**
   ```go
   func buildCodeDashboard() layout.Component {
       title := layout.NewLabel("Service Health", titleStyle)
       titlePanel := layout.NewPanelContainer(title, layout.Insets(16))
       titlePanel.SetBackgroundColor(layout.Color{R: 32, G: 44, B: 72, A: 255})
       titlePanel.SetFillWidth(true)

       makeStat := func(name, value string) layout.Component {
           line := layout.NewLabel(value, labelStyle)
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
       grid.Add(makeStat("Latency", "82 ms"))
       grid.Add(makeStat("Requests", "12.4k/min"))
       grid.Add(makeStat("Errors", "0.14%"))
       grid.Add(makeStat("Active Users", "3,980"))

       column := layout.NewVStack(titlePanel, grid)
       column.Spacing = 18
       column.SetFillWidth(true)

       column.SetBackgroundColor(layout.Color{R: 18, G: 22, B: 34, A: 255})
       return column
   }
   ```
3. **Render**
   ```go
   func (g *game) rebuildRoot() {
       g.root = buildCodeDashboard()
   }
   ```
   - Rebuild when data changes; update labels with `SetText`.

**3. XML + CSS Version**
1. **Create XML (`dashboard.xml`)**
```xml
<?xml-stylesheet href="dashboard.css" type="text/css"?>
<VStack class="page" spacing="18dp">
     <Panel id="title" class="title" text="Service Health"/>
     <Grid id="stats" columns="auto" spacing="12dp">
       <Panel class="stat" text="Latency 82 ms"/>
       <Panel class="stat" text="Requests 12.4k/min"/>
       <Panel class="stat" text="Errors 0.14%"/>
       <Panel class="stat" text="Active Users 3,980"/>
     </Grid>
   </VStack>
   ```
2. **Create CSS (`dashboard.css`)**
   ```
   .page { padding: 24dp; background: #121622; }
   .title { padding: 16dp; font-size: 26dp; align-h: center; background: #202c48; color: #eef2ff; min-width: 280dp; }
   .stat { padding: 14dp; background: #1a2238; color: #d2d8e6; font-size: 18dp; }
   ```
3. **Load in Go**
```go
func buildXMLDashboard(ctx *layout.Context) (layout.Component, xmlui.Result, error) {
    res, err := xmlui.BuildFile(ctx, "dashboard.xml", nil, xmlui.Options{})
    return res.Root, res, err
}
```
- `xmlui.BuildFile` resolves the `<?xml-stylesheet ...?>` processing instruction and any `@import` statements inside referenced CSS files. For custom sources (embed.FS, zip, etc.), use `xmlui.BuildFS`.
4. **Mutate at Runtime**
```go
root, res, _ := buildXMLDashboard(ctx)
stats := res.ByID["stats"].(*layout.Grid)
stats.Add(layout.NewPanelContainer(layout.NewLabel("Datacenters 6", labelStyle), layout.Insets(14)))

   title := res.ByID["title"].(*layout.PanelComponent)
   title.SetText("Service Health — Updated")
   ```

**4. Running in the Zoo**
- Drop the build function into `examples/zoo/screens`.
- Add to `NewScreens` slice to expose it in the demo carousel.
- For XML example, place files in `examples/zoo/screens/layouts`.
- Ebiten window resizing is already enabled.

**5. Experiment: Live Editing**
- Enable hot reload similar to `xml_demo`:
  ```go
  result, err := xmlui.Build(ctx, bytes.NewReader(xmlBytes), registry, xmlui.Options{Styles: stylesheet})
  // store timestamps + re-run Build when files change
  ```
- Reassign new root to scroll/overlay container and call `SetDirty()`.

**6. Next Steps**
- Mix code and XML: build a base view in XML, then compose additional Go components around it.
- Create reusable registries for repeated widgets (buttons, cards, etc.).
- Extend CSS parser with project-specific properties (e.g., icon references).
- Share styles across screens via a global stylesheet.
