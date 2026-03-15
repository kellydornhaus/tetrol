package main

import "math/rand"

type TileState int

const (
	TileIdle TileState = iota
	TileFalling
	TileClearing
	TileSwapping
	TileMatched // part of a detected word, shown with outline
)

const (
	ClearAnimTicks = 30
	SwapAnimTicks  = 6
)

type Tile struct {
	Letter     rune
	State      TileState
	AnimTick   int
	FallOffset float64 // negative = tile is above its grid position, animates toward 0
	SwapOffset   float64 // offset in tile units during swap animation (horizontal or vertical)
	SwapDir      int     // direction tile moved: +1 = right/down, -1 = left/up
	SwapVertical bool    // true if this is a vertical swap
}

// Letter frequency weights from wordpeek/scramble-quest (excluding Q, Z, X, J)
var letterWeights = []struct {
	Letter rune
	Weight int
}{
	{'A', 79}, {'B', 20}, {'C', 40}, {'D', 38}, {'E', 110},
	{'F', 14}, {'G', 30}, {'H', 23}, {'I', 86}, {'K', 10},
	{'L', 53}, {'M', 27}, {'N', 72}, {'O', 62}, {'P', 28},
	{'R', 73}, {'S', 87}, {'T', 67}, {'U', 33}, {'V', 10},
	{'W', 10}, {'Y', 16},
}

var totalWeight int

func init() {
	for _, lw := range letterWeights {
		totalWeight += lw.Weight
	}
}

func RandomLetter() rune {
	r := rand.Intn(totalWeight)
	for _, lw := range letterWeights {
		r -= lw.Weight
		if r < 0 {
			return lw.Letter
		}
	}
	return 'E'
}

func NewTile(letter rune) *Tile {
	return &Tile{Letter: letter, State: TileIdle}
}
