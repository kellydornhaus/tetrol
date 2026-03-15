module tetrol

go 1.24.2

require (
	github.com/hajimehoshi/ebiten/v2 v2.9.4
	github.com/kellydornhaus/layouter v0.0.0-00010101000000-000000000000
	github.com/kellydornhaus/layouter/adapters/ebiten v0.0.0-00010101000000-000000000000
	github.com/kellydornhaus/layouter/adapters/etxt v0.0.0-00010101000000-000000000000
	golang.org/x/image v0.32.0
)

require (
	github.com/ebitengine/gomobile v0.0.0-20250923094054-ea854a63cce1 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/go-text/typesetting v0.3.0 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/tinne26/etxt v0.0.9 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

replace github.com/kellydornhaus/layouter => ./external/layouter

replace github.com/kellydornhaus/layouter/adapters/ebiten => ./external/layouter/adapters/ebiten

replace github.com/kellydornhaus/layouter/adapters/etxt => ./external/layouter/adapters/etxt
