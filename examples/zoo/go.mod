module github.com/kellydornhaus/layouter/examples/zoo

go 1.24.0

toolchain go1.24.6

replace github.com/kellydornhaus/layouter => ../..

replace github.com/kellydornhaus/layouter/adapters/ebiten => ../../adapters/ebiten

replace github.com/kellydornhaus/layouter/adapters/etxt => ../../adapters/etxt

require (
	github.com/hajimehoshi/ebiten/v2 v2.8.8
	github.com/kellydornhaus/layouter v0.0.0-00010101000000-000000000000
	github.com/kellydornhaus/layouter/adapters/ebiten v0.0.0-00010101000000-000000000000
	github.com/kellydornhaus/layouter/adapters/etxt v0.0.0-00010101000000-000000000000
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20210519020934-456a8d69b780
)

require (
	github.com/ebitengine/gomobile v0.0.0-20240911145611-4856209ac325 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.8.0 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/tinne26/etxt v0.0.9 // indirect
	golang.org/x/image v0.31.0 // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
