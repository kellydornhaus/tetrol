Run the component zoo example (requires Go 1.24+):

- `cd examples/zoo`
- `go run .`
- `go run . -headless [-out screenshots] [-width 1024] [-height 768]` to export cold/warm PNG pairs and a timing summary (frame/layout/render + hashes) for every screen without opening the interactive window.

Keys:
- `←`/`→` (or `↑`/`↓`) to cycle screens
- Window resize is handled; layout uses dp and monitor scale

Notes:
- The example implements adapters for Ebiten rendering and etxt text.
- The core module (../../layout) remains dependency-free.
- Labs: Text basics, Stacks/Spacers, Alignment, Nesting, Dirty-state, Justify gallery, Spacing/Align gallery, Wrap gallery, Grid, Outlines, XML & CSS demo.
